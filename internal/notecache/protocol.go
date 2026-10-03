package notecache

const ProtocolVersion = 1
const ParserVersion = 2

type Request struct {
	Version       int       `json:"version,omitempty"`
	Operation     string    `json:"operation"`
	Scope         *Scope    `json:"scope,omitempty"`
	ProjectID     string    `json:"project_id"`
	NotesRoot     string    `json:"notes_root"`
	TasksFile     string    `json:"tasks_file,omitempty"`
	ExcludedRoots []string  `json:"excluded_roots,omitempty"`
	CacheDir      string    `json:"cache_dir,omitempty"`
	Overlays      []Overlay `json:"overlays,omitempty"`
	ID            string    `json:"id,omitempty"`
}

type Scope struct {
	ProjectID     string   `json:"project_id"`
	NotesRoot     string   `json:"notes_root"`
	TasksFile     string   `json:"tasks_file,omitempty"`
	ExcludedRoots []string `json:"excluded_roots,omitempty"`
}

type Overlay struct {
	Path     string `json:"path"`
	Text     string `json:"text"`
	Revision any    `json:"revision,omitempty"`
}

type Response struct {
	Version     int                     `json:"version"`
	OK          bool                    `json:"ok"`
	Status      string                  `json:"status,omitempty"`
	ScopeKey    string                  `json:"scope_key,omitempty"`
	Generation  int64                   `json:"generation,omitempty"`
	Documents   []Document              `json:"documents,omitempty"`
	Occurrences map[string][]Occurrence `json:"occurrences,omitempty"`
	Links       []Link                  `json:"links,omitempty"`
	Files       []FileFingerprint       `json:"files,omitempty"`
	Stats       Stats                   `json:"stats"`
	Errors      []Error                 `json:"errors,omitempty"`
}

type Document struct {
	File          string `json:"file"`
	ID            string `json:"id,omitempty"`
	Title         string `json:"title,omitempty"`
	Kind          string `json:"kind,omitempty"`
	IdentityError string `json:"identity_error,omitempty"`
}

type Occurrence struct {
	File     string `json:"file"`
	Position int    `json:"position"`
}

type Link struct {
	File     string `json:"file"`
	SourceID string `json:"source_id,omitempty"`
	TargetID string `json:"target_id"`
	Kind     string `json:"kind"`
	Position int    `json:"position"`
}

type FileFingerprint struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	Size      int64  `json:"size"`
	ModTimeNS int64  `json:"mtime_ns"`
	Overlay   bool   `json:"overlay,omitempty"`
}

type Stats struct {
	Scanned   int `json:"scanned"`
	Parsed    int `json:"parsed"`
	CacheHits int `json:"cache_hits"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
}

func failure(code, message string) Response {
	return Response{
		Version: ProtocolVersion,
		OK:      false,
		Status:  "error",
		Errors:  []Error{{Code: code, Message: message}},
	}
}
