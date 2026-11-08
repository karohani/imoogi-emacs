package orgrender

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/karohani/obsidian-clipper/native-host/internal/protocol"
)

var update = flag.Bool("update", false, "rewrite golden files")

const hostileCmd = "rm -rf ~"
const hostileLisp = "(x)"

func TestCorpusIsSafe(t *testing.T) {
	for _, tc := range maliciousCorpus(hostileCmd, hostileLisp, "/etc/passwd") {
		t.Run(tc.name, func(t *testing.T) {
			out := Render(tc.doc)
			problems, heads, tables := auditOrg(out)
			for _, p := range problems {
				t.Error(p)
			}
			if heads != tc.headlines {
				t.Errorf("headlines = %d, want %d", heads, tc.headlines)
			}
			if tables != tc.tableLines {
				t.Errorf("table lines = %d, want %d", tables, tc.tableLines)
			}
			if !utf8.ValidString(out) {
				t.Error("output is not valid UTF-8")
			}
			if t.Failed() {
				t.Logf("output:\n%s", out)
			}
		})
	}
}

func TestCorpusGolden(t *testing.T) {
	for _, tc := range maliciousCorpus(hostileCmd, hostileLisp, "/etc/passwd") {
		t.Run(tc.name, func(t *testing.T) {
			out := Render(tc.doc)
			path := filepath.Join("testdata", "golden", tc.name+".org")
			if *update {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("missing golden (run with -update): %v", err)
			}
			if string(want) != out {
				t.Errorf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", tc.name, out, want)
			}
		})
	}
}

func TestCheckerCatchesUnescapedOutput(t *testing.T) {
	// The checker must reject what a naive renderer would produce; otherwise
	// TestCorpusIsSafe proves nothing.
	naive := "#+TITLE: x\n#+DATE: [2026-11-08 Sun 10:00]\n#+SOURCE: https://e.com/\n\n" +
		"* TODO x\n#+begin_src sh\n:PROPERTIES:\n%%(x)\n  #+TBLFM: x\nSCHEDULED: <2026-01-01>\n" +
		"{{{m}}} src_sh{x} call_x() <<t>> @@h:x@@ \\(x\\) $$y$$ [fn:1] [[elisp:(x)]] [[file:/etc/passwd]] shell:x\n" +
		"# Local Variables:\n# End:\n-*- x -*-\n"
	problems, heads, _ := auditOrg(naive)
	if heads != 1 {
		t.Fatalf("heads %d", heads)
	}
	for _, want := range []string{"content keyword", "drawer", "diary", "planning", `\{\{\{`, "src_", "call_", "<<", "@@", `\\[(\[]`, `\$\$`, `\[fn:`, `\[\[`, "elisp", "local variables", "end:", `-\*-`} {
		found := false
		for _, p := range problems {
			if strings.Contains(p, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("checker missed %q; problems: %v", want, problems)
		}
	}
	if p, _, _ := auditOrg("#+TITLE: x\n#+DATE: [2026-11-08 Sun 10:00]\n#+SOURCE: https://e.com/\n\n#+begin_example\n* x\n#+end_example\n"); len(p) == 0 {
		t.Error("checker accepted an unescaped star inside an example block")
	}
}

func render1(blocks ...protocol.Block) string {
	out := Render(Doc{Title: "t", SourceURL: "https://example.com/", ClippedAt: clipTime, Blocks: blocks,
		Assets: map[string]string{"a1": "0123456789abcdef0123456789abcdef.png"}})
	// Drop the 3 header lines, the blank line and the Source line.
	parts := strings.SplitN(out, "\n", 7)
	return parts[len(parts)-1]
}

func TestHeaderShape(t *testing.T) {
	out := Render(Doc{Title: "Hello [x]", SourceURL: "https://example.com/a b?q=[1]", ClippedAt: clipTime})
	want := "#+TITLE: Hello [\u200bx]\n" +
		"#+DATE: [2026-11-08 Sun 10:00]\n" +
		"#+SOURCE: https://example.com/a%20b?q=%5B1%5D\n" +
		"\n" +
		"Source: [[https://example.com/a%20b?q=%5B1%5D][https\u200b:/\u200b/\u200bexample.com/\u200ba b?q=\u200b［\u200b1］]]\n"
	if out != want {
		t.Fatalf("got\n%q\nwant\n%q", out, want)
	}
}

func TestSourceLineOnlyForHTTP(t *testing.T) {
	out := Render(Doc{Title: "t", SourceURL: "ftp://example.com/", ClippedAt: clipTime})
	if strings.Contains(out, "Source:") || strings.Contains(out, "[[") {
		t.Fatalf("non-http source rendered as link:\n%s", out)
	}
}

func TestInlineMarkup(t *testing.T) {
	got := render1(para(tx("a "), bold(tx("b")), tx(" "), italic(tx("i")), tx(" "), strike(tx("s")), tx(" "), icode("c d")))
	want := "a \u200b*b*\u200b \u200b/i/\u200b \u200b+s+\u200b \u200b~c d~\u200b\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestEmphasisFallbacks(t *testing.T) {
	for name, in := range map[string]protocol.Inline{
		"empty":         bold(),
		"edge space":    bold(tx(" x")),
		"marker inside": bold(tx("a*b")),
		"code tilde":    icode("a~b"),
		"code edge":     icode("x "),
		"code empty":    icode(""),
	} {
		got := render1(para(in))
		if strings.Contains(got, "\u200b*") && name != "marker inside" || strings.Contains(got, "\u200b~") {
			t.Errorf("%s: emphasis emitted for unsafe content: %q", name, got)
		}
	}
	if got := render1(para(icode("a~b"))); got != "a~\u200bb\n" {
		t.Errorf("code fallback %q", got)
	}
}

func TestLinks(t *testing.T) {
	cases := map[string]struct {
		in   protocol.Inline
		want string
	}{
		"http":        {link("http://e.com/x", tx("d")), "[[http://e.com/x][d]]\n"},
		"scheme case": {link("HTTPS://e.com/", tx("d")), "[[https://e.com/][d]]\n"},
		"brackets":    {link("https://e.com/a]]b", tx("x ] y")), "[[https://e.com/a%5D%5Db][x ］ y]]\n"},
		"no desc":     {link("https://e.com/"), "[[https://e.com/]]\n"},
		"js":          {link("javascript:alert(1)", tx("js")), "js\n"},
		"js no desc":  {link("javascript:alert(1)"), ""},
		"nested":      {link("https://a.com/", link("https://b.com/", tx("in"))), "[[https://a.com/][in]]\n"},
	}
	for name, tc := range cases {
		if got := render1(para(tc.in)); got != tc.want {
			t.Errorf("%s: got %q want %q", name, got, tc.want)
		}
	}
}

func TestImages(t *testing.T) {
	if got := render1(bimg("a1")); got != "[[file:assets/0123456789abcdef0123456789abcdef.png]]\n" {
		t.Errorf("saved image %q", got)
	}
	if got := render1(bimg("a2")); got != "(image not saved)\n" {
		t.Errorf("missing image %q", got)
	}
	if got := render1(para(link("https://e.com/", tx("d"), iimg("a1")))); got != "[[https://e.com/][d(image)]] [[file:assets/0123456789abcdef0123456789abcdef.png]]\n" {
		t.Errorf("image in link %q", got)
	}
}

func TestCodeBlockCommaEscape(t *testing.T) {
	got := render1(code("ok\n* star\n,* comma star\n  #+end_example\n,#+x\n#not\r\nlast\x00"))
	want := "#+begin_example\nok\n,* star\n,,* comma star\n  ,#+end_example\n,,#+x\n#not\nlast\n#+end_example\n"
	if got != want {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
}

func TestTable(t *testing.T) {
	got := render1(table(true, row("a", "b|c"), row("d")))
	want := "| a | b\\vert{}c |\n|---+---|\n| d |  |\n"
	if got != want {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
	if got := render1(table(true)); got != "" {
		t.Fatalf("empty table rendered %q", got)
	}
}

func TestListsAndQuote(t *testing.T) {
	got := render1(list(true, item(para(tx("one")), para(tx("more"))), item(list(false, item(para(tx("deep")))))))
	want := "1. one\n\n   more\n2.\n   - deep\n"
	if got != want {
		t.Fatalf("list got\n%q\nwant\n%q", got, want)
	}
	got = render1(quote(para(tx("q")), quote(para(tx("inner")))))
	want = "#+begin_quote\nq\n\ninner\n#+end_quote\n"
	if got != want {
		t.Fatalf("quote got\n%q\nwant\n%q", got, want)
	}
}

func TestHeadingLevels(t *testing.T) {
	got := render1(heading(3, tx("Title")), heading(1, tx("ends with:")))
	want := "*** \u200bTitle\n\n* \u200bends with\u200b:\u200b\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestKoreanSurvives(t *testing.T) {
	got := render1(para(tx("한글 본문입니다. 좋아요!")))
	if got != "한글 본문입니다. 좋아요!\n" {
		t.Fatalf("got %q", got)
	}
}

// TestEmacsAudit loads every corpus document into a real Emacs/Org, parses it,
// opens every link-looking position, runs org-babel-execute-buffer, exports
// it (which expands macros, INCLUDE and babel), and applies file-local
// variables. The hostile payloads are replaced by sentinels so a failure is
// observable and harmless. Skipped when no Emacs binary is found.
func TestEmacsAudit(t *testing.T) {
	emacs := os.Getenv("ORG_CLIPPER_EMACS")
	if emacs == "" {
		if p, err := exec.LookPath("emacs"); err == nil {
			emacs = p
		} else if _, err := os.Stat("/Applications/Emacs.app/Contents/MacOS/Emacs"); err == nil {
			emacs = "/Applications/Emacs.app/Contents/MacOS/Emacs"
		}
	}
	if emacs == "" {
		t.Skip("no emacs binary; set ORG_CLIPPER_EMACS to run the Org-level audit")
	}
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "pwned")
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte("INCLUDED-SECRET-7f3a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	audit, err := filepath.Abs(filepath.Join("testdata", "audit.el"))
	if err != nil {
		t.Fatal(err)
	}
	lisp := "(setq org-clipper-pwned (quote lisp-ran))"
	for _, tc := range maliciousCorpus("touch "+sentinel, lisp, secret) {
		t.Run(tc.name, func(t *testing.T) {
			f := filepath.Join(dir, tc.name+".org")
			if err := os.WriteFile(f, []byte(Render(tc.doc)), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(emacs, "--batch", "-Q", "-l", audit, f, strconv.Itoa(tc.headlines))
			cmd.Env = append(os.Environ(), "HOME="+dir)
			out, err := cmd.CombinedOutput()
			if err != nil || !strings.Contains(string(out), "AUDIT-OK") {
				t.Errorf("emacs audit failed (%v):\n%s", err, out)
			}
			if _, err := os.Stat(sentinel); err == nil {
				t.Fatal("shell payload executed")
			}
		})
	}
}
