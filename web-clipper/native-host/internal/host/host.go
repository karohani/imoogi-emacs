// Package host implements the native messaging request loop.
package host

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/karohani/obsidian-clipper/native-host/internal/config"
	"github.com/karohani/obsidian-clipper/native-host/internal/fetch"
	"github.com/karohani/obsidian-clipper/native-host/internal/orgrender"
	"github.com/karohani/obsidian-clipper/native-host/internal/protocol"
	"github.com/karohani/obsidian-clipper/native-host/internal/store"
)

// TmpMaxAge is the age past which staging leftovers are cleaned at start.
const TmpMaxAge = 24 * time.Hour

// Fetcher downloads assets; *fetch.Downloader in production.
type Fetcher interface {
	FetchAll(ctx context.Context, assets []protocol.Asset) []fetch.Result
}

// Handler serves one Chrome port. Messages are handled sequentially.
type Handler struct {
	// LoadConfig returns the current configuration (config.Load in production).
	LoadConfig func() (*config.Config, error)
	Fetcher    Fetcher
	Log        *Logger
	Version    string
}

// Serve reads framed messages from r and writes responses to w until EOF.
// An oversize length prefix gets a too_large result and ends the session,
// because the unread body makes the stream unrecoverable.
//
// @MX:ANCHOR: [AUTO] the protocol loop: the only writer to stdout in host mode
// @MX:REASON: every response must go through send so the 64 KiB cap and framing hold; a stray write corrupts the stream
func (h *Handler) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	h.cleanTmp()
	send := func(v any) error {
		b, err := protocol.EncodeResponse(v)
		if err != nil {
			return err
		}
		return protocol.WriteMessage(w, b)
	}
	for {
		body, err := protocol.ReadMessage(r)
		if err == io.EOF {
			return nil
		}
		if errors.Is(err, protocol.ErrMessageTooLarge) {
			h.Log.Log(Entry{Type: "frame", Status: protocol.StatusFailed, ErrorCode: protocol.CodeTooLarge})
			send(protocol.FailedResult("", protocol.CodeTooLarge, "message exceeds the 8 MiB limit"))
			return err
		}
		if err != nil {
			return err
		}
		req, err := protocol.Parse(body)
		if err != nil {
			var pe *protocol.Error
			errors.As(err, &pe)
			h.Log.Log(Entry{Type: "invalid", RequestID: pe.RequestID, Status: protocol.StatusFailed, ErrorCode: pe.Code})
			if err := send(protocol.FailedResult(pe.RequestID, pe.Code, pe.Message)); err != nil {
				return err
			}
			continue
		}
		var resp any
		switch req.Type {
		case protocol.TypePing:
			cfg, _ := h.config()
			resp = protocol.Pong{Type: "pong", SchemaVersion: protocol.SchemaVersion, Configured: cfg != nil, Version: h.Version}
		case protocol.TypeStatus:
			resp = h.status(req.RequestID)
		case protocol.TypeSaveClip:
			if err := send(protocol.Ack{Type: "ack", RequestID: req.RequestID}); err != nil {
				return err
			}
			res := h.save(ctx, req.Save)
			h.Log.Log(Entry{Type: req.Type, RequestID: req.RequestID, Category: req.Save.Category, Status: res.Status,
				AssetsSaved: res.AssetsSaved, AssetsFail: len(res.FailedAssets), ErrorCode: errCode(res), SourceURL: req.Save.SourceURL})
			resp = res
		}
		if err := send(resp); err != nil {
			return err
		}
	}
}

func errCode(r protocol.Result) string {
	if r.Error != nil {
		return r.Error.Code
	}
	return ""
}

// config returns a usable configuration or nil.
func (h *Handler) config() (*config.Config, error) {
	c, err := h.LoadConfig()
	if err != nil || c == nil || c.Root == "" {
		if err == nil {
			err = config.ErrNotConfigured
		}
		return nil, err
	}
	return c, nil
}

func (h *Handler) cleanTmp() {
	cfg, _ := h.config()
	if cfg == nil {
		return
	}
	if st, err := store.Open(cfg.Root); err == nil {
		st.CleanTmp(TmpMaxAge)
		st.Close()
	}
}

func failed(id, code string) protocol.Result {
	return protocol.FailedResult(id, code, store.Message(code))
}

func (h *Handler) openStore(id string) (*store.Store, *protocol.Result) {
	cfg, _ := h.config()
	if cfg == nil {
		r := failed(id, protocol.CodeNotConfigured)
		return nil, &r
	}
	st, err := store.Open(cfg.Root)
	if err != nil {
		r := failed(id, protocol.CodeRootUnavailable)
		return nil, &r
	}
	return st, nil
}

func (h *Handler) status(id string) protocol.Result {
	st, fail := h.openStore(id)
	if fail != nil {
		return *fail
	}
	defer st.Close()
	if prev, err := st.Lookup(id); err == nil && prev != nil {
		prev.Duplicate = true
		return *prev
	}
	if st.InFlight(id) {
		return protocol.Result{Type: "result", RequestID: id, Status: protocol.StatusInProgress}
	}
	return protocol.Result{Type: "result", RequestID: id, Status: protocol.StatusUnknown}
}

// save performs one saveClip. Asset failures never fail the clip; any
// storage failure fails it and leaves nothing behind but the staging
// directory, which is removed.
func (h *Handler) save(ctx context.Context, req *protocol.SaveClip) protocol.Result {
	id := req.RequestID
	st, fail := h.openStore(id)
	if fail != nil {
		return *fail
	}
	defer st.Close()
	if prev, err := st.Lookup(id); err == nil && prev != nil {
		prev.Duplicate = true
		return *prev
	}
	release, busy, err := st.Acquire(id)
	if err != nil {
		return failed(id, store.Classify(err))
	}
	if busy {
		return protocol.Result{Type: "result", RequestID: id, Status: protocol.StatusInProgress}
	}
	defer release()
	// Another process may have finished between the first lookup and the lock.
	if prev, err := st.Lookup(id); err == nil && prev != nil {
		prev.Duplicate = true
		return *prev
	}
	stg, err := st.Stage(id)
	if err != nil {
		return failed(id, store.Classify(err))
	}
	defer stg.Discard() // no-op once committed

	saved := map[string]string{}
	var failedAssets []protocol.FailedAsset
	downloadFailed := false
	if req.DownloadAssets {
		for _, r := range h.Fetcher.FetchAll(ctx, req.Assets) {
			if r.Reason != "" {
				failedAssets = append(failedAssets, protocol.FailedAsset{ID: r.ID, Reason: r.Reason})
				downloadFailed = true
				continue
			}
			name := fetch.AssetName(r.Data, r.Ext)
			if err := stg.WriteAsset(name, r.Data); err != nil {
				return failed(id, store.Classify(err))
			}
			saved[r.ID] = name
		}
	} else {
		for _, a := range req.Assets {
			failedAssets = append(failedAssets, protocol.FailedAsset{ID: a.ID, Reason: protocol.ReasonSkipped})
		}
	}

	folder, _ := protocol.CategoryFolder(req.Category)
	name, err := st.FreeName(folder, store.DocBaseName(req.ClippedAt, req.Title))
	if err != nil {
		return failed(id, store.Classify(err))
	}
	doc := orgrender.Render(orgrender.Doc{Title: req.Title, SourceURL: req.SourceURL, ClippedAt: req.ClippedAt,
		Blocks: req.Blocks, Assets: saved, AssetDir: "assets/" + name})
	if err := stg.WriteDoc(doc); err != nil {
		return failed(id, store.Classify(err))
	}
	rel, err := stg.Commit(folder, name)
	if errors.Is(err, store.ErrExists) {
		return protocol.FailedResult(id, protocol.CodeInternal, "a clip with this name already exists")
	}
	if err != nil {
		return failed(id, store.Classify(err))
	}
	res := protocol.Result{Type: "result", RequestID: id, Status: protocol.StatusSaved, Path: rel,
		AssetsSaved: len(saved), FailedAssets: failedAssets}
	if downloadFailed {
		res.Status = protocol.StatusPartial
	}
	if res.FailedAssets == nil {
		res.FailedAssets = []protocol.FailedAsset{}
	}
	if err := st.WriteLedger(res); err != nil {
		// The clip is in place; only retry de-duplication is lost. Report
		// the truth about the files rather than a failure.
		h.Log.Log(Entry{Type: "ledger", RequestID: id, Status: res.Status, ErrorCode: store.Classify(err)})
	}
	return res
}
