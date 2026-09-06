package media_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/media"
)

// storedName re-derives design.md §9.3's naming function independently of
// the implementation, so the tests below assert against a value computed
// from the file's bytes rather than against whatever the code produced.
func storedName(t *testing.T, sanitizedBase, ext string, content []byte) string {
	t.Helper()
	sum := sha256.Sum256(content)
	return sanitizedBase + "-" + hex.EncodeToString(sum[:])[:12] + ext
}

// writeFile plants content at dir/name and returns the absolute path.
func writeFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// imgHTML is what go-org emits for a one-part [[file:X]] image link,
// measured in this repo against go-org v1.9.1 rather than assumed.
func imgHTML(src string) string {
	return `<p><img src="` + src + `" alt="` + src + `" title="` + src + `" /></p>`
}

// anchorHTML is what go-org emits for a two-part [[file:X][desc]] link.
func anchorHTML(href, desc string) string {
	return `<p><a href="` + href + `">` + desc + `</a></p>`
}

// --- AC-C-010a: resolution against the entry's OWN directory ---

func TestRewriteResolvesAgainstEntryOwnDirectory(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "deep", "notes")
	content := []byte("PNG-A")
	abs := writeFile(t, sub, "diagram.png", content)

	// A decoy of the same name at the sync root must NOT be the one chosen.
	writeFile(t, root, "diagram.png", []byte("PNG-DECOY"))

	fields, uploads, err := media.Rewrite(sub, root, map[string]string{
		"Back": imgHTML("diagram.png"),
	})
	if err != nil {
		t.Fatalf("Rewrite: unexpected error %v", err)
	}
	want := storedName(t, "diagram", ".png", content)
	if !strings.Contains(fields["Back"], `src="`+want+`"`) {
		t.Errorf("src not rewritten to the entry-directory file's stored name\n got: %s\nwant src=%q", fields["Back"], want)
	}
	if len(uploads) != 1 {
		t.Fatalf("uploads = %d, want 1", len(uploads))
	}
	// The reported path is canonical (absolute, symlinks evaluated), which
	// is the form AnkiConnect's storeMediaFile "path" parameter needs.
	wantPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		t.Fatal(err)
	}
	if uploads[0].Path != wantPath {
		t.Errorf("upload path = %q, want the entry-directory file %q", uploads[0].Path, wantPath)
	}
	if uploads[0].Filename != want {
		t.Errorf("upload filename = %q, want %q", uploads[0].Filename, want)
	}
}

// --- AC-C-010b: confinement rejects an escape above the sync root ---

func TestRewriteRejectsEscapeAboveSyncRoot(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "root")
	sub := filepath.Join(root, "notes")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	// A real, readable file that nonetheless lies OUTSIDE the sync root:
	// confinement, not existence, must be what rejects it.
	writeFile(t, outer, "secret.png", []byte("OUTSIDE"))

	_, uploads, err := media.Rewrite(sub, root, map[string]string{
		"Back": imgHTML("../../secret.png"),
	})
	var nf *media.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("err = %v, want *media.NotFoundError for a path outside the sync root", err)
	}
	if nf.Reference != "../../secret.png" {
		t.Errorf("NotFoundError.Reference = %q, want the offending reference", nf.Reference)
	}
	if len(uploads) != 0 {
		t.Errorf("uploads = %v, want none for a rejected reference", uploads)
	}
}

// --- AC-C-011a: content-hash suffix inserted BEFORE the extension ---

func TestRewriteStoredNameCarriesDigestBeforeExtension(t *testing.T) {
	root := t.TempDir()
	content := []byte("PNG-BYTES")
	writeFile(t, root, "diagram.png", content)

	fields, uploads, err := media.Rewrite(root, root, map[string]string{
		"Back": imgHTML("diagram.png"),
	})
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	shape := regexp.MustCompile(`^diagram-[0-9a-f]{12}\.png$`)
	if !shape.MatchString(uploads[0].Filename) {
		t.Errorf("filename %q does not match diagram-<12 hex>.png", uploads[0].Filename)
	}
	want := storedName(t, "diagram", ".png", content)
	if uploads[0].Filename != want {
		t.Errorf("filename = %q, want %q (SHA-256 of the file's bytes)", uploads[0].Filename, want)
	}
	if got := fields["Back"]; !strings.Contains(got, `src="`+want+`"`) {
		t.Errorf("rewritten src does not equal the stored filename: %s", got)
	}
	// The name is computed locally, so no upload response is consulted:
	// Rewrite issues no network call at all and still produced the name.
}

func TestRewriteLowercasesTheOriginalExtension(t *testing.T) {
	root := t.TempDir()
	content := []byte("UPPER-EXT")
	writeFile(t, root, "Shot.PNG", []byte("UPPER-EXT"))

	_, uploads, err := media.Rewrite(root, root, map[string]string{
		"Back": imgHTML("Shot.PNG"),
	})
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	want := storedName(t, "shot", ".png", content)
	if uploads[0].Filename != want {
		t.Errorf("filename = %q, want %q (basename lowercased, extension lowercased)", uploads[0].Filename, want)
	}
}

// --- AC-C-011b: two same-named images in different directories ---

func TestRewriteCrossDirectoryCollisionProducesDistinctNames(t *testing.T) {
	root := t.TempDir()
	dirA := filepath.Join(root, "a")
	dirB := filepath.Join(root, "b")
	writeFile(t, dirA, "diagram.png", []byte("CONTENT-A"))
	writeFile(t, dirB, "diagram.png", []byte("CONTENT-B"))

	_, upA, err := media.Rewrite(dirA, root, map[string]string{"Back": imgHTML("diagram.png")})
	if err != nil {
		t.Fatalf("Rewrite A: %v", err)
	}
	_, upB, err := media.Rewrite(dirB, root, map[string]string{"Back": imgHTML("diagram.png")})
	if err != nil {
		t.Fatalf("Rewrite B: %v", err)
	}
	if upA[0].Filename == upB[0].Filename {
		t.Errorf("two same-named images with different content share the stored name %q", upA[0].Filename)
	}
}

// --- AC-C-011c: identical content, identical and differing basenames ---

func TestRewriteIdenticalContentAndBasenameDedupesWithinEntry(t *testing.T) {
	root := t.TempDir()
	dirA := filepath.Join(root, "a")
	dirB := filepath.Join(root, "b")
	writeFile(t, dirA, "same.png", []byte("IDENTICAL"))
	writeFile(t, dirB, "same.png", []byte("IDENTICAL"))

	_, upA, _ := media.Rewrite(dirA, root, map[string]string{"Back": imgHTML("same.png")})
	_, upB, _ := media.Rewrite(dirB, root, map[string]string{"Back": imgHTML("same.png")})
	if upA[0].Filename != upB[0].Filename {
		t.Errorf("identical content + identical basename gave %q and %q, want one stored name",
			upA[0].Filename, upB[0].Filename)
	}

	// Two references to the same file inside ONE entry collapse to one upload.
	_, ups, err := media.Rewrite(dirA, root, map[string]string{
		"Front": imgHTML("same.png"),
		"Back":  imgHTML("same.png"),
	})
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	if len(ups) != 1 {
		t.Errorf("uploads = %d, want 1 after per-entry dedupe by stored name", len(ups))
	}
}

func TestRewriteIdenticalContentDifferentBasenamesDiffer(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "one.png", []byte("IDENTICAL"))
	writeFile(t, root, "two.png", []byte("IDENTICAL"))

	_, up1, _ := media.Rewrite(root, root, map[string]string{"Back": imgHTML("one.png")})
	_, up2, _ := media.Rewrite(root, root, map[string]string{"Back": imgHTML("two.png")})
	if up1[0].Filename == up2[0].Filename {
		t.Errorf("identical content with different basenames collapsed to %q; "+
			"the stored name is basename + digest, so content alone must not determine it", up1[0].Filename)
	}
}

// --- AC-C-012a/b: two-part links and the extension boundary ---

func TestRewriteTwoPartImageLinkBecomesImgCarryingTheDescription(t *testing.T) {
	root := t.TempDir()
	content := []byte("TWO-PART")
	writeFile(t, root, "diagram.png", content)

	fields, uploads, err := media.Rewrite(root, root, map[string]string{
		"Back": anchorHTML("diagram.png", "My diagram"),
	})
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	want := storedName(t, "diagram", ".png", content)
	got := fields["Back"]
	if !strings.Contains(got, `<img src="`+want+`" alt="My diagram"`) {
		t.Errorf("two-part link not converted to an img carrying the description\ngot: %s", got)
	}
	if strings.Contains(got, "<a ") || strings.Contains(got, "</a>") {
		t.Errorf("an anchor for the converted target survives: %s", got)
	}
	if len(uploads) != 1 || uploads[0].Filename != want {
		t.Errorf("uploads = %v, want one upload named %q", uploads, want)
	}
}

func TestRewriteTwoPartLinkOutsideGoOrgImageSetIsUntouched(t *testing.T) {
	root := t.TempDir()
	// The file exists and is readable: what keeps it an anchor is the
	// extension boundary, not a resolution failure.
	writeFile(t, root, "x.avif", []byte("AVIF"))
	in := anchorHTML("x.avif", "d")

	fields, uploads, err := media.Rewrite(root, root, map[string]string{"Back": in})
	if err != nil {
		t.Fatalf("Rewrite: unexpected error %v — .avif lies outside go-org's image set, "+
			"so it is not a media reference and cannot be a diagnostic", err)
	}
	if fields["Back"] != in {
		t.Errorf("anchor changed\n got: %s\nwant: %s", fields["Back"], in)
	}
	if len(uploads) != 0 {
		t.Errorf("uploads = %v, want none", uploads)
	}
}

// --- AC-C-012c: video and audio pass through exactly ---

func TestRewriteVideoAndAudioPassThroughUntouched(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "clip.webm", []byte("WEBM"))
	writeFile(t, root, "sound.mp3", []byte("MP3"))

	cases := map[string]string{
		// go-org renders a one-part .webm as a <video> element.
		"video":       `<p><video src="clip.webm" title="clip.webm">clip.webm</video></p>`,
		"videoAnchor": anchorHTML("clip.webm", "c"),
		"audio":       anchorHTML("sound.mp3", "s"),
		"audioOne":    `<p><a href="sound.mp3">sound.mp3</a></p>`,
	}
	fields, uploads, err := media.Rewrite(root, root, cases)
	if err != nil {
		t.Fatalf("Rewrite: unexpected error %v — video and audio are not media references", err)
	}
	for name, want := range cases {
		if fields[name] != want {
			t.Errorf("%s changed\n got: %s\nwant: %s", name, fields[name], want)
		}
	}
	if len(uploads) != 0 {
		t.Errorf("uploads = %v, want none for video or audio targets", uploads)
	}
}

// --- AC-C-013: remote references untouched ---

func TestRewriteLeavesRemoteReferencesByteUnchanged(t *testing.T) {
	root := t.TempDir()
	cases := map[string]string{
		"https":       imgHTML("https://example.com/img.png"),
		"http":        imgHTML("http://example.com/img.png"),
		"httpsAnchor": anchorHTML("https://example.com/img.png", "remote"),
	}
	fields, uploads, err := media.Rewrite(root, root, cases)
	if err != nil {
		t.Fatalf("Rewrite: unexpected error %v", err)
	}
	for name, want := range cases {
		if fields[name] != want {
			t.Errorf("%s changed\n got: %s\nwant: %s", name, fields[name], want)
		}
	}
	if len(uploads) != 0 {
		t.Errorf("uploads = %v, want none", uploads)
	}
}

// --- AC-C-015a: a missing file is a reported, contained failure ---

func TestRewriteMissingFileReportsNotFoundNamingTheReference(t *testing.T) {
	root := t.TempDir()
	_, uploads, err := media.Rewrite(root, root, map[string]string{
		"Back": imgHTML("gone.png"),
	})
	var nf *media.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("err = %v, want *media.NotFoundError", err)
	}
	if !strings.Contains(nf.Error(), "gone.png") {
		t.Errorf("error %q does not name the reference", nf.Error())
	}
	if len(uploads) != 0 {
		t.Errorf("uploads = %v, want none", uploads)
	}
}

func TestRewriteUnreadableFileReportsNotFound(t *testing.T) {
	root := t.TempDir()
	// A directory where a file is expected: it resolves and is confined,
	// but cannot be read as media content.
	if err := os.MkdirAll(filepath.Join(root, "dir.png"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, _, err := media.Rewrite(root, root, map[string]string{"Back": imgHTML("dir.png")})
	var nf *media.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("err = %v, want *media.NotFoundError for an unreadable target", err)
	}
}

// --- AC-C-015b: no stored name may begin with "_" ---

func TestStoredNamesNeverBeginWithUnderscore(t *testing.T) {
	root := t.TempDir()
	// Anki treats a leading "_" as "exempt from Check Media", which would
	// exempt imoogi's own uploads from the cleanup this SPEC relies on.
	for _, base := range []string{"_leading.png", "__double.png", "-dash.png", "%%%.png"} {
		writeFile(t, root, base, []byte("X"+base))
		_, uploads, err := media.Rewrite(root, root, map[string]string{"Back": imgHTML(base)})
		if err != nil {
			t.Fatalf("Rewrite %s: %v", base, err)
		}
		if strings.HasPrefix(uploads[0].Filename, "_") {
			t.Errorf("stored name for %s is %q, which begins with _", base, uploads[0].Filename)
		}
	}
}

// --- design.md §9.3: the sanitization rules, exercised end to end ---

func TestStoredNameSanitization(t *testing.T) {
	content := []byte("SANITIZE")
	cases := []struct {
		name     string
		source   string
		wantBase string
	}{
		{"lowercases", "Diagram.png", "diagram"},
		{"replaces disallowed characters", "my diagram!.png", "my-diagram"},
		{"collapses runs", "a---b.png", "a-b"},
		{"strips edges", "-edge-.png", "edge"},
		{"keeps underscores inside", "a_b.png", "a_b"},
		{"falls back when empty", "%%%.png", "img"},
		{"keeps digits", "fig2.png", "fig2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, tc.source, content)
			_, uploads, err := media.Rewrite(root, root, map[string]string{"Back": imgHTML(tc.source)})
			if err != nil {
				t.Fatalf("Rewrite: %v", err)
			}
			want := storedName(t, tc.wantBase, ".png", content)
			if uploads[0].Filename != want {
				t.Errorf("filename = %q, want %q", uploads[0].Filename, want)
			}
		})
	}
}

// --- resolution mechanics ---

func TestRewriteDecodesPercentEscapesInTheReference(t *testing.T) {
	root := t.TempDir()
	content := []byte("SPACED")
	writeFile(t, root, "a b.png", content)

	_, uploads, err := media.Rewrite(root, root, map[string]string{
		"Back": imgHTML("a%20b.png"),
	})
	if err != nil {
		t.Fatalf("Rewrite: %v — a percent-escaped reference must resolve to the real name", err)
	}
	if want := storedName(t, "a-b", ".png", content); uploads[0].Filename != want {
		t.Errorf("filename = %q, want %q", uploads[0].Filename, want)
	}
}

func TestRewriteResolvesASubdirectoryReference(t *testing.T) {
	root := t.TempDir()
	notes := filepath.Join(root, "notes")
	content := []byte("SUBDIR")
	writeFile(t, filepath.Join(notes, "img"), "d.png", content)

	fields, uploads, err := media.Rewrite(notes, root, map[string]string{
		"Back": imgHTML("img/d.png"),
	})
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	want := storedName(t, "d", ".png", content)
	if uploads[0].Filename != want {
		t.Errorf("filename = %q, want %q", uploads[0].Filename, want)
	}
	if !strings.Contains(fields["Back"], `src="`+want+`"`) {
		t.Errorf("src not rewritten: %s", fields["Back"])
	}
}

func TestRewriteUpstairsReferenceStillInsideRootResolves(t *testing.T) {
	root := t.TempDir()
	notes := filepath.Join(root, "notes")
	if err := os.MkdirAll(notes, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("SIBLING")
	writeFile(t, filepath.Join(root, "assets"), "s.png", content)

	_, uploads, err := media.Rewrite(notes, root, map[string]string{
		"Back": imgHTML("../assets/s.png"),
	})
	if err != nil {
		t.Fatalf("Rewrite: %v — the target is above the entry's directory but still inside the sync root", err)
	}
	if want := storedName(t, "s", ".png", content); uploads[0].Filename != want {
		t.Errorf("filename = %q, want %q", uploads[0].Filename, want)
	}
}

// --- output shape ---

func TestRewriteLeavesFieldsWithNoMediaByteIdentical(t *testing.T) {
	root := t.TempDir()
	in := map[string]string{
		"Front": "<p>Plain text</p>",
		"Back":  `<p>\(x^2\) and <code>src="a.png"</code> in prose</p>`,
	}
	fields, uploads, err := media.Rewrite(root, root, in)
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	for k, want := range in {
		if fields[k] != want {
			t.Errorf("%s changed\n got: %s\nwant: %s", k, fields[k], want)
		}
	}
	if uploads != nil {
		t.Errorf("uploads = %v, want nil", uploads)
	}
}

func TestRewriteDoesNotMutateTheCallersMap(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "d.png", []byte("M"))
	in := map[string]string{"Back": imgHTML("d.png")}
	original := in["Back"]

	if _, _, err := media.Rewrite(root, root, in); err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	if in["Back"] != original {
		t.Errorf("caller's map was mutated: %s", in["Back"])
	}
}

func TestRewriteUploadOrderIsDeterministic(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.png", []byte("A"))
	writeFile(t, root, "b.png", []byte("B"))
	in := map[string]string{
		"Front": imgHTML("b.png"),
		"Back":  imgHTML("a.png"),
	}
	var first []media.Upload
	for i := 0; i < 8; i++ {
		_, ups, err := media.Rewrite(root, root, in)
		if err != nil {
			t.Fatalf("Rewrite: %v", err)
		}
		if first == nil {
			first = ups
			continue
		}
		if len(ups) != len(first) {
			t.Fatalf("upload count varies across runs: %d vs %d", len(ups), len(first))
		}
		for j := range ups {
			if ups[j] != first[j] {
				t.Fatalf("upload order varies across runs at %d: %v vs %v", j, ups[j], first[j])
			}
		}
	}
	// Field names are visited in sorted order, so Back's image comes first.
	if len(first) != 2 || !strings.HasPrefix(first[0].Filename, "a-") {
		t.Errorf("uploads = %v, want the sorted-field-name order (Back before Front)", first)
	}
}

func TestRewriteEscapesTheDescriptionIntoTheAltAttribute(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "d.png", []byte("ALT"))
	// go-org emits inline markup inside the anchor and escapes quotes as
	// &#34;. Neither may reach the alt attribute as raw markup.
	fields, _, err := media.Rewrite(root, root, map[string]string{
		"Back": anchorHTML("d.png", `<strong>bold</strong> q&#34;r &amp; more`),
	})
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	got := fields["Back"]
	if strings.Contains(got, "<strong>") {
		t.Errorf("markup leaked into the alt attribute: %s", got)
	}
	if !strings.Contains(got, `alt="bold q&#34;r &amp; more"`) {
		t.Errorf("alt not escaped as expected\ngot: %s", got)
	}
}

func TestRewriteEmptySrcIsLeftAlone(t *testing.T) {
	root := t.TempDir()
	in := map[string]string{"Back": `<p><img src="" alt="" /></p>`}
	fields, _, err := media.Rewrite(root, root, in)
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	if fields["Back"] != in["Back"] {
		t.Errorf("empty src changed: %s", fields["Back"])
	}
}
