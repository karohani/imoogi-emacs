package clipboard

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGoldenRequest(t *testing.T) {
	data := readGolden(t, "request-import.json")
	request, err := DecodeRequest(data)
	if err != nil {
		t.Fatal(err)
	}
	if request.Operation != OperationImport || request.Owner == nil || request.Owner.Kind != OwnerProjectNotes {
		t.Fatalf("unexpected request: %#v", request)
	}
	if got, want := request.Paths, []string{"/tmp/a.png", "/tmp/b.pdf"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %#v, want %#v", got, want)
	}
}

func TestGoldenOrphansRequest(t *testing.T) {
	request, err := DecodeRequest(readGolden(t, "request-orphans.json"))
	if err != nil {
		t.Fatal(err)
	}
	if request.Operation != OperationOrphans || request.Owner == nil || request.Owner.AssetDir != "50. Assets" {
		t.Fatalf("unexpected request: %#v", request)
	}
	if got, want := request.Protected, []string{"/notes/project/50. Assets/unsaved.png"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("protected = %#v, want %#v", got, want)
	}
	if request.GraceSeconds != 3600 {
		t.Fatalf("grace_seconds = %d", request.GraceSeconds)
	}
	trash := []byte(`{"protocol_version":1,"operation":"trash","correlation_id":"c","paths":["/a.png"]}`)
	if request, err := DecodeRequest(trash); err != nil || request.Operation != OperationTrash {
		t.Fatalf("trash request = %#v, err = %v", request, err)
	}
}

func TestOrphanResponseEncoding(t *testing.T) {
	response := Success(OperationOrphans, "c")
	response.Orphaned = []Orphan{{Path: "/a.png", Size: 3, ModTime: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}}
	response.Trashed = []string{"/b.png"}
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	want := `"orphaned":[{"path":"/a.png","size":3,"mtime":"2026-01-02T03:04:05Z"}],"trashed":["/b.png"]`
	if !strings.Contains(string(data), want) {
		t.Fatalf("response = %s", data)
	}
}

func TestGoldenResponse(t *testing.T) {
	data := readGolden(t, "response-inspect.json")
	var response Response
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if response.ProtocolVersion != ProtocolVersion || response.ClipboardID != "macos:42" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestDecodeRequestRejectsUnknownField(t *testing.T) {
	_, err := DecodeRequest([]byte(`{"protocol_version":1,"operation":"version","correlation_id":"c","extra":true}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeRequestRejectsTrailingJSON(t *testing.T) {
	data := []byte(`{"protocol_version":1,"operation":"version","correlation_id":"c"} {}`)
	if _, err := DecodeRequest(data); err == nil {
		t.Fatal("trailing JSON value accepted")
	}
}

func TestOwnerShape(t *testing.T) {
	tests := []struct {
		name  string
		owner Owner
		ok    bool
	}{
		{"project", Owner{Kind: OwnerProjectNotes, Document: "/n/p/a.org", Root: "/n/p"}, true},
		{"project missing document", Owner{Kind: OwnerProjectNotes, Root: "/n/p"}, false},
		{"standalone", Owner{Kind: OwnerStandalone, Document: "/n/a.org", Root: "/n/a.assets"}, true},
		{"staging", Owner{Kind: OwnerStaging, Session: "s", Buffer: "b", Generation: 1}, true},
		{"staging generation", Owner{Kind: OwnerStaging, Session: "s", Buffer: "b"}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.owner.ValidateShape()
			if (err == nil) != test.ok {
				t.Fatalf("ValidateShape() error = %v, ok = %v", err, test.ok)
			}
		})
	}
}

func TestManifestHasRequiredFields(t *testing.T) {
	typeInfo := reflect.TypeOf(Manifest{})
	for _, name := range []string{"TransactionID", "TransactionToken", "Identity", "Owner", "State", "Assets", "Rewrite", "CreatedFiles", "CreatedAt", "UpdatedAt"} {
		if _, ok := typeInfo.FieldByName(name); !ok {
			t.Fatalf("Manifest missing %s", name)
		}
	}
}

func readGolden(t *testing.T, name string) []byte {
	t.Helper()
	file, err := os.Open(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
