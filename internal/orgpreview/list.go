package orgpreview

import (
	"fmt"
	"strings"
)

const listTabWidth = 8

type parsedListItem struct {
	indent int
	node   Node
}

func listIndentWidth(leading string) int {
	column := 0
	for _, char := range leading {
		if char == '\t' {
			column += listTabWidth - column%listTabWidth
		} else {
			column++
		}
	}
	return column
}

func buildList(items []parsedListItem, ids map[string]int) Node {
	depths := listDepths(items)
	index := 0
	return buildListLevel(items, depths, &index, 0, ids)
}

func listDepths(items []parsedListItem) []int {
	if len(items) == 0 {
		return nil
	}
	levels := []int{items[0].indent}
	depths := make([]int, len(items))
	for i := 1; i < len(items); i++ {
		indent := items[i].indent
		current := levels[len(levels)-1]
		switch {
		case indent > current:
			levels = append(levels, indent)
		case indent < current:
			for len(levels) > 0 && levels[len(levels)-1] > indent {
				levels = levels[:len(levels)-1]
			}
			if len(levels) == 0 {
				levels = append(levels, indent)
			} else if levels[len(levels)-1] < indent {
				levels = append(levels, indent)
			}
		}
		depths[i] = len(levels) - 1
	}
	return depths
}

func buildListLevel(items []parsedListItem, depths []int, index *int, depth int, ids map[string]int) Node {
	start := items[*index].node.Range.Start
	children := []Node{}
	end := start
	for *index < len(items) && depths[*index] == depth {
		item := items[*index].node
		end = item.Range.End
		(*index)++
		if *index < len(items) && depths[*index] > depth {
			nested := buildListLevel(items, depths, index, depth+1, ids)
			item.Children = append(item.Children, nested)
			end = nested.Range.End
		}
		children = append(children, item)
	}
	return Node{
		Kind:     "block",
		Type:     "list",
		ID:       nextID(ids, "list", fmt.Sprintf("%d:%d:%d", depth, len(children), start)),
		Range:    SourceRange{Start: start, End: end},
		Attrs:    map[string]string{"depth": fmt.Sprintf("%d", depth)},
		Children: children,
	}
}

func splitListItemChildren(children []Node) (inline, nested []Node) {
	for _, child := range children {
		if child.Type == "list" {
			nested = append(nested, child)
		} else {
			inline = append(inline, child)
		}
	}
	return inline, nested
}

func checkboxLabel(state string) string {
	switch state {
	case "unchecked":
		return "Unchecked"
	case "partial":
		return "Partially checked"
	case "checked":
		return "Checked"
	default:
		return ""
	}
}

func checkboxGlyph(state string) string {
	switch state {
	case "unchecked":
		return "[ ]"
	case "partial":
		return "[-]"
	case "checked":
		return "[X]"
	default:
		return ""
	}
}

func normalizedCheckboxState(mark string) string {
	switch strings.ToLower(mark) {
	case " ":
		return "unchecked"
	case "-":
		return "partial"
	case "x":
		return "checked"
	default:
		return ""
	}
}
