package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPersistentLoggerWritesPrivateJSONLWithoutCardContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "anki.jsonl")
	t.Setenv(logPathEnvironment, path)
	logger := openPersistentLogger()
	logger.Event("sync_result", map[string]any{
		"key":       "cards.org::0",
		"note_type": "imoogi-Basic",
		"action":    "added",
	})
	logger.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("log mode = %o, want 600", got)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var record map[string]any
	if err := json.NewDecoder(bufio.NewReader(file)).Decode(&record); err != nil {
		t.Fatal(err)
	}
	if record["event"] != "sync_result" || record["action"] != "added" || record["note_type"] != "imoogi-Basic" {
		t.Fatalf("unexpected record: %#v", record)
	}
	if record["timestamp"] == nil {
		t.Fatalf("timestamp missing: %#v", record)
	}
}

func TestPersistentLoggerIsDisabledWithoutEnvironment(t *testing.T) {
	t.Setenv(logPathEnvironment, "")
	logger := openPersistentLogger()
	logger.Event("ignored", map[string]any{"body": "must not be persisted"})
	logger.Close()
}

func TestSyncLogRecordsImoogiBasicEntryWithoutItsContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "anki.jsonl")
	t.Setenv(logPathEnvironment, path)
	request := `{
  "protocol_version": 2,
  "config": {
    "default_deck": "Default",
    "anki_connect_url": "http://127.0.0.1:1",
    "registry_path": "/tmp/unused-registry.json",
    "sync_root": "/tmp/notes",
    "exclude_patterns": [],
    "scan_complete": false
  },
  "census": [],
  "entries": [{
    "key": "cards.org::0",
    "note_id": null,
    "note_type": "imoogi-Basic",
    "source_path": "cards.org",
    "deck": null,
    "tags": [],
    "title": "SECRET TITLE",
    "body": "SECRET BODY"
  }]
}`
	var stdout, stderr bytes.Buffer
	_ = run([]string{"sync"}, bytes.NewBufferString(request), &stdout, &stderr)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !bytes.Contains(data, []byte(`"event":"sync_entry"`)) ||
		!bytes.Contains(data, []byte(`"note_type":"imoogi-Basic"`)) {
		t.Fatalf("sync entry was not logged: %s", text)
	}
	if bytes.Contains(data, []byte("SECRET TITLE")) || bytes.Contains(data, []byte("SECRET BODY")) {
		t.Fatalf("card content leaked into log: %s", text)
	}
}
