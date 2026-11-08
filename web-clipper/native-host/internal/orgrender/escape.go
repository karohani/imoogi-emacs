package orgrender

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// zwsp is U+200B ZERO WIDTH SPACE, Org's own escape character. Emacs gives it
// whitespace syntax, so an emphasis marker followed by it can never open
// markup, and inserting it into a token (`[[`, `{{{`, `src_`, `Local
// Variables:`) breaks the literal match every Org/Emacs scanner relies on.
const zwsp = "\xe2\x80\x8b" // U+200B

// wordJoiner is U+2060 WORD JOINER, used after `$` only (see escapeText).
const wordJoiner = "\xe2\x81\xa0" // U+2060

// escapeText makes arbitrary content inert in any Org context where objects
// are parsed (paragraphs, headings, the title, table cells, link
// descriptions). Control characters and line separators become spaces, so
// the result never contains a newline. Line-start constructs are handled
// separately by escapeLineStart.
//
// Rules (one per character, so a token can never be assembled from two
// adjacent escaped runs):
//   - `[ { < \` are followed by ZWSP: kills `[[` links, `[fn:`, `[cite:`,
//     timestamps, `{{{` macros, `<<` targets, `<%%(` sexps, `\(` `\[` LaTeX,
//     entities and `\\` breaks.
//   - `$` is followed by U+2060 WORD JOINER, not ZWSP: Org closes `$..$` when
//     the closing `$` is followed by a whitespace-syntax char (ZWSP is one);
//     WORD JOINER has word syntax, so no `$` can close math or form `$$`.
//   - emphasis markers `* / + = ~ _` are followed by ZWSP, so none can open
//     emphasis (the opening marker must be followed by non-whitespace).
//   - `_` preceded by `src`/`call` (any case) is also preceded by ZWSP:
//     `src_​x{}` still parses as inline src, `src​_x{}` does not.
//   - `@` followed by `@` gets ZWSP between (export snippets).
//   - `:` right after a letter or digit gets ZWSP before it: kills plain links
//     (`shell:`, `elisp:`), planning keywords, trailing tags, and the Emacs
//     `Local Variables:` / `End:` tokens.
//
// @MX:ANCHOR: [AUTO] the single escaping primitive for all parsed Org contexts
// @MX:REASON: title, paragraphs, headings, cells and link descriptions all rely on it; weakening one rule re-opens Org object injection everywhere
func escapeText(s string) string {
	rs := []rune(norm.NFC.String(s))
	var b strings.Builder
	b.Grow(len(s) + len(s)/4)
	prev := rune(-1)
	for i, r := range rs {
		if isLineBreakOrControl(r) {
			r = ' '
		}
		switch r {
		case '_':
			if hasSuffixFold(b.String(), "src") || hasSuffixFold(b.String(), "call") {
				b.WriteString(zwsp)
			}
			b.WriteRune(r)
			b.WriteString(zwsp)
		case '$':
			b.WriteRune(r)
			b.WriteString(wordJoiner)
		case '[', '{', '<', '\\', '*', '/', '+', '=', '~':
			b.WriteRune(r)
			b.WriteString(zwsp)
		case '@':
			b.WriteRune(r)
			if i+1 < len(rs) && rs[i+1] == '@' {
				b.WriteString(zwsp)
			}
		case ':':
			if unicode.IsLetter(prev) || unicode.IsDigit(prev) {
				b.WriteString(zwsp)
			}
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
		prev = r
	}
	return b.String()
}

func isLineBreakOrControl(r rune) bool {
	return unicode.IsControl(r) || r == ' ' || r == ' ' || r == '\u0085'
}

func hasSuffixFold(s, suffix string) bool {
	return len(s) >= len(suffix) && strings.EqualFold(s[len(s)-len(suffix):], suffix)
}

// escapeLineStart trims indentation from a content line and prefixes ZWSP
// when the line would otherwise start an Org element: headline `*`, keyword /
// comment / block `#`, table `|`, drawer or fixed-width `:`, diary sexp `%`,
// plain-list bullets (`-`/`+` + space, `1.`, `1)`, `a.`) and the `-----` rule.
// Every Org element regexp anchors on `^[ \t]*`, and ZWSP is neither.
func escapeLineStart(line string) string {
	line = strings.TrimLeft(line, " \t")
	if needsLinePrefix(line) {
		return zwsp + line
	}
	return line
}

func needsLinePrefix(l string) bool {
	if l == "" {
		return false
	}
	switch l[0] {
	case '*', '#', '|', ':', '%':
		return true
	case '-', '+':
		if len(l) == 1 || l[1] == ' ' || l[1] == '\t' || strings.HasPrefix(l, "-----") {
			return true
		}
	}
	i := 0
	for i < len(l) && l[i] >= '0' && l[i] <= '9' {
		i++
	}
	if i == 0 && (l[0] >= 'a' && l[0] <= 'z' || l[0] >= 'A' && l[0] <= 'Z') {
		i = 1
	}
	return i > 0 && i < len(l) && (l[i] == '.' || l[i] == ')') &&
		(i+1 == len(l) || l[i+1] == ' ' || l[i+1] == '\t')
}

// safeHref returns the Org link target for an http(s) href, or false for any
// other scheme (shell:, elisp:, javascript:, file:, mailto:, relative...).
func safeHref(h string) (string, bool) {
	u, err := url.Parse(h)
	if err != nil || u.Opaque != "" || u.Host == "" {
		return "", false
	}
	if u.Scheme != "http" && u.Scheme != "https" { // url.Parse lowercases Scheme
		return "", false
	}
	return encodeHref(u.Scheme + h[len(u.Scheme):]), true
}

// encodeHref percent-encodes every byte that could end or escape an Org link
// target, split a line, or be mistaken for markup: whitespace, control and
// format characters, brackets, backslash, braces, angle brackets, quotes,
// caret, backtick and pipe (table cells).
func encodeHref(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 || r == 0x7f || unicode.IsSpace(r) || unicode.Is(unicode.Cf, r) ||
			isLineBreakOrControl(r) || strings.ContainsRune("[]\\{}<>\"^`|", r) {
			for _, c := range []byte(string(r)) {
				fmt.Fprintf(&b, "%%%02X", c)
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// fullwidthBrackets replaces ASCII brackets in a link description so it can
// never terminate the link early or open a nested one.
func fullwidthBrackets(s string) string {
	return strings.NewReplacer("[", "［", "]", "］").Replace(s)
}

var (
	localVarsRe = regexp.MustCompile(`(?i)local variables:`)
	endTokenRe  = regexp.MustCompile(`(?i)(^|[^\p{L}])(end):`)
)

// breakEmacsTokens runs over the whole rendered document, including verbatim
// example blocks, and breaks the strings Emacs itself scans for before Org is
// involved: `-*-` (mode line, first line), `Local Variables:` and `End:` (the
// file-local variables block, searched in the last 3000 characters).
//
// @MX:WARN: [AUTO] residual risk: these tokens are matched literally
// @MX:REASON: Emacs matches them with case-fold and exact spacing; a future Emacs that relaxed the match (e.g. arbitrary whitespace, or ignoring ZWSP) would re-open file-local eval. Users should keep enable-local-variables at a prompting value.
func breakEmacsTokens(s string) string {
	s = localVarsRe.ReplaceAllStringFunc(s, func(m string) string {
		return m[:len(m)-1] + zwsp + ":"
	})
	for strings.Contains(s, "-*-") {
		s = strings.ReplaceAll(s, "-*-", "-"+zwsp+"*-")
	}
	return endTokenRe.ReplaceAllString(s, "${1}${2}"+zwsp+":")
}
