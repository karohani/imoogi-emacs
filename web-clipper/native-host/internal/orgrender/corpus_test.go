package orgrender

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/karohani/obsidian-clipper/native-host/internal/protocol"
)

// --- AST builders -----------------------------------------------------------

func tx(v string) protocol.Inline { return protocol.Inline{T: "text", V: v} }
func br() protocol.Inline         { return protocol.Inline{T: "br"} }
func bold(c ...protocol.Inline) protocol.Inline {
	return protocol.Inline{T: "bold", C: c}
}
func italic(c ...protocol.Inline) protocol.Inline {
	return protocol.Inline{T: "italic", C: c}
}
func strike(c ...protocol.Inline) protocol.Inline {
	return protocol.Inline{T: "strike", C: c}
}
func icode(v string) protocol.Inline { return protocol.Inline{T: "code", V: v} }
func link(href string, c ...protocol.Inline) protocol.Inline {
	return protocol.Inline{T: "link", Href: href, C: c}
}
func iimg(id string) protocol.Inline { return protocol.Inline{T: "image", Asset: id} }

func para(c ...protocol.Inline) protocol.Block { return protocol.Block{T: "para", Inl: c} }
func heading(level int, c ...protocol.Inline) protocol.Block {
	return protocol.Block{T: "heading", Level: level, Inl: c}
}
func code(text string) protocol.Block { return protocol.Block{T: "code", Text: text} }
func quote(b ...protocol.Block) protocol.Block {
	return protocol.Block{T: "quote", Blocks: b}
}
func list(ordered bool, items ...[]protocol.Block) protocol.Block {
	return protocol.Block{T: "list", Ordered: ordered, Items: items}
}
func item(b ...protocol.Block) []protocol.Block { return b }
func table(header bool, rows ...[][]protocol.Inline) protocol.Block {
	return protocol.Block{T: "table", Header: header, Rows: rows}
}
func row(cells ...string) [][]protocol.Inline {
	out := make([][]protocol.Inline, len(cells))
	for i, c := range cells {
		out[i] = []protocol.Inline{tx(c)}
	}
	return out
}
func bimg(id string) protocol.Block { return protocol.Block{T: "image", Asset: id} }
func hr() protocol.Block            { return protocol.Block{T: "hr"} }

var clipTime = time.Date(2026, 11, 8, 10, 0, 0, 0, time.FixedZone("KST", 9*3600))

// --- Malicious corpus -------------------------------------------------------

// corpusCase is one hostile document plus the structure the renderer is
// expected to produce from it (headline and table-line counts are the only
// places where the renderer itself emits `*` and `|` line starts).
type corpusCase struct {
	name       string
	doc        Doc
	headlines  int
	tableLines int
}

// maliciousCorpus builds the corpus. cmd and lisp are the payloads embedded in
// shell- and lisp-shaped attacks: the golden tests use the literal hostile
// strings from the threat list; the Emacs-backed test substitutes harmless
// sentinels so that a renderer bug can be observed without doing damage.
func maliciousCorpus(cmd, lisp, includePath string) []corpusCase {
	d := func(title string, blocks ...protocol.Block) Doc {
		return Doc{Title: title, SourceURL: "https://example.com/post?id=1", ClippedAt: clipTime, Blocks: blocks}
	}
	return []corpusCase{
		{name: "headline-and-todo", doc: d("t",
			para(tx("* TODO pwn")),
			para(tx("intro"), br(), tx("** DONE nested"), br(), tx("   * indented bullet")),
			para(tx("*bold-looking* /italic/ +strike+ =verbatim= ~code~ _under_")),
			heading(1, tx("TODO COMMENT [#A] real heading :tag:noexport:")),
		), headlines: 1},
		{name: "blocks-in-paragraph", doc: d("t",
			para(tx("#+begin_src sh :results output")),
			para(tx(cmd)),
			para(tx("#+end_src")),
			para(tx("#+end_example"), br(), tx("#+begin_example"), br(), tx("#+end_quote")),
			para(tx("#+CALL: x()"), br(), tx("# comment line"), br(), tx("#+BEGIN_SRC emacs-lisp"), br(), tx(lisp)),
		)},
		{name: "blocks-in-code", doc: d("t",
			code("#+end_example\n#+begin_src sh :results output\n"+cmd+"\n#+end_src\n* TODO inside code\n,#+already\n  #+indented\n:PROPERTIES:\n#+end_quote"),
			quote(code("#+end_quote\n#+end_example"), para(tx("#+end_quote"))),
		)},
		{name: "drawers-and-keywords", doc: d("t",
			heading(2, tx("section")),
			para(tx(":PROPERTIES:"), br(), tx(":header-args: :eval yes"), br(), tx(":END:")),
			para(tx("SCHEDULED: <2026-11-09 Mon> DEADLINE: <2026-11-10 Tue>")),
			para(tx("CLOCK: [2026-11-08 Sun 10:00]")),
			para(tx("#+PROPERTY: header-args :eval yes")),
			para(tx("#+INCLUDE: \""+includePath+"\"")),
			para(tx("#+SETUPFILE: https://evil.example/setup.org"), br(), tx("#+STARTUP: showall"), br(), tx("#+BIND: org-export-x t")),
			para(tx(": fixed width"), br(), tx("%%(diary-sexp "+lisp+")")),
		), headlines: 1},
		{name: "tblfm", doc: d("t",
			table(true, row("a", "#+TBLFM: $1=1"), row("b", "c")),
			para(tx("#+TBLFM: @2$1='("+lisp+")")),
			table(false, row("x|y", "multi\nline\r\ncell", "|---+---|")),
		), tableLines: 4},
		{name: "inline-objects", doc: d("t",
			para(tx("{{{macro}}} {{{keyword(TITLE)}}} {{"), tx("{split}}}")),
			para(tx("src_sh{"+cmd+"} SRC_sh{x} sr"), tx("c_sh{"+cmd+"} call_x() CALL_y[:results raw]()")),
			para(tx("<<target>> <<<radio>>> [fn:1] [fn::inline] @@html:<script>@@ \\(x\\) \\[y\\] $$z$$ \\alpha \\\\")),
			para(tx("<%%("+lisp+")> <2026-11-08 Sun> [cite:@key] [[shell:"+cmd+"]] [[elisp:"+lisp+"]]")),
			para(tx("plain shell:"+cmd+" elisp:"+lisp+" file:/etc/passwd https://plain.example/")),
			para(tx("["), tx("[elisp:"+lisp+"]]")),
		)},
		{name: "local-variables", doc: d("t",
			para(tx("# Local Variables:"), br(), tx("# eval: (shell-command \""+cmd+"\")"), br(), tx("# End:")),
			para(tx("Local "), tx("Variables: split across nodes")),
			code(";; Local Variables:\n;; eval: "+lisp+"\n;; End:"),
			para(tx("trailing -*- eval: "+lisp+" -*-")),
		)},
		{name: "mode-line-title", doc: d("-*- eval: " + lisp + " -*-")},
		{name: "links", doc: d("t",
			para(link("javascript:alert(1)", tx("js"))),
			para(link("elisp:"+lisp, tx("click me"))),
			para(link("file:/etc/passwd", tx("passwd"))),
			para(link("shell:"+cmd, tx("sh"))),
			para(link("mailto:a@example.com", tx("mail"))),
			para(link("/relative/path", tx("rel"))),
			para(link("https://example.com/a]]b[[elisp:"+lisp+"]] c", tx("desc ]] [[elisp:"+lisp+"]]"))),
			para(link("HTTPS://Example.com/x\\", tx("trailing backslash"))),
			para(link("https://example.com/", bold(tx("bold")), tx(" desc"), iimg("a1"))),
			para(link("https://example.com/empty")),
			para(link("https://user:pw@example.com/{{{m}}}<x>")),
		)},
		{name: "title-injection", doc: d("Line one\n#+SETUPFILE: x\r\n* TODO two\t{{{m}}} [[elisp:" + lisp + "]]")},
		{name: "korean-and-long", doc: d(strings.Repeat("긴 제목 ", 125),
			heading(1, tx("한국어 제목")),
			para(tx("안녕하세요, 세계! "), bold(tx("굵게")), tx(" 그리고 "), italic(tx("기울임")), tx(" "), strike(tx("취소")), tx(" "), icode("코드")),
			list(false, item(para(tx("하나"))), item(para(tx("둘")), list(true, item(para(tx("셋")))))),
		), headlines: 1},
		{name: "lists-and-images", doc: Doc{Title: "t", SourceURL: "http://example.com/", ClippedAt: clipTime,
			Assets: map[string]string{"a1": "0123456789abcdef0123456789abcdef.png"},
			Blocks: []protocol.Block{
				list(false,
					item(para(tx("- nested dash"), br(), tx("1. fake ordered"), br(), tx("#+begin_src"))),
					item(code("* in item\n#+end_example"), quote(para(tx("q")), quote(para(tx("inner")))), heading(1, tx("heading in list"))),
					item(),
					item(table(true, row("h"), row("v")), hr(), bimg("a1"), bimg("a2")),
				),
				list(true, item(para(tx("first"))), item(para(tx("second")), list(false, item(para(tx("deep")))))),
				quote(heading(3, tx("quoted heading")), para(tx("- [ ] checkbox"), br(), tx("+ plus"), br(), tx("2026. year"), br(), tx("a) alpha"), br(), tx("-----"))),
				bimg("a1"), bimg("a2"), para(tx("inline "), iimg("a1"), tx(" and "), iimg("a2")),
				hr(),
			}}, tableLines: 3},
	}
}

// --- Safety checker ---------------------------------------------------------

var (
	headerLineRe = []*regexp.Regexp{
		regexp.MustCompile(`^#\+TITLE: `),
		regexp.MustCompile(`^#\+DATE: \[\d{4}-\d{2}-\d{2} [A-Z][a-z]{2} \d{2}:\d{2}\]$`),
		regexp.MustCompile(`^#\+SOURCE: https?://\S+$`),
	}
	headlineRe    = regexp.MustCompile(`^\*+ `)
	planningRe    = regexp.MustCompile(`^(CLOSED|DEADLINE|SCHEDULED|CLOCK):`)
	allowedLinkRe = regexp.MustCompile(`\[\[(?:https?://[^\]\s\[]+|file:assets/[0-9a-f]{32}\.(?:png|jpg|webp))\](?:\[([^\]\[]*)\])?\]`)
	inlineDanger  = []*regexp.Regexp{
		regexp.MustCompile(`\[\[`),
		regexp.MustCompile(`\[fn:`),
		regexp.MustCompile(`\{\{\{`),
		regexp.MustCompile(`(?i)src_`),
		regexp.MustCompile(`(?i)call_`),
		regexp.MustCompile(`<<`),
		regexp.MustCompile(`@@`),
		regexp.MustCompile(`\\[(\[]`),
		regexp.MustCompile(`\$\$`),
		regexp.MustCompile(`\$([^\x{2060}]|$)`), // every content $ must be followed by WORD JOINER
		regexp.MustCompile(`(?i)\b(shell|elisp|file|javascript|help|info|doi|id|attachment|news|mailto|https?):`),
	}
	globalDanger = []*regexp.Regexp{
		regexp.MustCompile(`-\*-`),
		regexp.MustCompile(`(?i)local variables:`),
		regexp.MustCompile(`(?i)(^|[^\p{L}])end:`),
	}
)

// auditOrg scans rendered Org text for any construct only the renderer may
// emit, and for active links whose target is not http(s) or a saved asset.
// It returns the problems found plus the observed headline and table-line
// counts so callers can compare them with what the renderer was asked for.
func auditOrg(out string) (problems []string, headlines, tableLines int) {
	inExample := false
	for i, line := range strings.Split(out, "\n") {
		bad := func(why string) { problems = append(problems, fmt.Sprintf("line %d %s: %q", i+1, why, line)) }
		for _, re := range globalDanger {
			if re.MatchString(line) {
				bad("emacs file-variable token " + re.String())
			}
		}
		trimmed := strings.TrimLeft(line, " \t")
		if inExample {
			switch {
			case trimmed == "#+end_example":
				inExample = false
			case strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "#+"):
				bad("unescaped line inside example block")
			}
			continue
		}
		if i < 3 {
			if !headerLineRe[i].MatchString(line) {
				bad("header line malformed")
			}
			if i == 0 {
				scanInline(strings.TrimPrefix(line, "#+TITLE: "), bad)
			}
			continue
		}
		switch {
		case trimmed == "#+begin_example":
			inExample = true
			continue
		case trimmed == "#+begin_quote" || trimmed == "#+end_quote":
			continue
		case strings.HasPrefix(trimmed, "#"):
			bad("content keyword/comment/block line")
		case strings.HasPrefix(trimmed, ":"):
			bad("drawer or fixed-width line")
		case strings.HasPrefix(trimmed, "%%"):
			bad("diary sexp line")
		case strings.HasPrefix(trimmed, "*"):
			if trimmed == line && headlineRe.MatchString(line) {
				headlines++
			} else {
				bad("star line that is not a renderer headline")
			}
		case strings.HasPrefix(trimmed, "|"):
			tableLines++
		case planningRe.MatchString(trimmed):
			bad("planning/clock line")
		}
		scanInline(line, bad)
	}
	if inExample {
		problems = append(problems, "unterminated example block")
	}
	return problems, headlines, tableLines
}

func scanInline(line string, bad func(string)) {
	rest := allowedLinkRe.ReplaceAllString(line, "$1")
	for _, re := range inlineDanger {
		if re.MatchString(rest) {
			bad("inline construct " + re.String())
		}
	}
}
