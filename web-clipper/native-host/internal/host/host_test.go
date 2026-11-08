package host

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/karohani/obsidian-clipper/native-host/internal/config"
	"github.com/karohani/obsidian-clipper/native-host/internal/fetch"
	"github.com/karohani/obsidian-clipper/native-host/internal/protocol"
)

const rid = "8c3f0f8e-1f8a-4c55-9a59-0e5a3f8f9d21"

func frame(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var hdr [4]byte
	binary.LittleEndian.PutUint32(hdr[:], uint32(len(b)))
	return append(hdr[:], b...)
}

func readAll(t *testing.T, out []byte) []map[string]any {
	t.Helper()
	r := bytes.NewReader(out)
	var msgs []map[string]any
	for {
		b, err := protocol.ReadMessage(r)
		if err == io.EOF {
			return msgs
		}
		if err != nil {
			t.Fatalf("bad output framing: %v", err)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("bad output json %s: %v", b, err)
		}
		msgs = append(msgs, m)
	}
}

func imageServer(t *testing.T) (*httptest.Server, fetch.Options) {
	t.Helper()
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	img := b.Bytes()
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(img)
	}))
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.StartTLS()
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	opts := fetch.DefaultOptions()
	opts.CheckIP = func(a netip.Addr) bool { return a == netip.MustParseAddr("127.0.0.1") }
	opts.RootCAs = srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs
	opts.Port = u.Port()
	return srv, opts
}

func saveMsg(base string, mut func(map[string]any)) map[string]any {
	m := map[string]any{
		"schemaVersion": 1, "type": "saveClip", "requestId": rid, "category": "clippings",
		"title":     "원문 제목 #+SETUPFILE: x",
		"sourceUrl": "https://news.example.com/post?id=secret-query",
		"clippedAt": "2026-11-08T10:00:00+09:00", "downloadAssets": true,
		"blocks": []any{
			map[string]any{"t": "para", "inl": []any{map[string]any{"t": "text", "v": "* TODO body text"}}},
			map[string]any{"t": "image", "asset": "a1"},
			map[string]any{"t": "image", "asset": "a2"},
			map[string]any{"t": "image", "asset": "a3"},
		},
		"assets": []any{
			map[string]any{"id": "a1", "url": base + "/one.png"},
			map[string]any{"id": "a2", "url": "http://insecure.example.com/x.png"},
			map[string]any{"id": "a3", "url": base + "/same-bytes.png"},
		},
	}
	if mut != nil {
		mut(m)
	}
	return m
}

func newHandler(root string, f Fetcher, logw io.Writer) *Handler {
	return &Handler{
		LoadConfig: func() (*config.Config, error) { return &config.Config{Root: root}, nil },
		Fetcher:    f,
		Log:        NewLogger(logw),
		Version:    "9.9.9",
	}
}

func TestEndToEnd(t *testing.T) {
	srv, opts := imageServer(t)
	root := t.TempDir()
	var logBuf bytes.Buffer
	h := newHandler(root, fetch.New(opts), &logBuf)

	var in bytes.Buffer
	in.Write(frame(t, saveMsg(srv.URL, nil)))
	in.Write(frame(t, saveMsg(srv.URL, nil))) // retry with the same requestId
	in.Write(frame(t, map[string]any{"schemaVersion": 1, "type": "status", "requestId": rid}))
	in.Write(frame(t, map[string]any{"schemaVersion": 1, "type": "ping"}))
	var out bytes.Buffer
	if err := h.Serve(context.Background(), &in, &out); err != nil {
		t.Fatal(err)
	}
	msgs := readAll(t, out.Bytes())
	if len(msgs) != 6 {
		t.Fatalf("got %d messages: %v", len(msgs), msgs)
	}
	if msgs[0]["type"] != "ack" || msgs[0]["requestId"] != rid {
		t.Fatalf("first message not ack: %v", msgs[0])
	}
	res := msgs[1]
	wantPath := "00. Inbox/04. Clippings/2026-11-08-원문-제목-setupfile-x.org"
	if res["type"] != "result" || res["status"] != "partial" || res["path"] != wantPath ||
		res["assetsSaved"] != float64(2) || res["duplicate"] != false {
		t.Fatalf("result %v", res)
	}
	fa := res["failedAssets"].([]any)
	if len(fa) != 1 || fa[0].(map[string]any)["id"] != "a2" || fa[0].(map[string]any)["reason"] != "not_https" {
		t.Fatalf("failedAssets %v", fa)
	}
	if msgs[2]["type"] != "ack" || msgs[3]["duplicate"] != true || msgs[3]["path"] != wantPath {
		t.Fatalf("retry not replayed: %v %v", msgs[2], msgs[3])
	}
	if msgs[4]["status"] != "partial" || msgs[4]["duplicate"] != true {
		t.Fatalf("status %v", msgs[4])
	}
	if msgs[5]["type"] != "pong" || msgs[5]["configured"] != true || msgs[5]["version"] != "9.9.9" {
		t.Fatalf("pong %v", msgs[5])
	}
	for _, m := range msgs {
		for _, v := range m {
			if s, ok := v.(string); ok && strings.Contains(s, root) {
				t.Fatalf("root path disclosed: %v", m)
			}
		}
	}

	// Files on disk.
	catDir := filepath.Join(root, "00. Inbox", "04. Clippings")
	entries, _ := os.ReadDir(catDir)
	if len(entries) != 2 { // the document and assets/
		t.Fatalf("want the document and assets/, got %v", entries)
	}
	clip := filepath.Join(root, filepath.FromSlash(wantPath))
	org, err := os.ReadFile(clip)
	if err != nil {
		t.Fatal(err)
	}
	assets, _ := os.ReadDir(filepath.Join(catDir, "assets", "2026-11-08-원문-제목-setupfile-x"))
	if len(assets) != 1 { // a1 and a3 have identical bytes -> one file
		t.Fatalf("assets %v", assets)
	}
	name := assets[0].Name()
	s := string(org)
	if strings.Count(s, "[[file:assets/2026-11-08-원문-제목-setupfile-x/"+name+"]]") != 2 || strings.Count(s, "(image not saved)") != 1 {
		t.Fatalf("doc:\n%s", s)
	}
	if strings.Contains(s, "\n* TODO") || strings.Contains(s, "\n#+SETUPFILE") {
		t.Fatalf("unescaped content:\n%s", s)
	}
	if tmp, _ := os.ReadDir(filepath.Join(root, ".org-clipper", "tmp")); len(tmp) != 0 {
		t.Fatalf("staging leftovers %v", tmp)
	}
	reqs, _ := os.ReadDir(filepath.Join(root, ".org-clipper", "requests"))
	if len(reqs) != 1 || reqs[0].Name() != rid+".json" {
		t.Fatalf("requests dir %v (lock must be gone)", reqs)
	}

	// Logs carry the host only.
	logs := logBuf.String()
	if !strings.Contains(logs, "host=news.example.com") || !strings.Contains(logs, "status=partial") {
		t.Fatalf("log missing fields: %s", logs)
	}
	for _, leak := range []string{"secret-query", "원문", "SETUPFILE", "TODO", "insecure", "/post"} {
		if strings.Contains(logs, leak) {
			t.Fatalf("log leaks %q: %s", leak, logs)
		}
	}
}

func TestDownloadAssetsFalse(t *testing.T) {
	root := t.TempDir()
	h := newHandler(root, nil, io.Discard) // a nil Fetcher proves nothing is fetched
	var in, out bytes.Buffer
	in.Write(frame(t, saveMsg("https://cdn.example.com", func(m map[string]any) { m["downloadAssets"] = false })))
	if err := h.Serve(context.Background(), &in, &out); err != nil {
		t.Fatal(err)
	}
	msgs := readAll(t, out.Bytes())
	res := msgs[1]
	if res["status"] != "saved" || res["assetsSaved"] != float64(0) || len(res["failedAssets"].([]any)) != 3 {
		t.Fatalf("%v", res)
	}
	if res["failedAssets"].([]any)[0].(map[string]any)["reason"] != "skipped" {
		t.Fatalf("%v", res)
	}
	b, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(res["path"].(string))))
	if strings.Contains(string(b), "file:") || strings.Count(string(b), "(image not saved)") != 3 {
		t.Fatalf("doc:\n%s", b)
	}
}

func TestNotConfiguredAndRootMissing(t *testing.T) {
	h := &Handler{LoadConfig: func() (*config.Config, error) { return &config.Config{}, config.ErrNotConfigured }, Log: NewLogger(io.Discard)}
	var in, out bytes.Buffer
	in.Write(frame(t, map[string]any{"schemaVersion": 1, "type": "ping"}))
	in.Write(frame(t, saveMsg("https://x.example", nil)))
	in.Write(frame(t, map[string]any{"schemaVersion": 1, "type": "status", "requestId": rid}))
	h.Serve(context.Background(), &in, &out)
	msgs := readAll(t, out.Bytes())
	if msgs[0]["configured"] != false {
		t.Fatalf("pong %v", msgs[0])
	}
	for _, m := range []map[string]any{msgs[2], msgs[3]} {
		if m["status"] != "failed" || m["error"].(map[string]any)["code"] != "not_configured" {
			t.Fatalf("%v", m)
		}
	}

	h = newHandler(filepath.Join(t.TempDir(), "gone"), nil, io.Discard)
	in.Reset()
	out.Reset()
	in.Write(frame(t, saveMsg("https://x.example", nil)))
	h.Serve(context.Background(), &in, &out)
	msgs = readAll(t, out.Bytes())
	if msgs[1]["error"].(map[string]any)["code"] != "root_unavailable" {
		t.Fatalf("%v", msgs[1])
	}
}

func TestInvalidAndOversize(t *testing.T) {
	h := newHandler(t.TempDir(), nil, io.Discard)
	var in, out bytes.Buffer
	in.Write(frame(t, map[string]any{"schemaVersion": 1, "type": "saveClip", "requestId": rid}))
	in.Write(frame(t, map[string]any{"schemaVersion": 1, "type": "ping"}))
	var hdr [4]byte
	binary.LittleEndian.PutUint32(hdr[:], protocol.MaxIncoming+1)
	in.Write(hdr[:])
	in.Write(frame(t, map[string]any{"schemaVersion": 1, "type": "ping"})) // never reached
	err := h.Serve(context.Background(), &in, &out)
	if !errors.Is(err, protocol.ErrMessageTooLarge) {
		t.Fatalf("want ErrMessageTooLarge, got %v", err)
	}
	msgs := readAll(t, out.Bytes())
	if len(msgs) != 3 {
		t.Fatalf("%v", msgs)
	}
	if msgs[0]["error"].(map[string]any)["code"] != "invalid_request" || msgs[0]["requestId"] != rid {
		t.Fatalf("%v", msgs[0])
	}
	if msgs[1]["type"] != "pong" || msgs[2]["error"].(map[string]any)["code"] != "too_large" {
		t.Fatalf("%v", msgs)
	}
}

func TestInProgressAndStaleLock(t *testing.T) {
	root := t.TempDir()
	reqDir := filepath.Join(root, ".org-clipper", "requests")
	os.MkdirAll(reqDir, 0o700)
	lock := filepath.Join(reqDir, rid+".lock")
	os.WriteFile(lock, nil, 0o600)
	h := newHandler(root, nil, io.Discard)
	msg := saveMsg("https://x.example", func(m map[string]any) { m["downloadAssets"] = false })
	var in, out bytes.Buffer
	in.Write(frame(t, msg))
	in.Write(frame(t, map[string]any{"schemaVersion": 1, "type": "status", "requestId": rid}))
	h.Serve(context.Background(), &in, &out)
	msgs := readAll(t, out.Bytes())
	if msgs[1]["status"] != "in_progress" || msgs[2]["status"] != "in_progress" {
		t.Fatalf("%v", msgs)
	}
	if _, err := os.Stat(lock); err != nil {
		t.Fatal("foreign lock removed")
	}
	old := time.Now().Add(-time.Hour)
	os.Chtimes(lock, old, old)
	in.Reset()
	out.Reset()
	in.Write(frame(t, msg))
	h.Serve(context.Background(), &in, &out)
	if msgs = readAll(t, out.Bytes()); msgs[1]["status"] != "saved" {
		t.Fatalf("stale lock not taken over: %v", msgs)
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Fatal("lock left behind")
	}
}

func TestStatusUnknownAndTmpCleanup(t *testing.T) {
	root := t.TempDir()
	stale := filepath.Join(root, ".org-clipper", "tmp", rid+"-deadbeef")
	os.MkdirAll(stale, 0o700)
	old := time.Now().Add(-48 * time.Hour)
	os.Chtimes(stale, old, old)
	h := newHandler(root, nil, io.Discard)
	var in, out bytes.Buffer
	in.Write(frame(t, map[string]any{"schemaVersion": 1, "type": "status", "requestId": rid}))
	h.Serve(context.Background(), &in, &out)
	if msgs := readAll(t, out.Bytes()); msgs[0]["status"] != "unknown" {
		t.Fatalf("%v", msgs)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("stale staging not cleaned at start")
	}
}

func TestExistingNameGetsSuffix(t *testing.T) {
	root := t.TempDir()
	cat := filepath.Join(root, "00. Inbox", "04. Clippings")
	os.MkdirAll(cat, 0o700)
	existing := filepath.Join(cat, "2026-11-08-원문-제목-setupfile-x.org")
	os.WriteFile(existing, []byte("mine"), 0o600)
	h := newHandler(root, nil, io.Discard)
	var in, out bytes.Buffer
	in.Write(frame(t, saveMsg("https://x.example", func(m map[string]any) { m["downloadAssets"] = false })))
	h.Serve(context.Background(), &in, &out)
	msgs := readAll(t, out.Bytes())
	if msgs[1]["status"] != "saved" || msgs[1]["path"] != "00. Inbox/04. Clippings/2026-11-08-원문-제목-setupfile-x-2.org" {
		t.Fatalf("%v", msgs[1])
	}
	if b, _ := os.ReadFile(existing); string(b) != "mine" {
		t.Fatal("existing clip overwritten")
	}
	if tmp, _ := os.ReadDir(filepath.Join(root, ".org-clipper", "tmp")); len(tmp) != 0 {
		t.Fatalf("staging leftovers %v", tmp)
	}
	if _, err := os.Lstat(filepath.Join(cat, "assets")); !os.IsNotExist(err) {
		t.Fatal("assets/ created for a clip without saved images")
	}
}

func TestLogTokenSanitising(t *testing.T) {
	var b bytes.Buffer
	NewLogger(&b).Log(Entry{RequestID: "x\nforged=1", SourceURL: "https://evil‮.example.com:8443/p?q=1", Status: "saved"})
	line := b.String()
	if strings.Count(line, "\n") != 1 || strings.Contains(line, "forged=1 ") || strings.Contains(line, "8443") || strings.Contains(line, "q=1") {
		t.Fatalf("%q", line)
	}
	logger, closer := OpenLog(filepath.Join(t.TempDir(), "logs"))
	logger.Log(Entry{Status: "saved"})
	closer.Close()
}
