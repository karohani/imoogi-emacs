package orgpreview

import (
	"fmt"
	"strings"
	"testing"
)

func TestMarkdownFrontmatterRendersMetadataSeparately(t *testing.T) {
	source := "---\nname: architect\ndescription: \"구현 전에 타입, 함수 시그니처, 모듈 구조를 설계합니다.\"\ndisable-model-invocation: true\n---\n> 원문: SKILL.md\n\n# 핵심 역할\n본문\n"
	doc, err := (MarkdownParser{}).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Nodes) < 2 || doc.Nodes[0].Type != "property_drawer" {
		t.Fatalf("frontmatter not parsed: %#v", doc.Nodes[0])
	}
	props := doc.Nodes[0].Children
	if len(props) != 3 || props[0].Attrs["key"] != "name" || props[0].Attrs["value"] != "architect" || props[1].Attrs["value"] != "구현 전에 타입, 함수 시그니처, 모듈 구조를 설계합니다." || props[2].Attrs["value"] != "true" {
		t.Fatalf("metadata = %#v", props)
	}
	bodyStart := strings.Index(source, "> 원문")
	if doc.Nodes[0].Range.End != bodyStart || doc.Nodes[1].Range.Start != bodyStart {
		t.Fatalf("body offset changed: %#v", doc.Nodes)
	}
	out := (Renderer{}).Render(doc)
	if !strings.Contains(out, `data-org-format="yaml"`) {
		t.Fatalf("YAML metadata lacks a semantic marker: %s", out)
	}
	if !strings.Contains(out, `<th scope="row">name</th><td>architect</td>`) || strings.Contains(out, "---") || !strings.Contains(out, "원문: SKILL.md") {
		t.Fatalf("HTML: %s", out)
	}
}

func TestOrgPropertyDrawerDoesNotReceiveYAMLMarker(t *testing.T) {
	doc, err := (FallbackParser{}).Parse("* Task\n:PROPERTIES:\n:ID: example\n:END:\nBody\n")
	if err != nil {
		t.Fatal(err)
	}
	out := (Renderer{}).Render(doc)
	if !strings.Contains(out, `class="org-properties"`) || strings.Contains(out, `data-org-format="yaml"`) {
		t.Fatalf("Org property drawer must remain distinct from YAML metadata: %s", out)
	}
}

func TestMarkdownFrontmatterPreservesLineEndingsRangesAndNestedValues(t *testing.T) {
	source := "\ufeff---\r\nname: '한글: 설계'\r\ndescription: |\r\n  첫 줄\r\n  둘째 줄\r\ntags: [design, emacs]\r\nsettings:\r\n  enabled: true\r\n---\r\n# 본문\r\n"
	doc, err := (MarkdownParser{}).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Nodes) != 2 || doc.Nodes[0].Type != "property_drawer" {
		t.Fatalf("nodes: %#v", doc.Nodes)
	}
	props := doc.Nodes[0].Children
	if len(props) != 4 || props[0].Attrs["value"] != "한글: 설계" || props[1].Attrs["value"] != "첫 줄\n둘째 줄\n" || !strings.Contains(props[2].Attrs["value"], "design") || !strings.Contains(props[3].Attrs["value"], "enabled: true") {
		t.Fatalf("properties: %#v", props)
	}
	if doc.Range.End != len(source) || doc.Nodes[1].Range.Start != strings.Index(source, "# 본문") {
		t.Fatalf("ranges: %#v", doc)
	}
	for _, prop := range props {
		if !prop.Range.ValidFor(len(source)) || prop.Range.End > doc.Nodes[0].Range.End {
			t.Fatalf("property range: %#v", prop)
		}
	}
	if !strings.Contains((Renderer{}).Render(doc), "white-space:pre-wrap") {
		t.Fatal("multiline metadata whitespace would collapse")
	}
}

func TestMarkdownFrontmatterLeavesNonMetadataUntouched(t *testing.T) {
	for _, source := range []string{
		"---\nname: unfinished\n# Body\n", "---\nname: [broken\n---\nBody\n",
		"---\nname: one\nname: two\n---\nBody\n", "---\nplain text\n---\nBody\n",
		"Before\n\n---\nname: not-metadata\n---\n", "```yaml\n---\nname: example\n---\n```\n",
		"---\nrecursive: &loop [*loop]\n---\nBody\n", "---\nname: *missing\n---\nBody\n",
		"---\nsettings: {key: one, key: two}\n---\nBody\n", "---\n- item\n---\nBody\n",
	} {
		t.Run(source, func(t *testing.T) {
			doc, err := (MarkdownParser{}).Parse(source)
			if err != nil {
				t.Fatal(err)
			}
			if contains(flattenTypes(doc.Nodes), "property_drawer") {
				t.Fatal("non-metadata parsed as frontmatter")
			}
			if !strings.Contains((Renderer{}).Render(doc), "---") {
				t.Fatal("original content lost")
			}
		})
	}
}

func TestMarkdownFrontmatterServerEscapesHTMLAndKeepsBody(t *testing.T) {
	server, err := NewServer(ServerConfig{Token: "test-token", Parser: FallbackParser{}})
	if err != nil {
		t.Fatal(err)
	}
	resp := postRevision(t, server.Handler(), "test-token", RevisionRequest{Envelope: NewEnvelope("frontmatter", "buffer", 1, OriginEmacs, "event"), Syntax: "markdown", Text: "---\nname: '<script>alert(1)</script>'\n---\n# Body\n"})
	if !resp.Committed || !strings.Contains(resp.HTML, "org-properties") || !strings.Contains(resp.HTML, "&lt;script&gt;") || strings.Contains(resp.HTML, "<script>") || !strings.Contains(resp.HTML, "Body") {
		t.Fatalf("response: %#v", resp)
	}
}

func TestMarkdownFrontmatterFlowMappingRangesAreNotEmpty(t *testing.T) {
	source := "---\n{a: x, b: y}\n---\nBody\n"
	doc, err := (MarkdownParser{}).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	props := doc.Nodes[0].Children
	if len(props) != 2 {
		t.Fatalf("properties: %#v", props)
	}
	for _, prop := range props {
		if prop.Range.Start >= prop.Range.End || !strings.Contains(source[prop.Range.Start:prop.Range.End], prop.Attrs["key"]+":") {
			t.Fatalf("empty or wrong source range: %#v", prop)
		}
	}
}

func TestMarkdownFrontmatterEmptyMappingIsMetadata(t *testing.T) {
	source := "---\n{}\n---\nBody\n"
	doc, err := (MarkdownParser{}).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Nodes) != 2 || doc.Nodes[0].Type != "property_drawer" || len(doc.Nodes[0].Children) != 0 || doc.Nodes[1].Text != "Body" {
		t.Fatalf("nodes: %#v", doc.Nodes)
	}
}

func TestMarkdownFrontmatterDoesNotHideTrailingYAMLContent(t *testing.T) {
	source := "---\nname: architect\n...\nnot valid: [yaml\n---\nBody\n"
	doc, err := (MarkdownParser{}).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	if contains(flattenTypes(doc.Nodes), "property_drawer") {
		t.Fatal("invalid content after YAML document end accepted")
	}
	if !strings.Contains((Renderer{}).Render(doc), "not valid: [yaml") {
		t.Fatal("trailing YAML content disappeared")
	}
}

// YAML counts bare CR and Unicode line separators as line breaks while the
// Markdown line splitter does not; such front matter must fall back, not panic.
func TestMarkdownFrontmatterWithForeignLineBreaksFallsBack(t *testing.T) {
	for _, source := range []string{
		"---\na: 1\rb: 2\rc: 3\rd: 4\re: 5\n---\nbody\n",
		"---\na: 1 # x        \nb: 2\n---\nbody\n",
		"---\na: |\n  x\u0085y\u0085z\u0085w\nb: 2\n---\nbody\n",
	} {
		t.Run(strings.ReplaceAll(source, "\n", `\n`), func(t *testing.T) {
			doc, err := (MarkdownParser{}).Parse(source)
			if err != nil {
				t.Fatal(err)
			}
			if contains(flattenTypes(doc.Nodes), "property_drawer") {
				t.Fatal("front matter with foreign line breaks must not become metadata")
			}
		})
	}
}

// Large front matter is treated as ordinary text so one file cannot pin a CPU
// core (yaml.v3's duplicate-key check is quadratic) or inflate the response.
func TestMarkdownFrontmatterSizeLimits(t *testing.T) {
	keys := func(n int) string {
		var b strings.Builder
		b.WriteString("---\n")
		for i := 0; i < n; i++ {
			fmt.Fprintf(&b, "k%d: v\n", i)
		}
		b.WriteString("---\nbody\n")
		return b.String()
	}
	for _, tc := range []struct {
		name   string
		source string
		want   bool
	}{
		{"at key limit", keys(maxFrontmatterKeys), true},
		{"over key limit", keys(maxFrontmatterKeys + 1), false},
		{"over byte limit", "---\nbig: [" + strings.Repeat("a,", maxFrontmatterBytes/2) + "a]\n---\nbody\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := (MarkdownParser{}).Parse(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			if got := contains(flattenTypes(doc.Nodes), "property_drawer"); got != tc.want {
				t.Fatalf("property_drawer = %v, want %v", got, tc.want)
			}
		})
	}
}
