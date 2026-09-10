package orgpreview

import (
	"crypto/sha1"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type Parser interface {
	Parse(text string) (Document, error)
	Name() string
}

type TreeSitterParser struct{}

func (TreeSitterParser) Name() string { return "tree-sitter-org" }

func (TreeSitterParser) Parse(string) (Document, error) {
	return Document{}, ErrTreeSitterOrgUnavailable
}

var ErrTreeSitterOrgUnavailable = fmt.Errorf("tree-sitter org grammar is not vendored")

type FallbackParser struct{}

func (FallbackParser) Name() string { return "fallback-org-v1" }

var (
	headingRE  = regexp.MustCompile(`^(\*+)\s+(.*)$`)
	listItemRE = regexp.MustCompile(`^(\s*)([-+*]|\d+[.)])\s+(.*)$`)
	linkRE     = regexp.MustCompile(`\[\[file:([^\]\n]+)\](?:\[([^\]\n]*)\])?\]`)
)

func (p FallbackParser) Parse(text string) (Document, error) {
	lines := splitLines(text)
	ids := map[string]int{}
	nodes := make([]Node, 0)
	for i := 0; i < len(lines); {
		line := lines[i]
		trim := strings.TrimRight(line.Text, "\r\n")
		if strings.TrimSpace(trim) == "" {
			i++
			continue
		}
		if match := headingRE.FindStringSubmatch(trim); match != nil {
			level := len(match[1])
			title := match[2]
			node := Node{
				Kind:  "block",
				Type:  "heading",
				ID:    nextID(ids, "heading", fmt.Sprintf("%d:%s", level, normalizeIDText(title))),
				Range: SourceRange{Start: line.Start, End: line.End},
				Attrs: map[string]string{"level": fmt.Sprintf("%d", level)},
				Text:  title,
			}
			node.Children = parseInline(title, line.Start+level+1, ids)
			nodes = append(nodes, node)
			i++
			continue
		}
		lower := strings.ToLower(strings.TrimSpace(trim))
		if strings.HasPrefix(lower, "#+begin_src") || strings.HasPrefix(lower, "#+begin_example") {
			start := line.Start
			var builder strings.Builder
			info := strings.TrimSpace(trim)
			end := line.End
			i++
			for i < len(lines) {
				current := strings.TrimRight(lines[i].Text, "\r\n")
				currentLower := strings.ToLower(strings.TrimSpace(current))
				end = lines[i].End
				if strings.HasPrefix(currentLower, "#+end_src") || strings.HasPrefix(currentLower, "#+end_example") {
					i++
					break
				}
				builder.WriteString(lines[i].Text)
				i++
			}
			nodes = append(nodes, Node{
				Kind:  "block",
				Type:  "code_block",
				ID:    nextID(ids, "code_block", normalizeIDText(builder.String())),
				Range: SourceRange{Start: start, End: end},
				Attrs: map[string]string{"info": info},
				Text:  strings.TrimRight(builder.String(), "\r\n"),
			})
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(trim), "|") {
			start := line.Start
			tableLines := []lineSpan{}
			for i < len(lines) {
				current := strings.TrimRight(lines[i].Text, "\r\n")
				if !strings.HasPrefix(strings.TrimSpace(current), "|") {
					break
				}
				tableLines = append(tableLines, lines[i])
				i++
			}
			children := parseTable(tableLines, ids)
			end := tableLines[len(tableLines)-1].End
			nodes = append(nodes, Node{
				Kind:     "block",
				Type:     "table",
				ID:       nextID(ids, "table", fmt.Sprintf("%d:%d", len(tableLines), start)),
				Range:    SourceRange{Start: start, End: end},
				Children: children,
			})
			continue
		}
		if listItemRE.MatchString(trim) && !headingRE.MatchString(trim) {
			start := line.Start
			items := []Node{}
			for i < len(lines) {
				current := strings.TrimRight(lines[i].Text, "\r\n")
				match := listItemRE.FindStringSubmatch(current)
				if match == nil || headingRE.MatchString(current) {
					break
				}
				contentStart := lines[i].Start + len(match[1]) + len(match[2]) + 1
				item := Node{
					Kind:  "block",
					Type:  "list_item",
					ID:    nextID(ids, "list_item", normalizeIDText(match[3])),
					Range: SourceRange{Start: lines[i].Start, End: lines[i].End},
					Attrs: map[string]string{"marker": match[2]},
					Text:  match[3],
				}
				item.Children = parseInline(match[3], contentStart, ids)
				items = append(items, item)
				i++
			}
			end := items[len(items)-1].Range.End
			nodes = append(nodes, Node{
				Kind:     "block",
				Type:     "list",
				ID:       nextID(ids, "list", fmt.Sprintf("%d:%d", len(items), start)),
				Range:    SourceRange{Start: start, End: end},
				Children: items,
			})
			continue
		}
		start := line.Start
		var paragraph strings.Builder
		end := line.End
		for i < len(lines) {
			current := strings.TrimRight(lines[i].Text, "\r\n")
			currentTrim := strings.TrimSpace(current)
			currentLower := strings.ToLower(currentTrim)
			if currentTrim == "" || headingRE.MatchString(current) || strings.HasPrefix(currentTrim, "|") ||
				(strings.HasPrefix(currentLower, "#+begin_src") || strings.HasPrefix(currentLower, "#+begin_example")) ||
				(listItemRE.MatchString(current) && !headingRE.MatchString(current)) {
				break
			}
			if paragraph.Len() > 0 {
				paragraph.WriteByte('\n')
			}
			paragraph.WriteString(current)
			end = lines[i].End
			i++
		}
		paragraphText := paragraph.String()
		node := Node{
			Kind:  "block",
			Type:  "paragraph",
			ID:    nextID(ids, "paragraph", normalizeIDText(paragraphText)),
			Range: SourceRange{Start: start, End: end},
			Text:  paragraphText,
		}
		node.Children = parseInline(paragraphText, start, ids)
		nodes = append(nodes, node)
	}
	return NewDocument(p.Name(), text, nodes), nil
}

type lineSpan struct {
	Text       string
	Start, End int
}

func splitLines(text string) []lineSpan {
	if text == "" {
		return nil
	}
	lines := []lineSpan{}
	start := 0
	for start < len(text) {
		end := strings.IndexByte(text[start:], '\n')
		if end == -1 {
			lines = append(lines, lineSpan{Text: text[start:], Start: start, End: len(text)})
			break
		}
		end = start + end + 1
		lines = append(lines, lineSpan{Text: text[start:end], Start: start, End: end})
		start = end
	}
	return lines
}

func parseTable(lines []lineSpan, ids map[string]int) []Node {
	rows := make([]Node, 0, len(lines))
	for _, line := range lines {
		text := strings.TrimRight(line.Text, "\r\n")
		if isTableSeparator(text) {
			continue
		}
		rawCells := strings.Split(strings.Trim(text, "|"), "|")
		cells := make([]Node, 0, len(rawCells))
		cursor := line.Start + strings.Index(text, "|") + 1
		for _, raw := range rawCells {
			cellText := strings.TrimSpace(raw)
			cellStart := cursor + leadingSpaces(raw)
			cellEnd := cellStart + len(cellText)
			cell := Node{
				Kind:  "block",
				Type:  "table_cell",
				ID:    nextID(ids, "table_cell", normalizeIDText(cellText)),
				Range: SourceRange{Start: cellStart, End: cellEnd},
				Text:  cellText,
			}
			cell.Children = parseInline(cellText, cellStart, ids)
			cells = append(cells, cell)
			cursor += len(raw) + 1
		}
		rows = append(rows, Node{
			Kind:     "block",
			Type:     "table_row",
			ID:       nextID(ids, "table_row", normalizeIDText(text)),
			Range:    SourceRange{Start: line.Start, End: line.End},
			Children: cells,
		})
	}
	return rows
}

func isTableSeparator(line string) bool {
	trim := strings.TrimSpace(line)
	if trim == "" {
		return false
	}
	for _, r := range trim {
		if r != '|' && r != '-' && r != '+' && r != ' ' {
			return false
		}
	}
	return strings.Contains(trim, "-")
}

func parseInline(text string, base int, ids map[string]int) []Node {
	var nodes []Node
	emitText := func(start, end int) {
		if end <= start {
			return
		}
		nodes = append(nodes, Node{
			Kind:  "span",
			Type:  "text",
			ID:    nextID(ids, "text", normalizeIDText(text[start:end])),
			Range: SourceRange{Start: base + start, End: base + end},
			Text:  text[start:end],
		})
	}
	pos := 0
	for _, loc := range linkRE.FindAllStringSubmatchIndex(text, -1) {
		emitText(pos, loc[0])
		target := text[loc[2]:loc[3]]
		label := target
		if loc[4] >= 0 {
			label = text[loc[4]:loc[5]]
		}
		linkType := "link"
		if isImageTarget(target) {
			linkType = "image"
		}
		nodes = append(nodes, Node{
			Kind:  "span",
			Type:  linkType,
			ID:    nextID(ids, linkType, normalizeIDText(target)+":"+normalizeIDText(label)),
			Range: SourceRange{Start: base + loc[0], End: base + loc[1]},
			Attrs: map[string]string{"target": target, "label": label},
			Text:  label,
		})
		pos = loc[1]
	}
	emitText(pos, len(text))
	return parseMarkup(nodes, ids)
}

func parseMarkup(nodes []Node, ids map[string]int) []Node {
	out := []Node{}
	for _, node := range nodes {
		if node.Type != "text" {
			out = append(out, node)
			continue
		}
		out = append(out, splitDelimited(node, "*", "strong", ids)...)
	}
	final := []Node{}
	for _, node := range out {
		if node.Type != "text" {
			final = append(final, node)
			continue
		}
		final = append(final, splitDelimited(node, "/", "emphasis", ids)...)
	}
	out = []Node{}
	for _, node := range final {
		if node.Type != "text" {
			out = append(out, node)
			continue
		}
		out = append(out, splitDelimited(node, "~", "code", ids)...)
	}
	return out
}

func splitDelimited(node Node, delim, typ string, ids map[string]int) []Node {
	text := node.Text
	start := strings.Index(text, delim)
	if start == -1 {
		return []Node{node}
	}
	end := strings.Index(text[start+1:], delim)
	if end == -1 {
		return []Node{node}
	}
	end = start + 1 + end
	var result []Node
	if start > 0 {
		result = append(result, Node{Kind: "span", Type: "text", ID: nextID(ids, "text", normalizeIDText(text[:start])), Range: SourceRange{Start: node.Range.Start, End: node.Range.Start + start}, Text: text[:start]})
	}
	inner := text[start+1 : end]
	result = append(result, Node{Kind: "span", Type: typ, ID: nextID(ids, typ, normalizeIDText(inner)), Range: SourceRange{Start: node.Range.Start + start, End: node.Range.Start + end + 1}, Text: inner})
	if end+1 < len(text) {
		result = append(result, Node{Kind: "span", Type: "text", ID: nextID(ids, "text", normalizeIDText(text[end+1:])), Range: SourceRange{Start: node.Range.Start + end + 1, End: node.Range.End}, Text: text[end+1:]})
	}
	return result
}

func isImageTarget(target string) bool {
	switch strings.ToLower(filepath.Ext(target)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg":
		return true
	default:
		return false
	}
}

func nextID(counts map[string]int, typ, text string) string {
	sum := sha1.Sum([]byte(typ + "\x00" + text))
	base := fmt.Sprintf("%s-%x", strings.ReplaceAll(typ, "_", "-"), sum[:5])
	counts[base]++
	if counts[base] == 1 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, counts[base])
}

func leadingSpaces(s string) int {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return i
		}
	}
	return len(s)
}
