package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

const validID = "8c3f0f8e-1f8a-4c55-9a59-0e5a3f8f9d21"

func frame(b []byte) []byte {
	var hdr [4]byte
	binary.LittleEndian.PutUint32(hdr[:], uint32(len(b)))
	return append(hdr[:], b...)
}

func TestReadMessageRoundTrip(t *testing.T) {
	r := bytes.NewReader(append(frame([]byte(`{"a":1}`)), frame([]byte(`{}`))...))
	m1, err := ReadMessage(r)
	if err != nil || string(m1) != `{"a":1}` {
		t.Fatalf("first: %q %v", m1, err)
	}
	m2, err := ReadMessage(r)
	if err != nil || string(m2) != `{}` {
		t.Fatalf("second: %q %v", m2, err)
	}
	if _, err := ReadMessage(r); err != io.EOF {
		t.Fatalf("want io.EOF, got %v", err)
	}
}

// countingReader fails the test if more than n bytes are read.
type countingReader struct {
	r    io.Reader
	read int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.read += n
	return n, err
}

func TestReadMessageRejectsOversizeBeforeBody(t *testing.T) {
	var hdr [4]byte
	binary.LittleEndian.PutUint32(hdr[:], MaxIncoming+1)
	cr := &countingReader{r: io.MultiReader(bytes.NewReader(hdr[:]), strings.NewReader(strings.Repeat("x", 1024)))}
	_, err := ReadMessage(cr)
	if !errors.Is(err, ErrMessageTooLarge) {
		t.Fatalf("want ErrMessageTooLarge, got %v", err)
	}
	if cr.read != 4 {
		t.Fatalf("read %d bytes, want only the 4-byte prefix", cr.read)
	}
}

func TestReadMessageTruncated(t *testing.T) {
	b := frame([]byte(`{"a":1}`))
	_, err := ReadMessage(bytes.NewReader(b[:6]))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("want unexpected EOF, got %v", err)
	}
	_, err = ReadMessage(bytes.NewReader(b[:2]))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("partial header: want unexpected EOF, got %v", err)
	}
}

func TestWriteMessageCap(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteMessage(&buf, bytes.Repeat([]byte("a"), MaxOutgoing+1)); err == nil {
		t.Fatal("oversize outgoing accepted")
	}
	if buf.Len() != 0 {
		t.Fatal("oversize write emitted bytes")
	}
	if err := WriteMessage(&buf, []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), frame([]byte(`{}`))) {
		t.Fatalf("bad frame %v", buf.Bytes())
	}
}

func saveJSON(mut func(m map[string]any)) []byte {
	m := map[string]any{
		"schemaVersion":  1,
		"type":           "saveClip",
		"requestId":      validID,
		"category":       "clippings",
		"title":          "Original page title",
		"sourceUrl":      "https://example.com/post?id=1",
		"clippedAt":      "2026-11-08T10:00:00+09:00",
		"downloadAssets": true,
		"blocks": []any{
			map[string]any{"t": "heading", "level": 1, "inl": []any{map[string]any{"t": "text", "v": "Hello"}}},
			map[string]any{"t": "para", "inl": []any{
				map[string]any{"t": "bold", "c": []any{map[string]any{"t": "text", "v": "b"}}},
				map[string]any{"t": "link", "href": "https://x.example/", "c": []any{map[string]any{"t": "text", "v": "x"}}},
				map[string]any{"t": "code", "v": "c"},
				map[string]any{"t": "br"},
				map[string]any{"t": "image", "asset": "a1"},
			}},
			map[string]any{"t": "list", "ordered": true, "items": []any{[]any{map[string]any{"t": "para", "inl": []any{}}}}},
			map[string]any{"t": "quote", "blocks": []any{map[string]any{"t": "hr"}}},
			map[string]any{"t": "table", "header": true, "rows": []any{[]any{[]any{map[string]any{"t": "text", "v": "c"}}}}},
			map[string]any{"t": "code", "text": "x := 1"},
			map[string]any{"t": "image", "asset": "a1"},
		},
		"assets": []any{map[string]any{"id": "a1", "url": "https://cdn.example.com/x.png"}},
	}
	if mut != nil {
		mut(m)
	}
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return b
}

func TestParseValidSaveClip(t *testing.T) {
	req, err := Parse(saveJSON(nil))
	if err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	if req.Type != TypeSaveClip || req.RequestID != validID || req.Save == nil {
		t.Fatalf("bad req %+v", req)
	}
	s := req.Save
	if s.Category != "clippings" || len(s.Blocks) != 7 || len(s.Assets) != 1 || !s.DownloadAssets {
		t.Fatalf("bad save %+v", s)
	}
	if s.ClippedAt.Format("2006-01-02 15:04 -07:00") != "2026-11-08 10:00 +09:00" {
		t.Fatalf("clippedAt %v", s.ClippedAt)
	}
	if s.Blocks[1].Inl[1].Href != "https://x.example/" || s.Blocks[4].Rows[0][0][0].V != "c" {
		t.Fatalf("decoded tree wrong: %+v", s.Blocks)
	}
}

func TestParsePingAndStatus(t *testing.T) {
	req, err := Parse([]byte(`{"schemaVersion":1,"type":"ping"}`))
	if err != nil || req.Type != TypePing {
		t.Fatalf("ping: %+v %v", req, err)
	}
	req, err = Parse([]byte(`{"schemaVersion":1,"type":"status","requestId":"` + validID + `"}`))
	if err != nil || req.Type != TypeStatus || req.RequestID != validID {
		t.Fatalf("status: %+v %v", req, err)
	}
	for _, bad := range []string{
		`{"schemaVersion":1,"type":"ping","requestId":"` + validID + `"}`,
		`{"schemaVersion":1,"type":"status"}`,
		`{"schemaVersion":1,"type":"status","requestId":"nope"}`,
		`{"schemaVersion":2,"type":"ping"}`,
		`{"type":"ping"}`,
		`{"schemaVersion":1,"type":"launch"}`,
		`{"schemaVersion":1.0,"type":"ping"}`,
		`{"schemaVersion":"1","type":"ping"}`,
		`{"schemaVersion":1,"type":"ping"} {}`,
		`{"schemaVersion":1,"type":"ping","type":"ping"}`,
		`[]`, `null`, ``, `{`,
	} {
		if _, err := Parse([]byte(bad)); codeOf(err) != CodeInvalidRequest {
			t.Errorf("%s: want invalid_request, got %v", bad, err)
		}
	}
}

func codeOf(err error) string {
	var pe *Error
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ""
}

func nested(depth int) any {
	// depth counts nodes: a para at depth 1 holding bold nests.
	var inl any = map[string]any{"t": "text", "v": "x"}
	for i := 0; i < depth-2; i++ {
		inl = map[string]any{"t": "bold", "c": []any{inl}}
	}
	return map[string]any{"t": "para", "inl": []any{inl}}
}

func TestParseRejections(t *testing.T) {
	setBlocks := func(b ...any) func(map[string]any) {
		return func(m map[string]any) { m["blocks"] = b }
	}
	manyNodes := make([]any, 0, MaxNodes+1)
	for i := 0; i <= MaxNodes; i++ {
		manyNodes = append(manyNodes, map[string]any{"t": "hr"})
	}
	bigText := strings.Repeat("a", MaxTextBytes/2+1)
	cases := map[string]struct {
		mut  func(map[string]any)
		code string
	}{
		"unknown block t":     {setBlocks(map[string]any{"t": "script"}), CodeInvalidRequest},
		"unknown inline t":    {setBlocks(map[string]any{"t": "para", "inl": []any{map[string]any{"t": "html", "v": "x"}}}), CodeInvalidRequest},
		"extra field":         {setBlocks(map[string]any{"t": "hr", "level": 1}), CodeInvalidRequest},
		"foreign field":       {setBlocks(map[string]any{"t": "para", "inl": []any{}, "onclick": "x"}), CodeInvalidRequest},
		"missing field":       {setBlocks(map[string]any{"t": "heading", "inl": []any{}}), CodeInvalidRequest},
		"null field":          {setBlocks(map[string]any{"t": "heading", "level": nil, "inl": []any{}}), CodeInvalidRequest},
		"wrong type":          {setBlocks(map[string]any{"t": "heading", "level": "1", "inl": []any{}}), CodeInvalidRequest},
		"float level":         {setBlocks(map[string]any{"t": "heading", "level": 1.5, "inl": []any{}}), CodeInvalidRequest},
		"level 7":             {setBlocks(map[string]any{"t": "heading", "level": 7, "inl": []any{}}), CodeInvalidRequest},
		"level 0":             {setBlocks(map[string]any{"t": "heading", "level": 0, "inl": []any{}}), CodeInvalidRequest},
		"depth 13":            {setBlocks(nested(13)), CodeInvalidRequest},
		"too many nodes":      {setBlocks(manyNodes...), CodeInvalidRequest},
		"too much text":       {setBlocks(map[string]any{"t": "code", "text": bigText}, map[string]any{"t": "code", "text": bigText}), CodeInvalidRequest},
		"unknown image asset": {setBlocks(map[string]any{"t": "image", "asset": "a9"}), CodeInvalidRequest},
		"bad category":        {func(m map[string]any) { m["category"] = "../etc" }, CodeUnknownCategory},
		"bad request id":      {func(m map[string]any) { m["requestId"] = "8C3F0F8E-1F8A-4C55-9A59-0E5A3F8F9D21" }, CodeInvalidRequest},
		"uuid v1":             {func(m map[string]any) { m["requestId"] = "8c3f0f8e-1f8a-1c55-9a59-0e5a3f8f9d21" }, CodeInvalidRequest},
		"empty title":         {func(m map[string]any) { m["title"] = " \n\t " }, CodeInvalidRequest},
		"long title":          {func(m map[string]any) { m["title"] = strings.Repeat("가", 1001) }, CodeInvalidRequest},
		"javascript source":   {func(m map[string]any) { m["sourceUrl"] = "javascript:alert(1)" }, CodeInvalidRequest},
		"file source":         {func(m map[string]any) { m["sourceUrl"] = "file:///etc/passwd" }, CodeInvalidRequest},
		"long source":         {func(m map[string]any) { m["sourceUrl"] = "https://e.com/" + strings.Repeat("a", 4096) }, CodeInvalidRequest},
		"bad clippedAt":       {func(m map[string]any) { m["clippedAt"] = "yesterday" }, CodeInvalidRequest},
		"missing assets":      {func(m map[string]any) { delete(m, "assets") }, CodeInvalidRequest},
		"unknown top field":   {func(m map[string]any) { m["root"] = "/etc" }, CodeInvalidRequest},
		"bad asset id":        {func(m map[string]any) { m["assets"] = []any{map[string]any{"id": "../x", "url": "https://a/"}} }, CodeInvalidRequest},
		"duplicate asset id": {func(m map[string]any) {
			m["assets"] = []any{map[string]any{"id": "a1", "url": "https://a/"}, map[string]any{"id": "a1", "url": "https://b/"}}
		}, CodeInvalidRequest},
		"asset extra field": {func(m map[string]any) {
			m["assets"] = []any{map[string]any{"id": "a1", "url": "https://a/", "name": "x.png"}}
		}, CodeInvalidRequest},
		"asset url too long": {func(m map[string]any) {
			m["assets"] = []any{map[string]any{"id": "a1", "url": "https://a/" + strings.Repeat("a", 4096)}}
		}, CodeInvalidRequest},
		"downloadAssets str":   {func(m map[string]any) { m["downloadAssets"] = "true" }, CodeInvalidRequest},
		"list items not array": {setBlocks(map[string]any{"t": "list", "ordered": false, "items": []any{map[string]any{"t": "hr"}}}), CodeInvalidRequest},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse(saveJSON(tc.mut))
			if got := codeOf(err); got != tc.code {
				t.Fatalf("want %s, got %q (%v)", tc.code, got, err)
			}
		})
	}
}

func TestParseTooManyAssets(t *testing.T) {
	_, err := Parse(saveJSON(func(m map[string]any) {
		var as []any
		for i := 0; i <= MaxAssets; i++ {
			as = append(as, map[string]any{"id": "a" + itoa(i), "url": "https://a/"})
		}
		m["assets"] = as
		m["blocks"] = []any{}
	}))
	if codeOf(err) != CodeInvalidRequest {
		t.Fatalf("got %v", err)
	}
}

func itoa(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}

func TestParseDepth12Accepted(t *testing.T) {
	if _, err := Parse(saveJSON(func(m map[string]any) { m["blocks"] = []any{nested(12)} })); err != nil {
		t.Fatalf("depth 12 rejected: %v", err)
	}
}

func TestParseInvalidUTF8(t *testing.T) {
	b := saveJSON(func(m map[string]any) { m["title"] = "TITLEMARK" })
	b = bytes.Replace(b, []byte("TITLEMARK"), []byte("ab\xff\xfe"), 1)
	if _, err := Parse(b); codeOf(err) != CodeInvalidRequest {
		t.Fatalf("invalid utf-8 accepted: %v", err)
	}
}

func TestParseRequestIDEchoOnlyWhenValid(t *testing.T) {
	_, err := Parse(saveJSON(func(m map[string]any) { m["title"] = "" }))
	var pe *Error
	if !errors.As(err, &pe) || pe.RequestID != validID {
		t.Fatalf("want requestId echoed for valid id, got %+v", err)
	}
	_, err = Parse(saveJSON(func(m map[string]any) { m["requestId"] = "<script>" }))
	if !errors.As(err, &pe) || pe.RequestID != "" {
		t.Fatalf("invalid id echoed: %+v", err)
	}
	if strings.Contains(pe.Message, "script") {
		t.Fatalf("message echoes content: %q", pe.Message)
	}
}

func TestTitleNormalised(t *testing.T) {
	req, err := Parse(saveJSON(func(m map[string]any) { m["title"] = "  a\r\n#+SETUPFILE: x\t\u0000b  " }))
	if err != nil {
		t.Fatal(err)
	}
	if req.Save.Title != "a #+SETUPFILE: x b" {
		t.Fatalf("title %q", req.Save.Title)
	}
}

func TestEncodeResultBounded(t *testing.T) {
	r := Result{Type: "result", RequestID: validID, Status: StatusPartial}
	for i := 0; i < 5000; i++ {
		r.FailedAssets = append(r.FailedAssets, FailedAsset{ID: "a1", Reason: "blocked_address"})
	}
	r.Error = &ErrorInfo{Code: CodeInternal, Message: strings.Repeat("m", 100000)}
	b, err := EncodeResponse(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) > MaxOutgoing {
		t.Fatalf("encoded %d > cap", len(b))
	}
	var back Result
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("truncated output not valid JSON: %v", err)
	}
	if back.Status != StatusPartial || back.RequestID != validID {
		t.Fatalf("essential fields lost: %+v", back)
	}
}

func TestEncodeResultShape(t *testing.T) {
	b, err := EncodeResponse(Result{Type: "result", RequestID: validID, Status: StatusSaved, Path: "p/index.org"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"failedAssets":[]`, `"duplicate":false`, `"assetsSaved":0`, `"path":"p/index.org"`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s in %s", want, s)
		}
	}
	if strings.Contains(s, `"error"`) {
		t.Errorf("error present on success: %s", s)
	}
}

func TestCategoryFolder(t *testing.T) {
	if f, ok := CategoryFolder("ai-research"); !ok || f != "00. Inbox/05. AI Research" {
		t.Fatal(f)
	}
	if _, ok := CategoryFolder("__proto__"); ok {
		t.Fatal("unknown category resolved")
	}
}
