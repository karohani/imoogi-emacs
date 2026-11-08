// Package orgrender turns a validated clip AST into an Org document.
//
// Security model: only this package's own code may emit an Org line that
// starts with `#+`, `*`, `|` or `:PROPERTIES:`, and the only `file:` links it
// emits point at assets it was told were saved. Content never reaches the
// output except through escapeText (parsed contexts), escapeLineStart (line
// starts), encodeHref (link targets) or the comma-escaped example block.
package orgrender

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/karohani/obsidian-clipper/native-host/internal/protocol"
)

// Doc is the renderer input.
type Doc struct {
	Title     string
	SourceURL string
	ClippedAt time.Time
	Blocks    []protocol.Block
	// Assets maps an asset id to the saved file name inside assets/. An image
	// whose id is absent renders as plain text, never as a link.
	Assets map[string]string
	// AssetDir is the document-relative directory holding the saved images
	// (e.g. "assets/2026-11-08-title"); empty means "assets".
	AssetDir string
}

const imageNotSaved = "(image not saved)"

var assetNameRe = regexp.MustCompile(`^[0-9a-f]{32}\.(png|jpg|webp)$`)

type mode int

const (
	modePara mode = iota // br becomes an Org line break
	modeLine             // single line: br becomes a space
	modeDesc             // link description: single line, images deferred
)

type renderer struct {
	assets   map[string]string
	assetDir string
}

// Render produces the complete index.org text.
//
// @MX:ANCHOR: [AUTO] sole producer of index.org content
// @MX:REASON: the store writes exactly this string; any construct emitted here is trusted by Org on open
func Render(d Doc) string {
	r := &renderer{assets: d.Assets, assetDir: "assets"}
	if d.AssetDir != "" {
		r.assetDir = d.AssetDir
	}
	var out strings.Builder
	out.WriteString("#+TITLE: " + escapeText(protocol.NormalizeLine(d.Title)) + "\n")
	out.WriteString("#+DATE: " + d.ClippedAt.Format("[2006-01-02 Mon 15:04]") + "\n")
	src := protocol.NormalizeLine(d.SourceURL)
	target, isHTTP := safeHref(src)
	if !isHTTP {
		target = encodeHref(src)
	}
	// #+SOURCE is a custom keyword: Org neither exports nor evaluates it, and
	// its value is not parsed for objects. The URL is percent-encoded so it
	// stays a single token.
	out.WriteString("#+SOURCE: " + target + "\n")
	if isHTTP {
		out.WriteString("\nSource: [[" + target + "][" + fullwidthBrackets(escapeText(src)) + "]]\n")
	}
	for _, lines := range r.blocks(d.Blocks, ctx{}) {
		out.WriteString("\n")
		for _, l := range lines {
			out.WriteString(l + "\n")
		}
	}
	return breakEmacsTokens(out.String())
}

// ctx describes where a block sequence is being rendered.
type ctx struct {
	nested  bool // inside a list item or quote: no headlines possible
	inQuote bool // nested quotes are flattened
}

// blocks renders a sequence, one []string of lines per non-empty block.
func (r *renderer) blocks(bs []protocol.Block, c ctx) [][]string {
	var out [][]string
	for _, b := range bs {
		if b.T == "quote" && c.inQuote {
			out = append(out, r.blocks(b.Blocks, c)...)
			continue
		}
		if lines := r.block(b, c); len(lines) > 0 {
			out = append(out, lines)
		}
	}
	return out
}

func (r *renderer) block(b protocol.Block, c ctx) []string {
	switch b.T {
	case "heading":
		if c.nested {
			return r.para(b.Inl)
		}
		text := r.inlines(b.Inl, modeLine, nil)
		if strings.HasSuffix(text, ":") {
			text += zwsp // never let a heading end in a tag group
		}
		// The leading ZWSP keeps content from supplying a TODO keyword,
		// COMMENT, or a priority cookie.
		return []string{strings.Repeat("*", b.Level) + " " + zwsp + text}
	case "para":
		return r.para(b.Inl)
	case "list":
		return r.list(b)
	case "quote":
		inner := r.blocks(b.Blocks, ctx{nested: true, inQuote: true})
		return append(append([]string{"#+begin_quote"}, joinBlocks(inner)...), "#+end_quote")
	case "table":
		return r.table(b)
	case "code":
		return codeBlock(b.Text)
	case "image":
		return []string{r.image(b.Asset)}
	case "hr":
		return []string{"-----"}
	}
	return nil // unreachable for validated input
}

// joinBlocks flattens per-block lines with one blank line between blocks.
func joinBlocks(bs [][]string) []string {
	var out []string
	for i, lines := range bs {
		if i > 0 {
			out = append(out, "")
		}
		out = append(out, lines...)
	}
	return out
}

func (r *renderer) para(ins []protocol.Inline) []string {
	text := r.inlines(ins, modePara, nil)
	var lines []string
	for _, l := range strings.Split(text, "\n") {
		lines = append(lines, escapeLineStart(l))
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	return lines
}

func (r *renderer) list(b protocol.Block) []string {
	var out []string
	for i, item := range b.Items {
		bullet := "-"
		if b.Ordered {
			bullet = strconv.Itoa(i+1) + "."
		}
		cont := strings.Repeat(" ", len(bullet)+1)
		rendered := r.blocks(item, ctx{nested: true})
		if len(rendered) == 0 {
			out = append(out, bullet)
			continue
		}
		start := 0
		if item[0].T == "para" {
			out = append(out, bullet+" "+rendered[0][0])
			for _, l := range rendered[0][1:] {
				out = append(out, indent(cont, l))
			}
			start = 1
		} else {
			out = append(out, bullet)
		}
		for j, lines := range rendered[start:] {
			if j > 0 || start == 1 {
				out = append(out, "")
			}
			for _, l := range lines {
				out = append(out, indent(cont, l))
			}
		}
	}
	return out
}

func indent(prefix, l string) string {
	if l == "" {
		return ""
	}
	return prefix + l
}

func (r *renderer) table(b protocol.Block) []string {
	width := 0
	for _, row := range b.Rows {
		width = max(width, len(row))
	}
	if width == 0 {
		return nil
	}
	var out []string
	for i, row := range b.Rows {
		cells := make([]string, width)
		for j, cell := range row {
			text := strings.TrimSpace(r.inlines(cell, modeLine, nil))
			cells[j] = strings.ReplaceAll(text, "|", `\vert{}`)
		}
		out = append(out, "| "+strings.Join(cells, " | ")+" |")
		if i == 0 && b.Header && len(b.Rows) > 1 {
			out = append(out, "|"+strings.Repeat("---+", width-1)+"---|")
		}
	}
	// The caller always follows a block with a blank line, so no content line
	// can ever sit directly under the table (where #+TBLFM would bind).
	return out
}

var commaEscapeRe = regexp.MustCompile(`(?m)^([ \t]*)(,*(?:\*|#\+))`)

// codeBlock renders code as an example block (never src, never header args).
// Lines Org would read as a headline or a keyword are comma-escaped the way
// Org itself does it, which also neutralises an inner #+end_example.
func codeBlock(text string) []string {
	text = strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(text)
	var b strings.Builder
	for _, r := range text {
		if r == '\n' || r == '\t' || !isLineBreakOrControl(r) {
			b.WriteRune(r)
		}
	}
	text = commaEscapeRe.ReplaceAllString(b.String(), "$1,$2")
	text = strings.TrimRight(text, "\n")
	out := []string{"#+begin_example"}
	out = append(out, strings.Split(text, "\n")...)
	return append(out, "#+end_example")
}

func (r *renderer) assetName(id string) (string, bool) {
	name, ok := r.assets[id]
	if !ok || !assetNameRe.MatchString(name) {
		return "", false
	}
	return name, true
}

func (r *renderer) image(id string) string {
	if name, ok := r.assetName(id); ok {
		return "[[file:" + r.assetDir + "/" + name + "]]"
	}
	return imageNotSaved
}

// inlines renders an inline sequence. Consecutive text nodes are merged before
// escaping so a token cannot be assembled across node boundaries. In modeDesc,
// saved images are collected into deferred instead of being emitted.
func (r *renderer) inlines(ins []protocol.Inline, m mode, deferred *[]string) string {
	var out, run strings.Builder
	flush := func() {
		if run.Len() > 0 {
			out.WriteString(escapeText(run.String()))
			run.Reset()
		}
	}
	child := m
	if m == modePara {
		child = modeLine
	}
	for _, in := range ins {
		switch in.T {
		case "text":
			run.WriteString(in.V)
		case "br":
			if m == modePara {
				flush()
				out.WriteString(`\\` + "\n")
			} else {
				run.WriteString(" ")
			}
		case "bold", "italic", "strike":
			flush()
			out.WriteString(emphasis(markerFor(in.T), r.inlines(in.C, child, deferred)))
		case "code":
			flush()
			out.WriteString(inlineCode(in.V))
		case "link":
			flush()
			out.WriteString(r.link(in, m, deferred))
		case "image":
			flush()
			if name, ok := r.assetName(in.Asset); ok && m == modeDesc {
				*deferred = append(*deferred, name)
				out.WriteString("(image)")
			} else {
				out.WriteString(r.image(in.Asset))
			}
		}
	}
	flush()
	return out.String()
}

func markerFor(t string) rune {
	switch t {
	case "bold":
		return '*'
	case "italic":
		return '/'
	}
	return '+'
}

// emphasis wraps inner in markers only when Org would parse exactly that
// span: non-empty, no edge whitespace, no marker inside. The ZWSP on both
// sides is valid Org PRE/POST context, so the markup works mid-word.
func emphasis(marker rune, inner string) string {
	if !emphasisSafe(inner, marker) {
		return inner
	}
	m := string(marker)
	return zwsp + m + inner + m + zwsp
}

func emphasisSafe(inner string, marker rune) bool {
	if inner == "" || strings.ContainsRune(inner, marker) || strings.ContainsRune(inner, '\n') {
		return false
	}
	first, _ := utf8.DecodeRuneInString(inner)
	last, _ := utf8.DecodeLastRuneInString(inner)
	return !isEdge(first) && !isEdge(last)
}

func isEdge(r rune) bool { return unicode.IsSpace(r) || r == '​' || isLineBreakOrControl(r) }

// inlineCode emits ~verbatim~ (Org parses nothing inside it) or falls back to
// escaped plain text when the content could not be represented safely.
func inlineCode(v string) string {
	clean := strings.Map(func(r rune) rune {
		if isLineBreakOrControl(r) {
			return ' '
		}
		return r
	}, v)
	if !emphasisSafe(clean, '~') {
		return escapeText(clean)
	}
	return zwsp + "~" + clean + "~" + zwsp
}

func (r *renderer) link(in protocol.Inline, m mode, deferred *[]string) string {
	target, ok := safeHref(in.Href)
	if m == modeDesc || !ok {
		// Nested links collapse to their description; non-http(s) links
		// render as the description text only.
		return r.inlines(in.C, m, deferred)
	}
	var imgs []string
	desc := fullwidthBrackets(r.inlines(in.C, modeDesc, &imgs))
	s := "[[" + target + "]"
	if strings.TrimSpace(strings.ReplaceAll(desc, zwsp, "")) != "" {
		s += "[" + desc + "]"
	}
	s += "]"
	for _, name := range imgs {
		s += " [[file:" + r.assetDir + "/" + name + "]]"
	}
	return s
}
