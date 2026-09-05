package planner

import (
	"context"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/hashing"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

// AC-C-003a — the sync hot path never writes a note type, whatever shape the
// run takes.
//
// This is the SPEC's standing regression fence (plan.md milestone M1). It is
// written before any note-type write exists anywhere in the codebase, so it
// passes trivially today and keeps passing for the rest of the SPEC: the
// moment a later milestone's install code is reached from the sync path
// rather than from the install path, this test fails.
//
// The three logs asserted here are the three model-write endpoints the
// AnkiConnector interface actually carries after M1's five additions. The
// interface exposes no scheduling, review-history, or collection-styling
// endpoint at all, so a count assertion over those would assert nothing —
// their continued absence is a separate property (AC-C-022b), not a count.
func TestRun_AC_C_003a_SyncRunOfAnyShapeRecordsZeroModelWrites(t *testing.T) {
	tests := []struct {
		name string
		// setup returns the request for a run of one particular shape, having
		// seeded the registry and the stub collection as that shape requires.
		setup func(t *testing.T, reg *registry.Registry, client *fakeClient) protocol.Request
		// wantAction, when non-empty, is the result action the shape must
		// actually produce — the discipline that keeps this fence honest: a
		// shape that silently degraded to a no-op would still record zero
		// model writes and would pass while asserting nothing.
		wantAction string
	}{
		{
			name: "add — a new entry with no note identifier",
			setup: func(t *testing.T, reg *registry.Registry, client *fakeClient) protocol.Request {
				return protocol.Request{
					Config: baseConfig(),
					Entries: []protocol.Entry{
						{Key: "a.org::0", NoteType: "Basic", SourcePath: "a.org", Title: "T", Body: "B"},
					},
				}
			},
			wantAction: protocol.ActionAdded,
		},
		{
			name: "update — an entry whose content changed since it was recorded",
			setup: func(t *testing.T, reg *registry.Registry, client *fakeClient) protocol.Request {
				reg.Put(registry.Entry{
					NoteID: 2001, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "Inbox",
					ContentHash: "a-stale-hash-so-the-entry-reads-as-changed",
				})
				client.seedNote(2001, "Basic", map[string]string{"Front": "old", "Back": "old"}, nil, []int{92001})
				id := 2001
				return protocol.Request{
					Config: baseConfig(),
					Entries: []protocol.Entry{
						{Key: "a.org::0", NoteID: &id, NoteType: "Basic", SourcePath: "a.org", Title: "New title", Body: "New body"},
					},
				}
			},
			wantAction: protocol.ActionUpdated,
		},
		{
			name: "deck move — the same content resolved into a different deck",
			setup: func(t *testing.T, reg *registry.Registry, client *fakeClient) protocol.Request {
				fields := map[string]string{"Front": "F", "Back": "B"}
				reg.Put(registry.Entry{
					NoteID: 2002, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "Inbox",
					ContentHash: hashing.Hash("Basic", fields, "Inbox", nil),
				})
				client.seedNote(2002, "Basic", fields, nil, []int{92002})
				client.decks["Inbox"] = true
				client.decks["Moved"] = true
				id := 2002
				deck := "Moved"
				return protocol.Request{
					Config: baseConfig(),
					Entries: []protocol.Entry{
						{Key: "a.org::0", NoteID: &id, NoteType: "Basic", SourcePath: "a.org", Deck: &deck, Title: "T", Body: "B"},
					},
				}
			},
		},
		{
			name: "delete — an orphaned registry entry whose heading is gone",
			setup: func(t *testing.T, reg *registry.Registry, client *fakeClient) protocol.Request {
				fields := map[string]string{"Front": "F", "Back": "B"}
				reg.Put(registry.Entry{
					NoteID: 2003, SourcePath: "gone.org", NoteType: "Basic", ResolvedDeck: "Inbox",
					ContentHash: hashing.Hash("Basic", fields, "Inbox", nil),
				})
				client.seedNote(2003, "Basic", fields, nil, []int{92003})
				return protocol.Request{
					Config:  baseConfig(),
					Census:  []protocol.CensusEntry{},
					Entries: []protocol.Entry{},
				}
			},
			wantAction: protocol.ActionDeleted,
		},
		{
			name: "no-op — an entry already matching what was recorded",
			setup: func(t *testing.T, reg *registry.Registry, client *fakeClient) protocol.Request {
				hash := renderAndHash(t, "Basic", "T", "B", "Inbox", nil)
				reg.Put(registry.Entry{
					NoteID: 2004, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "Inbox",
					ContentHash: hash,
				})
				client.seedNote(2004, "Basic", map[string]string{"Front": "F", "Back": "B"}, nil, []int{92004})
				client.decks["Inbox"] = true
				id := 2004
				return protocol.Request{
					Config: baseConfig(),
					Entries: []protocol.Entry{
						{Key: "a.org::0", NoteID: &id, NoteType: "Basic", SourcePath: "a.org", Title: "T", Body: "B"},
					},
				}
			},
			wantAction: protocol.ActionSkipped,
		},
		{
			name: "empty — a run with nothing to do at all",
			setup: func(t *testing.T, reg *registry.Registry, client *fakeClient) protocol.Request {
				return protocol.Request{Config: baseConfig(), Entries: []protocol.Entry{}}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := newTestRegistry(t)
			client := newFakeClient()
			req := tt.setup(t, reg, client)

			results, _ := Run(context.Background(), req, reg, client)

			if tt.wantAction != "" {
				var seen bool
				for _, r := range results {
					if r.Action == tt.wantAction {
						seen = true
						break
					}
				}
				if !seen {
					t.Fatalf("no result with action %q — this shape did not actually exercise the path it names; results = %+v", tt.wantAction, results)
				}
			}

			if n := len(client.createModelCalls); n != 0 {
				t.Errorf("createModelCalls = %d, want 0 — a sync run must never create a note type", n)
			}
			if n := len(client.updateModelStylingCalls); n != 0 {
				t.Errorf("updateModelStylingCalls = %d, want 0 — a sync run must never restyle a note type", n)
			}
			if n := len(client.updateModelTemplatesCalls); n != 0 {
				t.Errorf("updateModelTemplatesCalls = %d, want 0 — a sync run must never rewrite a note type's templates", n)
			}
		})
	}
}

// The sync path must not probe the note-type list or push media either. These
// two are not part of AC-C-003a's three-log assertion, but the same M1
// property covers them: no synchronization path calls any of the five new
// methods, so all five logs stay empty until a later milestone's install and
// media paths drive them from their own entry points.
func TestRun_SyncRunCallsNeitherModelNamesNorStoreMediaFile(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{Key: "a.org::0", NoteType: "Basic", SourcePath: "a.org", Title: "T", Body: "B"},
		},
	}

	if _, errs := Run(context.Background(), req, reg, client); len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}

	if n := client.modelNamesCalls; n != 0 {
		t.Errorf("modelNamesCalls = %d, want 0 — the sync path does not probe note types", n)
	}
	if n := len(client.storeMediaFileCalls); n != 0 {
		t.Errorf("storeMediaFileCalls = %d, want 0 — the sync path uploads no media in M1", n)
	}
}
