package orgpreview

import (
	"fmt"
	"regexp"
	"strings"
)

// MarkdownParser parses the common Markdown constructs used by the live
// preview without adding a runtime dependency.  It produces the same IR as the
// Org parser so navigation, the outline, and heading colors stay shared.
type MarkdownParser struct{}

func (MarkdownParser) Name() string { return "fallback-markdown-v1" }

var (
	markdownHeadingRE = regexp.MustCompile(`^(#{1,6})\s+(.*?)(?:\s+#+)?$`)
	markdownListRE    = regexp.MustCompile(`^(\s*)([-+*]|\d+[.)])\s+(.*)$`)
	markdownLinkRE    = regexp.MustCompile(`!?\[([^]\n]*)\]\(([^)\n]+)\)`)
)

func (p MarkdownParser) Parse(text string) (Document, error) {
	lines, ids, nodes := splitLines(text), map[string]int{}, []Node{}
	for i := 0; i < len(lines); {
		line, trim := lines[i], strings.TrimRight(lines[i].Text, "\r\n")
		if strings.TrimSpace(trim) == "" {
			i++
			continue
		}
		if match := markdownHeadingRE.FindStringSubmatch(trim); match != nil {
			level, title := len(match[1]), match[2]
			node := Node{Kind: "block", Type: "heading", ID: nextID(ids, "heading", fmt.Sprintf("%d:%s", level, normalizeIDText(title))), Range: SourceRange{Start: line.Start, End: line.End}, Attrs: map[string]string{"level": fmt.Sprintf("%d", level)}, Text: title}
			node.Children = parseMarkdownInline(title, line.Start+level+1, ids)
			nodes, i = append(nodes, node), i+1
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(trim), "```") || strings.HasPrefix(strings.TrimSpace(trim), "~~~") {
			start, fence, info, end := line.Start, strings.TrimSpace(trim)[:3], strings.TrimSpace(strings.TrimSpace(trim)[3:]), line.End
			var body strings.Builder
			i++
			for i < len(lines) {
				current := strings.TrimRight(lines[i].Text, "\r\n")
				end = lines[i].End
				if strings.HasPrefix(strings.TrimSpace(current), fence) {
					i++
					break
				}
				body.WriteString(lines[i].Text)
				i++
			}
			nodes = append(nodes, Node{Kind: "block", Type: "code_block", ID: nextID(ids, "code_block", normalizeIDText(body.String())), Range: SourceRange{Start: start, End: end}, Attrs: map[string]string{"info": info}, Text: strings.TrimRight(body.String(), "\r\n")})
			continue
		}
		if markdownListRE.MatchString(trim) {
			items := []parsedListItem{}
			for i < len(lines) {
				current := strings.TrimRight(lines[i].Text, "\r\n")
				match := markdownListRE.FindStringSubmatchIndex(current)
				if match == nil {
					break
				}
				leading := current[match[2]:match[3]]
				marker := current[match[4]:match[5]]
				content := current[match[6]:match[7]]
				base := lines[i].Start + match[6]
				item := Node{Kind: "block", Type: "list_item", ID: nextID(ids, "list_item", normalizeIDText(content)), Range: SourceRange{Start: lines[i].Start, End: lines[i].End}, Attrs: map[string]string{"marker": marker}, Text: content}
				item.Children = parseMarkdownInline(content, base, ids)
				items = append(items, parsedListItem{indent: listIndentWidth(leading), node: item})
				i++
			}
			nodes = append(nodes, buildList(items, ids))
			continue
		}
		start, end, paragraph := line.Start, line.End, strings.Builder{}
		for i < len(lines) {
			current := strings.TrimRight(lines[i].Text, "\r\n")
			if strings.TrimSpace(current) == "" || markdownHeadingRE.MatchString(current) || markdownListRE.MatchString(current) || strings.HasPrefix(strings.TrimSpace(current), "```") || strings.HasPrefix(strings.TrimSpace(current), "~~~") {
				break
			}
			if paragraph.Len() > 0 {
				paragraph.WriteByte('\n')
			}
			paragraph.WriteString(current)
			end = lines[i].End
			i++
		}
		body := paragraph.String()
		node := Node{Kind: "block", Type: "paragraph", ID: nextID(ids, "paragraph", normalizeIDText(body)), Range: SourceRange{Start: start, End: end}, Text: body}
		node.Children = parseMarkdownInline(body, start, ids)
		nodes = append(nodes, node)
	}
	return NewDocument(p.Name(), text, nodes), nil
}

func parseMarkdownInline(text string, base int, ids map[string]int) []Node {
	nodes, pos := []Node{}, 0
	for _, loc := range markdownLinkRE.FindAllStringSubmatchIndex(text, -1) {
		if loc[0] > pos {
			nodes = append(nodes, Node{Kind: "span", Type: "text", ID: nextID(ids, "text", normalizeIDText(text[pos:loc[0]])), Range: SourceRange{Start: base + pos, End: base + loc[0]}, Text: text[pos:loc[0]]})
		}
		image, label, target := text[loc[0]] == '!', text[loc[2]:loc[3]], text[loc[4]:loc[5]]
		typ := "link"
		if image {
			typ = "image"
		}
		nodes = append(nodes, Node{Kind: "span", Type: typ, ID: nextID(ids, typ, normalizeIDText(target)), Range: SourceRange{Start: base + loc[0], End: base + loc[1]}, Attrs: map[string]string{"target": target, "label": label}, Text: label})
		pos = loc[1]
	}
	if pos < len(text) {
		nodes = append(nodes, Node{Kind: "span", Type: "text", ID: nextID(ids, "text", normalizeIDText(text[pos:])), Range: SourceRange{Start: base + pos, End: base + len(text)}, Text: text[pos:]})
	}
	return nodes
}
