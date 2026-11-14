package orgpreview

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// Front matter beyond these limits is left to the Markdown parser as plain text:
// yaml.v3 checks duplicate keys pairwise, and non-scalar values are re-encoded.
const (
	maxFrontmatterBytes = 64 << 10
	maxFrontmatterKeys  = 1000
)

// Only a closed mapping at the beginning of the document is metadata. On a
// malformed or unfinished edit, leave the original text to the Markdown parser.
func parseMarkdownFrontmatter(lines []lineSpan, ids map[string]int) (Node, int, bool) {
	if len(lines) < 3 || strings.TrimRight(strings.TrimPrefix(lines[0].Text, "\ufeff"), " \t\r\n") != "---" {
		return Node{}, 0, false
	}
	for end := 1; end < len(lines); end++ {
		if strings.TrimRight(lines[end].Text, " \t\r\n") != "---" {
			continue
		}
		var source strings.Builder
		for _, line := range lines[1:end] {
			source.WriteString(line.Text)
		}
		if source.Len() > maxFrontmatterBytes || hasForeignLineBreak(source.String()) {
			return Node{}, 0, false
		}
		var document yaml.Node
		decoder := yaml.NewDecoder(strings.NewReader(source.String()))
		if err := decoder.Decode(&document); err != nil || len(document.Content) != 1 {
			return Node{}, 0, false
		}
		if err := decoder.Decode(new(yaml.Node)); err != io.EOF {
			return Node{}, 0, false
		}
		mapping := document.Content[0]
		if mapping.Kind != yaml.MappingNode || len(mapping.Content)/2 > maxFrontmatterKeys {
			return Node{}, 0, false
		}
		// Node parsing preserves order; decoding additionally rejects duplicate keys
		// and invalid aliases (including recursive/excessively expanded aliases).
		var validated map[string]any
		if err := mapping.Decode(&validated); err != nil {
			return Node{}, 0, false
		}
		properties := make([]Node, 0, len(mapping.Content)/2)
		for i := 0; i < len(mapping.Content); i += 2 {
			key, value := mapping.Content[i], mapping.Content[i+1]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
				return Node{}, 0, false
			}
			text := value.Value
			if value.Kind != yaml.ScalarNode {
				encoded, err := yaml.Marshal(value)
				if err != nil {
					return Node{}, 0, false
				}
				text = strings.TrimSuffix(string(encoded), "\n")
			}
			next := end
			if i+2 < len(mapping.Content) {
				next = mapping.Content[i+2].Line
			}
			// yaml.v3 lines are 1-based within the front matter, which is offset by
			// the opening marker; anything outside it means the two disagree.
			if key.Line < 1 || key.Line >= end || next < key.Line || next > end {
				return Node{}, 0, false
			}
			endByte := lines[next].Start
			if next == key.Line {
				endByte = lines[key.Line].End
			}
			properties = append(properties, Node{
				Kind: "span", Type: "property", ID: nextID(ids, "property", normalizeIDText(key.Value)+":"+normalizeIDText(text)),
				Range: SourceRange{Start: lines[key.Line].Start, End: endByte},
				Attrs: map[string]string{"key": key.Value, "value": text}, Text: text,
			})
		}
		return Node{Kind: "block", Type: "property_drawer", ID: nextID(ids, "property_drawer", fmt.Sprintf("%d:%d", lines[0].Start, lines[end].End)), Range: SourceRange{Start: lines[0].Start, End: lines[end].End}, Attrs: map[string]string{"format": "yaml"}, Children: properties}, end + 1, true
	}
	return Node{}, 0, false
}

// yaml.v3 also breaks lines on a bare CR, NEL, LS and PS, which the Markdown
// line splitter does not; their line numbers would no longer match.
func hasForeignLineBreak(text string) bool {
	for i, r := range text {
		switch r {
		case '\r':
			if i+1 >= len(text) || text[i+1] != '\n' {
				return true
			}
		case '\u0085', '\u2028', '\u2029':
			return true
		}
	}
	return false
}
