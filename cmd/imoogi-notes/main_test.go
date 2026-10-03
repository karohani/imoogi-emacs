package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunKeepsExistingMoveOperation(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "note.org"), []byte("#+TITLE: Move\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	realSource, err := filepath.EvalSymlinks(source)
	if err != nil {
		t.Fatal(err)
	}
	realDestinationParent, err := filepath.EvalSymlinks(filepath.Dir(destination))
	if err != nil {
		t.Fatal(err)
	}
	realDestination := filepath.Join(realDestinationParent, filepath.Base(destination))
	var stdout, stderr bytes.Buffer
	code := run(nil, strings.NewReader(`{"operation":"move","source":"`+source+`","destination":"`+destination+`"}`), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	var response struct {
		OK          bool   `json:"ok"`
		Source      string `json:"source"`
		Destination string `json:"destination"`
		Files       int    `json:"files"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON response %q: %v", stdout.String(), err)
	}
	if !response.OK || response.Source != realSource || response.Destination != realDestination || response.Files != 1 {
		t.Fatalf("unexpected move response: %#v", response)
	}
	if _, err := os.Stat(filepath.Join(destination, "note.org")); err != nil {
		t.Fatalf("destination missing moved file: %v", err)
	}
}

func TestRunDispatchesNestedScopeCacheOperation(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(t.TempDir(), "cache")
	if err := os.WriteFile(filepath.Join(root, "doc.org"), []byte(":PROPERTIES:\n:ID: DOC\n:END:\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	input := `{"version":1,"operation":"catalog","scope":{"project_id":"p","notes_root":"` + root + `"},"cache_dir":"` + cache + `"}`
	var stdout, stderr bytes.Buffer
	code := run(nil, strings.NewReader(input), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	var response struct {
		Version     int  `json:"version"`
		OK          bool `json:"ok"`
		Occurrences map[string][]struct {
			File     string `json:"file"`
			Position int    `json:"position"`
		} `json:"occurrences"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON response %q: %v", stdout.String(), err)
	}
	if response.Version != 1 || !response.OK || len(response.Occurrences["DOC"]) != 1 {
		t.Fatalf("unexpected cache response: %#v", response)
	}
}

func TestReadRequestRejectsOversizedPayload(t *testing.T) {
	payload := strings.NewReader(strings.Repeat("x", (16<<20)+1))
	if _, err := readRequest(payload); err == nil {
		t.Fatal("oversized request was accepted")
	}
}
