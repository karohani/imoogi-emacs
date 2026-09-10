package orgpreview

import (
	"html"
	"net/url"
	"strconv"
	"strings"
)

type Renderer struct {
	Assets         AssetResolver
	BaseFile       string
	AssetToken     string
	AssetSessionID string
	AssetBufferID  string
}

func (r Renderer) Render(doc Document) string {
	var b strings.Builder
	b.WriteString(`<main class="org-preview" data-protocol="`)
	b.WriteString(html.EscapeString(doc.Version))
	b.WriteString(`">`)
	for _, node := range doc.Nodes {
		r.renderNode(&b, node)
	}
	b.WriteString(`</main>`)
	return b.String()
}

func (r Renderer) renderNode(b *strings.Builder, n Node) {
	switch n.Type {
	case "heading":
		depth, err := strconv.Atoi(n.Attrs["level"])
		if err != nil || depth < 1 {
			depth = 1
		}
		level := strconv.Itoa(min(depth, 6))
		palette := [4][3]string{
			{"red", "#ff8c92", "#3b2930"},
			{"blue", "#82b7ff", "#253449"},
			{"green", "#a5d67d", "#2c392b"},
			{"yellow", "#f2d479", "#3d3726"},
		}
		color := palette[(depth-1)%len(palette)]
		r.openMappedAttrs(b, "h"+level, n, map[string]string{
			"class":          "org-heading org-heading-level-" + strconv.Itoa(depth) + " palette-" + color[0],
			"data-org-level": strconv.Itoa(depth),
			"style":          "color:" + color[1] + ";background-color:" + color[2],
		})
		r.renderChildrenOrText(b, n)
		b.WriteString("</h")
		b.WriteString(level)
		b.WriteByte('>')
	case "paragraph":
		r.openMapped(b, "p", n)
		r.renderChildrenOrText(b, n)
		b.WriteString("</p>")
	case "list":
		r.openMapped(b, "ul", n)
		for _, child := range n.Children {
			r.renderNode(b, child)
		}
		b.WriteString("</ul>")
	case "list_item":
		r.openMapped(b, "li", n)
		r.renderChildrenOrText(b, n)
		b.WriteString("</li>")
	case "code_block":
		r.openMapped(b, "pre", n)
		b.WriteString("<code>")
		b.WriteString(escapeText(n.Text))
		b.WriteString("</code></pre>")
	case "table":
		r.openMapped(b, "table", n)
		b.WriteString("<tbody>")
		for _, child := range n.Children {
			r.renderNode(b, child)
		}
		b.WriteString("</tbody></table>")
	case "table_row":
		r.openMapped(b, "tr", n)
		for _, child := range n.Children {
			r.renderNode(b, child)
		}
		b.WriteString("</tr>")
	case "table_cell":
		r.openMapped(b, "td", n)
		r.renderChildrenOrText(b, n)
		b.WriteString("</td>")
	case "strong":
		r.openMapped(b, "strong", n)
		b.WriteString(escapeText(n.Text))
		b.WriteString("</strong>")
	case "emphasis":
		r.openMapped(b, "em", n)
		b.WriteString(escapeText(n.Text))
		b.WriteString("</em>")
	case "code":
		r.openMapped(b, "code", n)
		b.WriteString(escapeText(n.Text))
		b.WriteString("</code>")
	case "link":
		r.renderLink(b, n)
	case "image":
		r.renderImage(b, n)
	case "text":
		b.WriteString(escapeText(n.Text))
	default:
		r.openMapped(b, "span", n)
		b.WriteString(escapeText(n.Text))
		b.WriteString("</span>")
	}
}

func (r Renderer) renderChildrenOrText(b *strings.Builder, n Node) {
	if len(n.Children) == 0 {
		b.WriteString(escapeText(n.Text))
		return
	}
	for _, child := range n.Children {
		r.renderNode(b, child)
	}
}

func (r Renderer) renderLink(b *strings.Builder, n Node) {
	target := n.Attrs["target"]
	label := n.Attrs["label"]
	if label == "" {
		label = target
	}
	resolved, err := r.Assets.Resolve(r.BaseFile, target)
	if err != nil {
		r.openMapped(b, "a", n)
		b.WriteString(escapeText(label))
		b.WriteString("</a>")
		return
	}
	r.openMappedAttrs(b, "a", n, map[string]string{"href": r.assetURL(resolved)})
	b.WriteString(escapeText(label))
	b.WriteString("</a>")
}

func (r Renderer) renderImage(b *strings.Builder, n Node) {
	target := n.Attrs["target"]
	label := n.Attrs["label"]
	resolved, err := r.Assets.Resolve(r.BaseFile, target)
	if err != nil {
		r.openMapped(b, "span", n)
		b.WriteString(escapeText(label))
		b.WriteString("</span>")
		return
	}
	r.openMappedAttrs(b, "img", n, map[string]string{"src": r.assetURL(resolved), "alt": label})
}

func (r Renderer) openMappedAttrs(b *strings.Builder, tag string, n Node, attrs map[string]string) {
	b.WriteByte('<')
	b.WriteString(tag)
	b.WriteString(` data-org-id="`)
	b.WriteString(html.EscapeString(n.ID))
	b.WriteString(`" data-org-preview-id="`)
	b.WriteString(html.EscapeString(n.ID))
	b.WriteString(`" data-org-kind="`)
	b.WriteString(html.EscapeString(n.Type))
	b.WriteString(`" data-org-range-start="`)
	b.WriteString(intString(n.Range.Start))
	b.WriteString(`" data-org-range-end="`)
	b.WriteString(intString(n.Range.End))
	b.WriteByte('"')
	for key, value := range attrs {
		b.WriteByte(' ')
		b.WriteString(html.EscapeString(key))
		b.WriteString(`="`)
		b.WriteString(html.EscapeString(value))
		b.WriteByte('"')
	}
	b.WriteByte('>')
}

func (r Renderer) openMapped(b *strings.Builder, tag string, n Node) {
	b.WriteByte('<')
	b.WriteString(tag)
	b.WriteString(` data-org-id="`)
	b.WriteString(html.EscapeString(n.ID))
	b.WriteString(`" data-org-preview-id="`)
	b.WriteString(html.EscapeString(n.ID))
	b.WriteString(`" data-org-kind="`)
	b.WriteString(html.EscapeString(n.Type))
	b.WriteString(`" data-org-range-start="`)
	b.WriteString(intString(n.Range.Start))
	b.WriteString(`" data-org-range-end="`)
	b.WriteString(intString(n.Range.End))
	b.WriteString(`">`)
}

func intString(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

func escapeText(s string) string {
	escaped := html.EscapeString(s)
	escaped = strings.ReplaceAll(escaped, "onerror=", "onerror&#61;")
	escaped = strings.ReplaceAll(escaped, "onload=", "onload&#61;")
	return escaped
}

func (r Renderer) assetURL(path string) string {
	values := url.Values{"path": []string{path}}
	if r.AssetToken != "" {
		values.Set("token", r.AssetToken)
	}
	if r.AssetSessionID != "" {
		values.Set("session", r.AssetSessionID)
	}
	if r.AssetBufferID != "" {
		values.Set("buffer", r.AssetBufferID)
	}
	return "/asset?" + values.Encode()
}
