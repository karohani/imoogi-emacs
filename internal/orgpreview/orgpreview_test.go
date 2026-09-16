package orgpreview

import (
	"bytes"
	"encoding/json"
	stdhtml "html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const fixture = `* Title
Paragraph with *bold*, /italic/, ~code~, [[file:doc.txt][doc]], and [[file:image.png][image]].

- first
- second

#+begin_src go
fmt.Println("<escaped>")
#+end_src

| Name | Value |
|------+-------|
| A    | B     |
`

func TestFallbackParserV1Constructs(t *testing.T) {
	doc, err := FallbackParser{}.Parse(fixture)
	if err != nil {
		t.Fatal(err)
	}
	types := flattenTypes(doc.Nodes)
	for _, want := range []string{"heading", "paragraph", "strong", "emphasis", "code", "link", "image", "list", "list_item", "code_block", "table", "table_row", "table_cell"} {
		if !contains(types, want) {
			t.Fatalf("missing node type %q in %#v", want, types)
		}
	}
	seen := map[string]struct{}{}
	var checkIDs func([]Node)
	checkIDs = func(nodes []Node) {
		for _, node := range nodes {
			if node.ID == "" {
				t.Fatalf("node %s has empty id", node.Type)
			}
			if _, ok := seen[node.ID]; ok {
				t.Fatalf("duplicate id %q", node.ID)
			}
			seen[node.ID] = struct{}{}
			if !node.Range.ValidFor(len(fixture)) {
				t.Fatalf("bad range for %s: %#v", node.Type, node.Range)
			}
			checkIDs(node.Children)
		}
	}
	checkIDs(doc.Nodes)
}

func TestPropertyDrawerRendersAsEscapedInformationTable(t *testing.T) {
	source := "* Task\n:PROPERTIES:\n:ID: abc-123\n:OWNER: <jay>\n:END:\nBody\n"
	doc, err := FallbackParser{}.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(flattenTypes(doc.Nodes), "property_drawer") {
		t.Fatalf("property drawer missing from %#v", flattenTypes(doc.Nodes))
	}
	out := Renderer{}.Render(doc)
	for _, want := range []string{`class="org-properties"`, `<th scope="row">ID</th><td>abc-123</td>`, `<th scope="row">OWNER</th><td>&lt;jay&gt;</td>`} {
		if !strings.Contains(out, want) {
			t.Fatalf("property table missing %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{":PROPERTIES:", ":END:", "<jay>"} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("property table leaked %q:\n%s", unwanted, out)
		}
	}
}

func TestMalformedPropertyDrawerRemainsPlainText(t *testing.T) {
	source := "* Task\n:PROPERTIES:\nnot-a-property\n:END:\n"
	doc, err := FallbackParser{}.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	if contains(flattenTypes(doc.Nodes), "property_drawer") {
		t.Fatal("malformed drawer was accepted")
	}
	if !strings.Contains(Renderer{}.Render(doc), ":PROPERTIES:") {
		t.Fatal("malformed drawer content disappeared")
	}
}

func TestMarkdownParserConstructsAndHeadingPalette(t *testing.T) {
	source := "# Red\n## Blue\n### Green\n#### Yellow\n\nParagraph with [doc](doc.txt) and ![image](image.png).\n\n- first\n- second\n\n```go\nfmt.Println(\"ok\")\n```\n"
	doc, err := MarkdownParser{}.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Parser != "fallback-markdown-v1" {
		t.Fatalf("parser = %q", doc.Parser)
	}
	types := flattenTypes(doc.Nodes)
	for _, want := range []string{"heading", "paragraph", "link", "image", "list", "list_item", "code_block"} {
		if !contains(types, want) {
			t.Fatalf("missing node type %q in %#v", want, types)
		}
	}
	out := Renderer{}.Render(doc)
	for _, want := range []string{"palette-red", "palette-blue", "palette-green", "palette-yellow"} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered Markdown heading palette missing %q:\n%s", want, out)
		}
	}
}

func TestOrgAndMarkdownListsPreserveNestedMarkers(t *testing.T) {
	tests := []struct {
		name   string
		parser Parser
	}{
		{name: "org", parser: FallbackParser{}},
		{name: "markdown", parser: MarkdownParser{}},
	}
	source := "- root\n  1. numbered\n    + deep\n  2) sibling\n- tail\n"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := tt.parser.Parse(source)
			if err != nil {
				t.Fatal(err)
			}
			list := firstNodeOfType(t, doc.Nodes, "list")
			if len(list.Children) != 2 {
				t.Fatalf("root items = %d, want 2: %#v", len(list.Children), list)
			}
			root := list.Children[0]
			nested := directChildOfType(t, root, "list")
			if len(nested.Children) != 2 {
				t.Fatalf("nested items = %d, want 2: %#v", len(nested.Children), nested)
			}
			deep := directChildOfType(t, nested.Children[0], "list")
			if got := deep.Children[0].Attrs["marker"]; got != "+" {
				t.Fatalf("deep marker = %q, want +", got)
			}
			if got := nested.Children[0].Attrs["marker"]; got != "1." {
				t.Fatalf("numbered marker = %q, want 1.", got)
			}
			if got := nested.Children[1].Attrs["marker"]; got != "2)" {
				t.Fatalf("dedented marker = %q, want 2)", got)
			}

			out := Renderer{}.Render(doc)
			for _, want := range []string{`class="list-marker">-</span>`, `class="list-marker">1.</span>`, `class="list-marker">+</span>`, `class="list-marker">2)</span>`} {
				if !strings.Contains(out, want) {
					t.Fatalf("rendered list missing %q:\n%s", want, out)
				}
			}
			if !strings.Contains(out, `class="org-list nested-list"`) {
				t.Fatalf("nested list class missing:\n%s", out)
			}
		})
	}
}

func TestListIndentJumpAndMixedWhitespaceAreDeterministic(t *testing.T) {
	source := "- root\n\t+ tab child\n            1. jumped child\n  - shallower child\n- tail\n"
	for _, parser := range []Parser{FallbackParser{}, MarkdownParser{}} {
		doc, err := parser.Parse(source)
		if err != nil {
			t.Fatal(err)
		}
		root := firstNodeOfType(t, doc.Nodes, "list").Children[0]
		level2 := directChildOfType(t, root, "list")
		if len(level2.Children) != 2 || level2.Children[0].Text != "tab child" || level2.Children[1].Text != "shallower child" {
			t.Fatalf("unexpected recovered second level for %s: %#v", parser.Name(), level2.Children)
		}
		level3 := directChildOfType(t, level2.Children[0], "list")
		if len(level3.Children) != 1 || level3.Children[0].Text != "jumped child" {
			t.Fatalf("unexpected indent jump for %s: %#v", parser.Name(), level3.Children)
		}
	}
}

func TestOrgCheckboxesAreReadOnlyAndMarkdownCheckboxTextIsLiteral(t *testing.T) {
	source := "- [ ] 할 일\n- [-] 진행 중\n- [X] 완료\n"
	doc, err := (FallbackParser{}).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	list := firstNodeOfType(t, doc.Nodes, "list")
	wants := []struct{ state, text string }{{"unchecked", "할 일"}, {"partial", "진행 중"}, {"checked", "완료"}}
	for i, want := range wants {
		item := list.Children[i]
		if item.Attrs["checkbox"] != want.state || item.Text != want.text {
			t.Fatalf("item %d = checkbox %q text %q", i, item.Attrs["checkbox"], item.Text)
		}
	}
	out := Renderer{}.Render(doc)
	for _, want := range []string{`class="list-checkbox checkbox-unchecked"`, `class="list-checkbox checkbox-partial"`, `class="list-checkbox checkbox-checked"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("Org checkbox rendering missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, `<input`) {
		t.Fatalf("Org checkbox became interactive:\n%s", out)
	}

	markdown, err := (MarkdownParser{}).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	markdownItem := firstNodeOfType(t, markdown.Nodes, "list").Children[0]
	if markdownItem.Text != "[ ] 할 일" || markdownItem.Attrs["checkbox"] != "" {
		t.Fatalf("Markdown checkbox syntax was interpreted: %#v", markdownItem)
	}
}

func TestNestedListRangesKeepInlineNavigation(t *testing.T) {
	source := "- parent\n  -    [ ] 한글 [[file:child.org][링크]]\n- tail\n"
	doc, err := (FallbackParser{}).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	list := firstNodeOfType(t, doc.Nodes, "list")
	nested := directChildOfType(t, list.Children[0], "list")
	item := nested.Children[0]
	textOffset := strings.Index(source, "한글")
	if item.Text != "한글 [[file:child.org][링크]]" {
		t.Fatalf("nested text = %q", item.Text)
	}
	if item.Children[0].Range.Start != textOffset {
		t.Fatalf("inline range starts at %d, want %d", item.Children[0].Range.Start, textOffset)
	}
	found := FindNodeAt(doc.Nodes, textOffset)
	if found == nil || found.Type != "list_item" || found.ID != item.ID {
		t.Fatalf("FindNodeAt(%d) = %#v, want nested list item %q", textOffset, found, item.ID)
	}
	if !strings.Contains(Renderer{}.Render(doc), `>링크</a>`) {
		t.Fatal("nested inline link was not rendered")
	}
}

func TestRendererEscapesHTMLAndMapsElements(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "doc.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "image.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := FallbackParser{}.Parse(fixture)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := NewAssetResolver([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	out := Renderer{
		Assets:         resolver,
		BaseFile:       filepath.Join(root, "note.org"),
		AssetToken:     "test-token",
		AssetSessionID: "s1",
		AssetBufferID:  "b1",
	}.Render(doc)
	for _, want := range []string{"data-org-id=", "<strong", "<em", ">code</code>", "&lt;escaped&gt;", "<table", "<img"} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered html missing %q:\n%s", want, out)
		}
	}
	for _, want := range []string{"token=test-token", "session=s1", "buffer=b1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("asset URL missing %q:\n%s", want, out)
		}
	}
	for _, want := range []string{"/preview?", "file_path=", "session_id=s1", "buffer_id=b1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("document preview URL missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "<escaped>") {
		t.Fatalf("raw code HTML was not escaped:\n%s", out)
	}
}

func TestRendererEmitsMermaidContainerOnlyForMermaidSourceBlocks(t *testing.T) {
	source := `#+begin_src mermaid
flowchart LR
  A["<script>alert(1)</script>"] --> B
#+end_src

#+begin_src go
fmt.Println("ordinary code")
#+end_src
`
	doc, err := FallbackParser{}.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	out := Renderer{}.Render(doc)
	for _, want := range []string{
		`class="mermaid"`,
		`data-org-kind="code_block"`,
		`flowchart LR`,
		`&lt;script&gt;alert(1)&lt;/script&gt;`,
		`<pre data-org-id=`,
		`ordinary code`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered Mermaid HTML missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, `<script>alert(1)</script>`) {
		t.Fatalf("Mermaid source was not escaped:\n%s", out)
	}
}

func TestRendererStylesHeadingPaletteByOrgDepth(t *testing.T) {
	doc, err := FallbackParser{}.Parse("* Red\n** Blue\n*** Green\n**** Yellow\n***** Red Again\n****** Blue Again\n")
	if err != nil {
		t.Fatal(err)
	}
	out := Renderer{}.Render(doc)
	for _, want := range []string{
		`class="org-heading org-heading-level-1 palette-red"`,
		`data-org-level="1"`,
		`style="color:#ff8c92;background-color:#3b2930"`,
		`class="org-heading org-heading-level-2 palette-blue"`,
		`data-org-level="2"`,
		`style="color:#82b7ff;background-color:#253449"`,
		`class="org-heading org-heading-level-3 palette-green"`,
		`data-org-level="3"`,
		`style="color:#a5d67d;background-color:#2c392b"`,
		`class="org-heading org-heading-level-4 palette-yellow"`,
		`data-org-level="4"`,
		`style="color:#f2d479;background-color:#3d3726"`,
		`class="org-heading org-heading-level-5 palette-red"`,
		`data-org-level="5"`,
		`class="org-heading org-heading-level-6 palette-blue"`,
		`data-org-level="6"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered heading palette missing %q:\n%s", want, out)
		}
	}
}

func TestProtocolValidation(t *testing.T) {
	valid := NewEnvelope("session-1", "buffer-1", 1, OriginEmacs, "event-1")
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid envelope rejected: %v", err)
	}
	invalid := valid
	invalid.Version = "old"
	if err := invalid.Validate(); err == nil {
		t.Fatal("invalid version accepted")
	}
	invalid = valid
	invalid.Origin = "loop"
	if err := invalid.Validate(); err == nil {
		t.Fatal("invalid origin accepted")
	}
	invalid = valid
	invalid.SessionID = "../escape"
	if err := invalid.Validate(); err == nil {
		t.Fatal("invalid session accepted")
	}
}

func TestAssetResolverAllowsOnlyExactOpaqueAssetMapping(t *testing.T) {
	root := t.TempDir()
	asset := filepath.Join(root, "screen.png")
	other := filepath.Join(root, "other.png")
	if err := os.WriteFile(asset, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolver, err := NewAssetResolverWithMap(nil, map[string]string{"imoogi-asset:one": asset})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := resolver.Resolve("", "imoogi-asset:one")
	canonicalAsset, canonicalErr := filepath.EvalSymlinks(asset)
	if canonicalErr != nil {
		t.Fatal(canonicalErr)
	}
	if err != nil || resolved != canonicalAsset {
		t.Fatalf("Resolve exact = %q, %v", resolved, err)
	}
	if _, err := resolver.Resolve("", "imoogi-asset:other"); err == nil {
		t.Fatal("unmapped opaque asset accepted")
	}
	if _, err := resolver.Resolve("", other); err == nil {
		t.Fatal("broad staging root was inferred")
	}
}

func TestAssetResolverRejectsTraversalAndSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}
	resolver, err := NewAssetResolver([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(root, "note.org")
	if _, err := resolver.Resolve(base, "file://"+filepath.Join(root, "ok.txt")); err == nil {
		t.Fatal("file:// target accepted")
	}
	if _, err := resolver.Resolve(base, "../secret.txt"); err == nil {
		t.Fatal("path traversal accepted")
	}
	if _, err := resolver.Resolve(base, "link.txt"); err == nil {
		t.Fatal("symlink escape accepted")
	}
}

func TestServerRevisionAuthAndNewestWins(t *testing.T) {
	server, err := NewServer(ServerConfig{Token: "test-token", Parser: FallbackParser{}})
	if err != nil {
		t.Fatal(err)
	}

	req := RevisionRequest{Envelope: NewEnvelope("s1", "b1", 2, OriginEmacs, "e2"), Text: "* New\n"}
	resp := postRevision(t, server.Handler(), "test-token", req)
	if !resp.Committed || resp.Revision != 2 || !strings.Contains(resp.HTML, "New") {
		t.Fatalf("unexpected revision response: %#v", resp)
	}
	staleReq := RevisionRequest{Envelope: NewEnvelope("s1", "b1", 1, OriginEmacs, "e1"), Text: "* Old\n"}
	stale := postRevision(t, server.Handler(), "test-token", staleReq)
	if !stale.Stale || stale.Committed {
		t.Fatalf("stale revision committed: %#v", stale)
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/api/emacs/revisions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httpReq)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated revision status = %d", rec.Code)
	}
}

func TestServerSelectsMarkdownParserFromRevisionSyntax(t *testing.T) {
	server, err := NewServer(ServerConfig{Token: "test-token", Parser: FallbackParser{}})
	if err != nil {
		t.Fatal(err)
	}
	req := RevisionRequest{
		Envelope: NewEnvelope("s-md", "b-md", 1, OriginEmacs, "e-md"),
		Syntax:   "markdown",
		Text:     "# Markdown title\n",
	}
	resp := postRevision(t, server.Handler(), "test-token", req)
	if resp.Document.Parser != "fallback-markdown-v1" {
		t.Fatalf("parser = %q", resp.Document.Parser)
	}
	if !strings.Contains(resp.HTML, "Markdown title") || !strings.Contains(resp.HTML, "palette-red") {
		t.Fatalf("unexpected Markdown HTML: %s", resp.HTML)
	}
}

func TestServerMapsEmacsNavigationCursorToRenderedSemanticID(t *testing.T) {
	server, err := NewServer(ServerConfig{Token: "test-token", Parser: FallbackParser{}})
	if err != nil {
		t.Fatal(err)
	}
	source := "* Title\nBody text\n"
	cursor := strings.Index(source, "Body")
	req := RevisionRequest{
		Envelope:   NewEnvelope("s1", "b1", 1, OriginEmacs, "revision-1"),
		Text:       source,
		CursorByte: cursor,
	}
	resp := postRevision(t, server.Handler(), "test-token", req)
	want := FindNodeAt(resp.Document.Nodes, cursor)
	if want == nil {
		t.Fatal("fixture did not produce a node at cursor")
	}

	ev := NavigationEvent{
		Envelope:   NewEnvelope("s1", "b1", 1, OriginEmacs, "nav-1"),
		CursorByte: cursor,
		ElementID:  "emacs-org-element-id",
	}
	mapped, ok := server.mapEmacsNavigation(ev)
	if !ok {
		t.Fatal("cursor navigation was not mapped")
	}
	if mapped.ElementID != want.ID || mapped.Range != want.Range {
		t.Fatalf("mapped navigation = %#v, want id=%q range=%#v", mapped, want.ID, want.Range)
	}

	body, err := json.Marshal(NavigationEvent{
		Envelope:   NewEnvelope("s1", "b1", 1, OriginEmacs, "nav-2"),
		CursorByte: cursor,
	})
	if err != nil {
		t.Fatal(err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/api/emacs/navigation", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Org-Preview-Token", "test-token")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httpReq)
	if rec.Code != http.StatusOK {
		t.Fatalf("cursor-only navigation status = %d, body=%s", rec.Code, rec.Body.String())
	}
}

func TestServerAssetRequiresTokenAndMatchingSessionBufferRoot(t *testing.T) {
	root := t.TempDir()
	assetPath := filepath.Join(root, "image.png")
	if err := os.WriteFile(assetPath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(ServerConfig{Token: "test-token", Parser: FallbackParser{}})
	if err != nil {
		t.Fatal(err)
	}
	req := RevisionRequest{
		Envelope:     NewEnvelope("s1", "b1", 1, OriginEmacs, "e1"),
		Generation:   1,
		Text:         "[[file:image.png][image]]",
		Path:         filepath.Join(root, "note.org"),
		AllowedRoots: []string{root},
	}
	resp := postRevision(t, server.Handler(), "test-token", req)
	if strings.Contains(resp.HTML, "test-token") || strings.Contains(resp.HTML, assetPath) || !strings.Contains(resp.HTML, "session=s1") || !strings.Contains(resp.HTML, "generation=1") {
		t.Fatalf("rendered asset URL is not session authenticated:\n%s", resp.HTML)
	}
	assetURL := firstRenderedURL(t, resp.HTML, "src", "/asset?")

	request := httptest.NewRequest(http.MethodGet, "/asset?path="+assetPath+"&session=s1&buffer=b1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, request)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated asset status = %d", rec.Code)
	}

	request = httptest.NewRequest(http.MethodGet, strings.Replace(assetURL, "buffer=b1", "buffer=other", 1), nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, request)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong buffer asset status = %d", rec.Code)
	}

	request = httptest.NewRequest(http.MethodGet, assetURL, nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, request)
	if rec.Code != http.StatusOK || rec.Body.String() != "png" {
		t.Fatalf("authorized asset status/body = %d/%q", rec.Code, rec.Body.String())
	}
	if disposition := rec.Header().Get("Content-Disposition"); disposition != "" {
		t.Fatalf("asset unexpectedly forces a download: %q", disposition)
	}
}

func TestServerFilePreviewRendersSavedOrgAndReusesActiveRevision(t *testing.T) {
	root := t.TempDir()
	linked := filepath.Join(root, "linked.org")
	if err := os.WriteFile(linked, []byte("* Saved heading\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(ServerConfig{Token: "test-token", Parser: FallbackParser{}})
	if err != nil {
		t.Fatal(err)
	}
	origin := postRevision(t, server.Handler(), "test-token", RevisionRequest{
		Envelope:   NewEnvelope("s1", "origin", 1, OriginEmacs, "origin-1"),
		Generation: 1,
		Text:       "[[file:linked.org][linked]]", Path: filepath.Join(root, "origin.org"), AllowedRoots: []string{root},
	})

	previewURL := strings.Replace(firstRenderedURL(t, origin.HTML, "href", "/preview?"), "/preview?", "/api/file-preview?", 1)
	preview := getFilePreview(t, server.Handler(), previewURL)
	if !strings.Contains(preview.HTML, "Saved heading") {
		t.Fatalf("saved Org was not rendered: %s", preview.HTML)
	}

	postRevision(t, server.Handler(), "test-token", RevisionRequest{
		Envelope: NewEnvelope("s1", "linked-buffer", 2, OriginEmacs, "linked-2"), Generation: 1,
		Text: "* Unsaved live heading\n", Path: linked, AllowedRoots: []string{root},
	})
	preview = getFilePreview(t, server.Handler(), previewURL)
	if !strings.Contains(preview.HTML, "Unsaved live heading") || strings.Contains(preview.HTML, "Saved heading") {
		t.Fatalf("active revision was not preferred: %s", preview.HTML)
	}
}

func TestServerFilePreviewRendersSavedMarkdownAndText(t *testing.T) {
	root := t.TempDir()
	markdown := filepath.Join(root, "linked.md")
	textFile := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(markdown, []byte("# Markdown heading\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(textFile, []byte("<plain text>"), 0o644); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(ServerConfig{Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	origin := postRevision(t, server.Handler(), "test-token", RevisionRequest{
		Envelope:   NewEnvelope("s1", "b1", 1, OriginEmacs, "e1"),
		Generation: 1, Text: "[[file:linked.md][md]]\n[[file:notes.txt][text]]\n", Path: filepath.Join(root, "origin.org"), AllowedRoots: []string{root},
	})
	urls := renderedURLs(t, origin.HTML, "href", "/preview?")
	preview := getFilePreview(t, server.Handler(), strings.Replace(urls[0], "/preview?", "/api/file-preview?", 1))
	if !strings.Contains(preview.HTML, "Markdown heading") || !strings.Contains(preview.HTML, "palette-red") {
		t.Fatalf("saved Markdown was not rendered: %s", preview.HTML)
	}
	preview = getFilePreview(t, server.Handler(), strings.Replace(urls[1], "/preview?", "/api/file-preview?", 1))
	if !strings.Contains(preview.HTML, "&lt;plain text&gt;") || strings.Contains(preview.HTML, "<plain text>") {
		t.Fatalf("plain text was not safely rendered: %s", preview.HTML)
	}
}

func TestServerFilePreviewEmbedsPDFWithoutForcingDownload(t *testing.T) {
	root := t.TempDir()
	pdf := filepath.Join(root, "reference.pdf")
	if err := os.WriteFile(pdf, []byte("%PDF-1.4 test"), 0o644); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(ServerConfig{Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	origin := postRevision(t, server.Handler(), "test-token", RevisionRequest{
		Envelope:   NewEnvelope("s1", "b1", 1, OriginEmacs, "e1"),
		Generation: 1, Text: "[[file:reference.pdf][pdf]]\n", Path: filepath.Join(root, "origin.org"), AllowedRoots: []string{root},
	})
	preview := getFilePreview(t, server.Handler(), strings.Replace(firstRenderedURL(t, origin.HTML, "href", "/preview?"), "/preview?", "/api/file-preview?", 1))
	if !strings.Contains(preview.HTML, `<iframe class="linked-media"`) || !strings.Contains(preview.HTML, "/asset?") {
		t.Fatalf("PDF was not embedded in the preview: %s", preview.HTML)
	}
	assetURL := firstRenderedURL(t, preview.HTML, "src", "/asset?")
	parsedAssetURL, err := url.Parse(assetURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsedAssetURL.Query().Get("id") == "" || parsedAssetURL.Query().Get("token") != "" || parsedAssetURL.Query().Get("path") != "" || strings.Contains(preview.HTML, pdf) {
		t.Fatalf("linked PDF URL exposed control credentials or a path: %s", assetURL)
	}
	request := httptest.NewRequest(http.MethodGet, assetURL, nil)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "%PDF-1.4 test" {
		t.Fatalf("linked PDF asset fetch = %d/%q", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(preview.HTML, "%PDF-1.4") {
		t.Fatalf("PDF body leaked into preview HTML: %s", preview.HTML)
	}
}

func TestServerFilePreviewRequiresAuthentication(t *testing.T) {
	server, err := NewServer(ServerConfig{Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/file-preview?path=/tmp/file", nil)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated file preview status = %d", recorder.Code)
	}
}

func TestServerFilePreviewBlocksUnsupportedAndOutsideFiles(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	archive := filepath.Join(root, "archive.zip")
	executable := filepath.Join(root, "script.sh")
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(archive, []byte("PK binary payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("#!/bin/sh\necho secret\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(ServerConfig{Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	origin := postRevision(t, server.Handler(), "test-token", RevisionRequest{
		Envelope:   NewEnvelope("s1", "b1", 1, OriginEmacs, "e1"),
		Generation: 1, Text: "[[file:archive.zip][archive]]\n[[file:script.sh][script]]\n", Path: filepath.Join(root, "origin.org"), AllowedRoots: []string{root},
	})
	urls := renderedURLs(t, origin.HTML, "href", "/preview?")
	preview := getFilePreview(t, server.Handler(), strings.Replace(urls[0], "/preview?", "/api/file-preview?", 1))
	if !strings.Contains(preview.HTML, "미리보기 미지원") || strings.Contains(preview.HTML, "PK binary payload") {
		t.Fatalf("unsupported file response = %s", preview.HTML)
	}
	preview = getFilePreview(t, server.Handler(), strings.Replace(urls[1], "/preview?", "/api/file-preview?", 1))
	if !strings.Contains(preview.HTML, "미리보기 미지원") || strings.Contains(preview.HTML, "echo secret") {
		t.Fatalf("executable file response = %s", preview.HTML)
	}

	request := httptest.NewRequest(http.MethodGet,
		strings.Replace(strings.Replace(urls[0], "/preview?", "/api/file-preview?", 1), "id=", "id=forged", 1), nil)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest || strings.Contains(recorder.Body.String(), "secret") {
		t.Fatalf("outside preview status/body = %d/%q", recorder.Code, recorder.Body.String())
	}
}

func getFilePreview(t *testing.T, handler http.Handler, target string) filePreviewResponse {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("file preview status/body = %d/%q", recorder.Code, recorder.Body.String())
	}
	var response filePreviewResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func TestServerServesPreviewShellAndBrowserWebSocketQueryContract(t *testing.T) {
	server, err := NewServer(ServerConfig{Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/preview?session_id=s1&buffer_id=b1&token=test-token", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, request)

	if rec.Code != http.StatusOK {
		t.Fatalf("preview shell status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"session_id",
		"buffer_id",
		"/ws/browser",
		`id="toc"`,
		`id="overview"`,
		"--red:#ff8c92",
		"--blue:#82b7ff",
		"--green:#a5d67d",
		"--yellow:#f2d479",
		".org-preview .palette-red{color:var(--red);background:var(--red-bg)}",
		"function markElement",
		"function rebuildSidebars",
		"function levelOf(el){return Number(el.dataset.orgLevel)",
		"window.scrollTo(x, y)",
		`<script src="/static/mermaid.min.js"></script>`,
		"mermaid.initialize({startOnLoad:false,securityLevel:'strict',theme:'dark'})",
		"await mermaid.run({nodes:nodes,suppressErrors:true})",
		"/api/file-preview",
		"if (linkedFileID) loadLinkedFile(); else connect();",
		".org-properties",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("preview shell missing %q:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{"window.onscroll", "block:'center'"} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("preview shell should not auto-scroll on navigation; found %q:\n%s", unwanted, body)
		}
	}
}

func TestServerServesEmbeddedMermaidRuntime(t *testing.T) {
	server, err := NewServer(ServerConfig{Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/static/mermaid.min.js", nil)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("Mermaid runtime status = %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "text/javascript") {
		t.Fatalf("Mermaid runtime content type = %q", got)
	}
	if recorder.Body.Len() < 1_000_000 || !strings.Contains(recorder.Body.String(), "mermaid") {
		t.Fatalf("embedded Mermaid runtime looks incomplete: %d bytes", recorder.Body.Len())
	}
}

func TestServerDeduplicatesOriginEvents(t *testing.T) {
	server, err := NewServer(ServerConfig{Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	ev := NavigationEvent{Envelope: NewEnvelope("s1", "b1", 3, OriginBrowser, "same-event"), ElementID: "heading-1"}
	if !server.markSeen(ev.SessionID, ev.EventID) {
		t.Fatal("first event should be accepted")
	}
	if server.markSeen(ev.SessionID, ev.EventID) {
		t.Fatal("duplicate event should be suppressed")
	}
}

func postRevision(t *testing.T, handler http.Handler, token string, req RevisionRequest) RevisionResponse {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/api/emacs/revisions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Org-Preview-Token", token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httpReq)
	data, _ := io.ReadAll(rec.Result().Body)
	if rec.Code != http.StatusAccepted && rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, data)
	}
	var out RevisionResponse
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func firstRenderedURL(t *testing.T, body, attribute, prefix string) string {
	t.Helper()
	urls := renderedURLs(t, body, attribute, prefix)
	return urls[0]
}

func renderedURLs(t *testing.T, body, attribute, prefix string) []string {
	t.Helper()
	re := regexp.MustCompile(attribute + `="([^"]+)"`)
	var urls []string
	for _, match := range re.FindAllStringSubmatch(body, -1) {
		value := stdhtml.UnescapeString(match[1])
		if strings.HasPrefix(value, prefix) {
			urls = append(urls, value)
		}
	}
	if len(urls) == 0 {
		t.Fatalf("no rendered %s URL with prefix %q in %s", attribute, prefix, body)
	}
	return urls
}

func flattenTypes(nodes []Node) []string {
	var out []string
	var walk func([]Node)
	walk = func(list []Node) {
		for _, node := range list {
			out = append(out, node.Type)
			walk(node.Children)
		}
	}
	walk(nodes)
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func firstNodeOfType(t *testing.T, nodes []Node, typ string) Node {
	t.Helper()
	for _, node := range nodes {
		if node.Type == typ {
			return node
		}
	}
	t.Fatalf("node type %q not found in %#v", typ, nodes)
	return Node{}
}

func directChildOfType(t *testing.T, node Node, typ string) Node {
	t.Helper()
	for _, child := range node.Children {
		if child.Type == typ {
			return child
		}
	}
	t.Fatalf("direct child type %q not found in %#v", typ, node)
	return Node{}
}

func TestHeadingColorsPreserveOrgDepth(t *testing.T) {
	colors := []string{"#ff8c92", "#82b7ff", "#a5d67d", "#f2d479"}
	backgrounds := []string{"#3b2930", "#253449", "#2c392b", "#3d3726"}
	for depth := 1; depth <= 12; depth++ {
		source := strings.Repeat("*", depth) + " Heading with ~code~\n"
		doc, err := (FallbackParser{}).Parse(source)
		if err != nil {
			t.Fatal(err)
		}
		output := (Renderer{}).Render(doc)
		wants := []string{
			`data-org-level="` + strconv.Itoa(depth) + `"`,
			`<h` + strconv.Itoa(min(depth, 6)) + ` `,
			`color:` + colors[(depth-1)%4] + `;background-color:` + backgrounds[(depth-1)%4],
		}
		for _, want := range wants {
			if !strings.Contains(output, want) {
				t.Fatalf("depth %d missing %s: %s", depth, want, output)
			}
		}
	}
}
