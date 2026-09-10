package orgpreview

import "strings"

const ProtocolVersion = "org-preview/v1"

type Kind string

const (
	KindHeading   Kind = "heading"
	KindParagraph Kind = "paragraph"
	KindList      Kind = "list"
	KindListItem  Kind = "list_item"
	KindCodeBlock Kind = "code_block"
	KindTable     Kind = "table"
	KindImage     Kind = "image"
	KindFileLink  Kind = "file_link"
	KindText      Kind = "text"
)

type SourceRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

func (r SourceRange) ValidFor(size int) bool {
	return r.Start >= 0 && r.End >= r.Start && r.End <= size
}

type Node struct {
	Kind     string            `json:"kind"`
	Type     string            `json:"type"`
	ID       string            `json:"id"`
	Range    SourceRange       `json:"range"`
	Attrs    map[string]string `json:"attrs,omitempty"`
	Text     string            `json:"text,omitempty"`
	Children []Node            `json:"children,omitempty"`
}

type Document struct {
	Version string      `json:"version"`
	Parser  string      `json:"parser"`
	Range   SourceRange `json:"range"`
	Nodes   []Node      `json:"nodes"`
}

type FlatNode struct {
	Kind      Kind
	ID        string
	Text      string
	StartByte int
	EndByte   int
}

func NewDocument(parser, text string, nodes []Node) Document {
	return Document{
		Version: ProtocolVersion,
		Parser:  parser,
		Range:   SourceRange{Start: 0, End: len(text)},
		Nodes:   nodes,
	}
}

func FindNodeAt(nodes []Node, offset int) *Node {
	var best *Node
	var walk func([]Node)
	walk = func(list []Node) {
		for i := range list {
			node := &list[i]
			if offset < node.Range.Start || offset > node.Range.End {
				continue
			}
			if node.Kind == "block" {
				best = node
			}
			walk(node.Children)
		}
	}
	walk(nodes)
	return best
}

func Flatten(doc Document) []FlatNode {
	var out []FlatNode
	var walk func([]Node)
	walk = func(nodes []Node) {
		for _, node := range nodes {
			kind := Kind(node.Type)
			if kind == "link" {
				kind = KindFileLink
			}
			text := aggregateText(node)
			out = append(out, FlatNode{
				Kind:      kind,
				ID:        node.ID,
				Text:      text,
				StartByte: node.Range.Start,
				EndByte:   node.Range.End,
			})
			walk(node.Children)
		}
	}
	walk(doc.Nodes)
	return out
}

func DiffNormalizedIR(a, b Document) string {
	left := Flatten(a)
	right := Flatten(b)
	if len(left) != len(right) {
		return "node count differs"
	}
	for i := range left {
		if left[i].Kind != right[i].Kind || left[i].Text != right[i].Text ||
			left[i].StartByte != right[i].StartByte || left[i].EndByte != right[i].EndByte {
			return "node differs"
		}
	}
	return ""
}

func aggregateText(node Node) string {
	parts := []string{}
	if node.Text != "" {
		parts = append(parts, node.Text)
	}
	if target := node.Attrs["target"]; target != "" {
		parts = append(parts, target)
	}
	for _, child := range node.Children {
		if text := aggregateText(child); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

func normalizeIDText(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}
