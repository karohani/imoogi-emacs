# Acceptance Criteria — SPEC-ANKICARD-004

Fifteen criteria, `AC-SW-001`..`AC-SW-015`, contiguous. Every requirement in
`spec.md` § 3 is covered by at least one, and every criterion names the
requirements it verifies and the command that decides it.

Each criterion is binary: it passes or it fails, and the deciding command's
output says which. A criterion that needs a human to judge "looks right" is not
a criterion, and none below is one — where real-Anki behavior is the only
authority, § Residual Risk records it as unverified rather than smuggling it in
as a check.

## Corpus

The composition corpus is one table exercised by the arrow-identification and
composition tests. Thirty-two rows; every one is reachable from a body a user
could write. Rows 18 through 24 were added at plan audit iteration 1 and rows 25
through 32 at iteration 2; each pins a measured defect or a rule that would
otherwise be untested. The project's notes are Korean, so the corpus carries multi-byte
rows rather than testing only the case this repository does not have.

| # | Body (remaining) | Arrow lines found | Note |
|---|---|---|---|
| 1 | `도쿄 :-> 일본` | 1 | the ordinary case, multi-byte |
| 2 | `도쿄 :-> 일본`⏎`서울 :-> 한국` | 2 | consecutive; one paragraph |
| 3 | `A :<- B` | 1 | left only |
| 4 | `A :<-> B` | 1 | both sides, two cards |
| 5 | `A :-> B :-> C` | 1 | first token only; right side is `B :-> C` |
| 6 | `A:->B` | 1 | no surrounding whitespace |
| 7 | `A :->` | 0 | right side absent |
| 8 | `:-> B` | 0 | left side absent |
| 9 | `A :->` + three spaces | 0 | right side whitespace-only after trimming |
| 10 | `- 도쿄 :-> 일본` | 0 | list item |
| 11 | `- Tokyo :: Japan :-> x` | 0 | description item; the `::` claims the line |
| 12 | `#+BEGIN_SRC go`⏎`x :-> y`⏎`#+END_SRC` | 0 | block interior |
| 13 | `#+BEGIN_QUOTE`⏎`x :-> y`⏎`#+END_QUOTE` | 0 | block interior; the stated cost of one rule |
| 14 | `문맥 한 줄`⏎`도쿄 :-> 일본` | 1 | prose above, same paragraph |
| 15 | `A :-> B`⏎⏎`text`⏎⏎`C :-> D` | 2 | two runs, blank-separated |
| 16 | `{{c1::A}} :-> B` | 1 | left side pre-marked |
| 17 | `도쿄 :-> 일본`⏎⏎`- 참고 사항` | 1 | an arrow-free list below; it is context, not answers |
| 18 | `{{c1::도쿄 :-> 일본}}` | 0 | the arrow sits inside a marker's span (D1) |
| 19 | `{{c1::도쿄 :-> 일본` | 0 | unclosed opening; treated as spanning to the body's end |
| 20 | `:-> B :-> C` | 0 | the first token is in the left side (D2) |
| 21 | `- 수도`⏎`  도쿄 :-> 일본` | 0 | list continuation; no bullet on the arrow line (D12) |
| 22 | `\| A :-> B \|` | 0 | table row; the right side would swallow the cell delimiter |
| 23 | `도쿄 :-> 일본`⏎`참고: 성립 조건` | 1 | prose immediately below an arrow line (D3) |
| 24 | `{{c1::메모`⏎⏎`도쿄 :-> 일본` | 0 | unclosed opening on an earlier line; the span reaches the arrow |
| 25 | `: A :-> B` | 0 | fixed-width line; renders inside `<pre>` (N1) |
| 26 | `[fn:1] A :-> B` | 0 | footnote definition; renders to nothing (N1) |
| 27 | `* A :-> B` | 0 | heading (N6) |
| 28 | `#+TITLE: A :-> B` | 0 | document keyword (N6) |
| 29 | `# A :-> B` | 0 | comment; renders to nothing (N6) |
| 30 | `\begin{align}`⏎`A :-> B`⏎`\end{align}` | 0 | LaTeX environment interior; raw passthrough, context-dependent |
| 31 | `~도쿄 :-> 일본~` | 1 | **not repaired** — inline code span; composes and breaks (N2) |
| 32 | `[fn::도쿄 :-> 일본]` | 1 | **not repaired** — inline footnote; Anki refuses the note (N2) |

## AC Matrix

### AC-SW-001 — Arrow lines are identified by the stated rule (REQ-SW-001)

**Given** each body in the corpus above,
**When** the arrow scanner runs over its remaining body,
**Then** it finds exactly the number of arrow lines the corpus records, and for
each one the left and right sides are the trimmed text on either flank of the
first arrow token.

Sub-criteria, each decided by its corpus rows:

- **a** (REQ-SW-001.1) Rows 1, 3, 4: the three tokens are each recognized, and
  row 4 is read as `:<->` rather than as `:<-` followed by `>`.
- **b** (REQ-SW-001.3) Row 5: one arrow line, right side `B :-> C`. Rendered,
  the second token appears as `:-&gt;` carrying no `<b>`. Row 20
  (`:-> B :-> C`): **zero** arrow lines — the expression selects the second
  token, so the first is in the left side and the line is rejected. Measured,
  the expression alone yields `L=":-> B"`, which is why this row exists.
- **c** (REQ-SW-001.4) Rows 7, 8, 9: zero arrow lines. Row 9 is the one the
  regex alone does not reject.
- **d** (REQ-SW-001.5, block interiors) Rows 12, 13: zero arrow lines, and the
  block text is rendered unchanged.
- **e** (REQ-SW-001.5, list content) Rows 10, 11, 21: zero arrow lines. Rows 10
  and 11 are bullet lines; row 21 is an indented **continuation** line carrying
  no bullet, which a line-local bullet test admits and the list-continuation
  rule rejects. In all three the list renders as one intact list — for row 21,
  one `<li>` holding both of its lines.
- **e2** (REQ-SW-001.5, context-free non-paragraph shapes) Rows 22 and 25
  through 29: zero arrow lines each, and each renders exactly as it does today.
  Row 22's table keeps both cell delimiters; row 25's fixed-width line stays
  inside `<pre class="example">`; rows 26 and 29 render to nothing at all,
  which is the pair that matters most — a marker composed there would be
  invisible and Anki would refuse the note with no deletion.
- **e3** (REQ-SW-001.5, the LaTeX environment) Row 30: zero arrow lines,
  **and** the composed body is byte-identical to the input — no snippet, no
  marker, no break. Both halves are required: a scanner that found zero arrow
  lines for the wrong reason would satisfy a count-only assertion, and the
  byte-identity half is what catches it. This row is context-dependent rather
  than context-free: measured, the interior line reads as an ordinary paragraph
  **on its own**, and only the enclosing environment makes it otherwise, so it
  is the row that fails if the environment's extent is not taken from the
  parser's whole-body classification.
- **f** (REQ-SW-001.7) Rows 14 and 17: the prose line renders unchanged, outside
  every generated marker, above the composed arrow line; and the arrow-free list
  below one renders as an ordinary list, carrying no generated marker and no
  `children-list` container — it is context, not answers, and swift composes no
  answer list at all.

- **g** (§ 5, inline constructs — **pinned, not repaired**) Rows 31 and 32: one
  arrow line each, because the line *is* ordinary paragraph text at block level
  and REQ-SW-001.5 admits it. The test asserts the measured broken output
  rather than a correct one: row 31's rendered field carries the literal text
  `@@html:` inside a `<code>` element, and row 32's renders its `{{c1::` and
  its `}}` in **different block-level elements**, so Anki pairs neither with
  the other and finds no deletion. The assertion is stated at that strength
  rather than as a byte-offset comparison: for row 32's exact body the closing
  `}}` does land at the lower offset, but a body with trailing prose after the
  footnote flips the order while staying just as broken, and a test author
  choosing their own body would then fail a correct implementation.
  Asserting the defect is what stops it changing silently — the precedent
  SPEC-ANKICARD-003 set for the dangling supplementary marker. A future card
  that repairs the class will change these two rows deliberately.

```bash
go test -run 'TestSwiftArrowLines' ./internal/anki/orgdoc -v
```

### AC-SW-002 — An entry with no arrow line is skipped with the new code (REQ-SW-002, REQ-SW-010)

**Given** a cloze-style entry carrying `ANKI_SWIFT: t` and a body drawn from any
corpus row whose arrow-line count is zero — rows 7 through 13, 18 through 22,
and 24 through 30,
**When** the entry is synchronized,
**Then** it is reported skipped with exactly one error carrying the code
`swift_arrow_missing`; the entry's existing note identifier is carried on the
result where it has one; no AnkiConnect write request is issued; and its
registry record is unchanged.

**And** given corpus row 12 specifically — a `#+BEGIN_SRC` block and nothing
else — under a **multiline** option instead,
**Then** it is skipped with `multiline_answer_missing` rather than this code, so
the two diagnostics are distinguishable on one body. Row 12 is named rather
than "the same bodies" because rows 10 and 11 are list items, and a list is
precisely what a multiline entry takes as its answers: those two bodies compose
successfully under a multiline option and are not a cross-check at all.

```bash
go test -run 'TestSwiftArrowMissing' ./internal/anki/planner -v
```

### AC-SW-003 — Numbering is positional and restarts per entry (REQ-SW-003)

**Given** an entry whose body is corpus row 2 (two arrow lines, both `:->`),
**When** it is composed,
**Then** line 1's right side carries `c1` and line 2's right side carries `c3`;
the numbers `c2` and `c4` are consumed by the unwrapped left sides and appear
nowhere.

**And** given a second entry with the same body in the same run,
**Then** its numbers also begin at `c1` — the count restarts per entry.

**And** given a body with an ordinary paragraph between two arrow lines,
**Then** the second arrow line is still line 2 and carries `c3`: the count runs
over arrow lines only, not over body lines.

```bash
go test -run 'TestSwiftNumbering' ./internal/anki/orgdoc -v
```

### AC-SW-004 — The token selects which sides are wrapped (REQ-SW-004)

**Given** three single-line bodies differing only in their token,
**When** each is composed,
**Then** `:->` wraps the right side and leaves the left bare; `:<-` wraps the
left and leaves the right bare; `:<->` wraps both.

**And** the `:<->` composition carries two distinct numbers, so the rendered
`Text` field yields two cards rather than one.

```bash
go test -run 'TestSwiftDirectionTable' ./internal/anki/orgdoc -v
```

### AC-SW-005 — Generated numbers do not collide with a hand-written marker (REQ-SW-005)

**Given** an entry whose body is corpus row 18 — `{{c1::도쿄 :-> 일본}}`, a
hand-written marker spanning the arrow,
**When** the arrow scanner runs,
**Then** the line is **not** an arrow line, so the entry composes no marker
around either side; and where it is the body's only candidate the entry is
skipped with `swift_arrow_missing`. The rendered field must contain no `{{c2::`
anywhere, which is the direct assertion that the corruption does not occur.

**And** given corpus row 19 — `{{c1::도쿄 :-> 일본`, an **unclosed** opening on
the arrow's own line,
**Then** the line is likewise not an arrow line.

**And** given corpus row 24 — an unclosed opening on an **earlier** line and a
clean arrow two lines below it,
**Then** that arrow line is **also** excluded, and the entry is skipped with
`swift_arrow_missing`. This row is what distinguishes the specified rule from a
line-local reading of it: line-locally the arrow line carries no marker at all
and would compose, and the generated `}}` would then close the author's
dangling opening around content they never marked. Row 19 alone does not
discriminate between the two readings, so it cannot stand in for this one.

**And** given an entry whose body is corpus row 16 — `{{c1::A}} :-> B`, where
the marker sits wholly on one side,
**When** it is composed,
**Then** the left side reaches the renderer byte-for-byte as
`{{c1::A}}`, carrying no second marker around it; and the right side is wrapped
at `c2`, not `c1`.

**And** given a body carrying `{{c5::…}}` on a non-arrow line above two `:->`
arrow lines,
**Then** the first arrow line's right side is `c6` and the second's is `c8` —
the offset applies to the whole formula, so the gap between consecutive lines is
unchanged at two.

**And** given a body where every arrow line's selected side already carries a
hand-written marker,
**Then** nothing is wrapped, the entry is **not** skipped, and the rendered
field carries the author's markers unchanged.

```bash
go test -run 'TestSwiftHandWrittenMarker' ./internal/anki/orgdoc -v
```

### AC-SW-006 — A generated marker cannot close early (REQ-SW-006)

**Given** an arrow line whose wrapped side is `\sqrt{a^{2}}`,
**When** it is composed,
**Then** the wrapped content carries the separated form `\sqrt{a^{2} }` and one
space before the closing `}}`, so the marker's first `}}` is its own.

**And** the Go composition and the editor command produce the **same** string
for the shared fixture the two implementations are compared against, so the two
copies of the rule cannot drift.

```bash
go test -run 'TestSwiftBraceSafety' ./internal/anki/orgdoc -v
make test-elisp 2>&1 | grep -i 'cloze-safe'
```

### AC-SW-007 — The arrow is emphasized via an export snippet (REQ-SW-007)

**Given** an arrow line carrying each of the three tokens,
**When** the entry is rendered,
**Then** the rendered `Text` field carries `<b>:-&gt;</b>`, `<b>:&lt;-</b>`, and
`<b>:&lt;-&gt;</b>` respectively — a real `<b>` element, not the escaped
characters `&lt;b&gt;`.

**And** REQ-SW-007.3's bound is decided by a **test**, not by reading a grep:
a test asserts that the snippet-producing table is the only producer of an
`@@html:` string in the package, that its entries are exactly the three arrow
constants plus the line break, and that composing every corpus row emits no
`@@html:` substring outside those four values. The grep below is a convenience
for a human reader, not the deciding step — the earlier wording claimed the
check was mechanical while leaving variable-versus-constant to a reader's
judgment, and that claim is withdrawn:

```bash
go test -run 'TestSwiftArrowEmphasis|TestSwiftSnippetBound' ./internal/anki/orgdoc -v
grep -n '@@html:' internal/anki/orgdoc/*.go | grep -v '_test.go'
```

### AC-SW-008 — The pipeline order is split, compose, gate, render (REQ-SW-008)

**Given** a swift entry whose only arrow line sits inside a `#+BEGIN_EXTRA`
block,
**When** it is rendered,
**Then** it is skipped with `swift_arrow_missing` — the split ran first, so
supplementary content never became an arrow line — and the `Back Extra` field
would have carried that content had the entry rendered.

**And** given a swift entry with an arrow line and **no** hand-written marker
anywhere,
**Then** it renders rather than being skipped with `cloze_marker_missing`: the
generated marker satisfies the gate.

**And** given a swift entry whose body carries a blank-line run,
**Then** the run reaches the renderer uncollapsed — asserted by rendering
corpus row 15 and observing three paragraphs, which is the tree's behavior
today and the evidence that DD-7's decision was implemented.

```bash
go test -run 'TestSwiftPipelineOrder' ./internal/anki/orgdoc -v
```

### AC-SW-009 — The dispatch is extended, not duplicated, and is total (REQ-SW-009)

**Given** the renderer package,
**When** its exported surface is enumerated,
**Then** it carries exactly two entry points — the unwidened one and the
option-taking one — and the unwidened one's signature assertion still compiles
unchanged.

**And** given an entry with no option at all,
**Then** the option-taking entry point's output is byte-identical to the
unwidened one's for the same inputs, over the full golden corpus.

**And** given an entry whose three options all carry their falsy resolution —
the swift option named explicitly beside the two the earlier SPEC named —
**Then** the rendered fields are byte-identical to the no-option rendering and
carry neither the `children-list` container nor any `swift-arrow` emphasis or
inline break.

**And** given the two unreachable branches — swift on beside a multiline option,
and a swift option on a non-cloze note type — called **directly** on the
renderer, bypassing the gate,
**Then** the first composes as swift and the second delegates to the unwidened
entry point. Both are asserted at the renderer's own boundary, because neither
is reachable through the planner.

```bash
go test -run 'TestRenderSignatureIsUnchanged|TestSwiftDispatch' ./internal/anki/orgdoc -v
```

### AC-SW-010 — The new code never competes for the one diagnostic slot (REQ-SW-010)

**Given** an entry offending against a validation-gate rule **and** carrying no
arrow line,
**When** it is synchronized,
**Then** exactly one error is reported and its code is the gate's, not
`swift_arrow_missing` — evidence the gate ran before the render.

**And** given an entry carrying swift beside a multiline option,
**Then** the reported code is `card_option_conflict`; neither
`swift_arrow_missing` nor `multiline_answer_missing` can be reported for it, so
the two renderer diagnostics can never co-occur.

```bash
go test -run 'TestGatePrecedesRendering|TestSwiftWithMultilineOptionIsRejected' ./internal/anki/planner -v
```

### AC-SW-011 — Consecutive arrow lines render apart, and the arrow is styleable (REQ-SW-011)

**Given** an entry whose body is corpus row 2 — two consecutive arrow lines,
**When** it is rendered,
**Then** the `Text` field carries a `<br>` between them, so the two render on
separate lines rather than collapsing to one.

**And** given corpus row 23 — an arrow line with **prose** immediately below it,
**Then** no break is emitted, and the prose renders exactly as it does today:
in the same paragraph, unclassed, with no `<br>` before it. This is the
symmetric case to row 14 and the one that keeps REQ-SW-001.7 true without
qualification.

**And** given corpus row 21 — an arrow inside a list item's continuation,
**Then** the list renders as one intact `<ul><li>` holding both lines, with no
paragraph split and no attribute line anywhere in the composed source. A grep
over the composition source finds no `#+ATTR_HTML` literal at all, which is the
mechanical form of REQ-SW-011.3.

**And** the arrow's emphasis element carries the class, and the stylesheet
carries a rule for it that names no network-hosted resource and derives nothing
from a deck name:

```bash
go test -run 'TestSwiftLineBreak|TestSwiftArrowClass' ./internal/anki/orgdoc -v
grep -n '#+ATTR_HTML' internal/anki/orgdoc/swift.go
grep -n 'swift-arrow' internal/anki/model/assets/base.css
go test ./internal/anki/model/... -count=1
```

The `#+ATTR_HTML` grep must return nothing.

### AC-SW-012 — A non-swift entry renders byte-identically (REQ-SW-012.1, .2, .3)

**Given** the byte-identity corpus SPEC-ANKICARD-002 recorded and the multiline
corpus SPEC-ANKICARD-003 recorded,
**When** both are re-run after this SPEC's changes,
**Then** every recorded value matches byte-for-byte, and neither recorded file
was modified:

```bash
go test -run 'TestRenderGoldenCorpus|TestMultilineGoldenCorpus' ./internal/anki/orgdoc -v
git diff --stat -- internal/anki/orgdoc/testdata/
```

The `git diff --stat` must print nothing for the two pre-existing golden files.
A newly added swift golden is the only permitted change under that path.

**And** the hash call site still takes exactly four arguments, asserted at
compile time; and the option-free request log golden is unchanged.

### AC-SW-013 — The hash probe is reshaped and states the whole contract (REQ-SW-012.4, .5)

**Given** two entries identical but for their card-option fields — one with all
three fields absent, one with all three present carrying the falsy spelling,
**When** each is synchronized,
**Then** both reach the add branch and the registry records the **same** content
hash for each. The option field is present on the wire in the second and absent
in the first, so an implementation that fed an option field to the hash would
fail here.

**And** given two entries identical but for the swift option — one absent, one
on — over a body carrying an arrow line,
**Then** both reach the add branch and the recorded hashes **differ**. This is
the correct and intended outcome: the option reaches the hash through the
rendered field value, which is one of the four hash inputs and always was.

**And** the **third leg** holds: the compile-time assertion that the hash call
site takes exactly its four existing arguments still compiles unmodified. The
first two halves alone admit an implementation that hashes the *resolved
boolean* while ignoring the wire field — it passes both — so the contract is
this trio, and the criterion fails if any one of the three is absent.

**And** the helper the first half runs through carries, in its own docstring,
the statement of what the probe no longer covers, why, and all **three** legs —
so the next reader of the probe finds the narrowing and the full contract where
the probe is, not only in `plan.md`.

**And** the shared option fixture the validation gate's tables run against
carries an arrow line, so those tables still test the gate:

```bash
go test -run 'TestHashIsUnchangedByCardOptions|TestHashChangesThroughRenderedValue|TestHashCallSiteTakesFourArguments' ./internal/anki/planner -v
go test -run 'TestCardOptionValueRecognition|TestSwiftWithMultilineOptionIsRejected' ./internal/anki/planner -v
```

### AC-SW-014 — The editor command writes the property correctly (REQ-SW-013)

**Given** a heading with no `ANKI_SWIFT`,
**When** the toggle runs,
**Then** the drawer carries `ANKI_SWIFT: t`, and `ANKI_NOTE_TYPE` is
`imoogi-Cloze` where it was absent and unchanged where it was already
cloze-style.

**And** given a heading that resolves `ANKI_SWIFT` to `t` through inheritance,
**When** the toggle runs,
**Then** the heading's **own** drawer carries the falsy spelling — the property
is not deleted. Verified through `imoogi-props-resolve-swift` or
`org-entry-properties`, **never** `org-entry-get`, which SPEC-ANKICARD-003
REQ-ML-013.2 measured returns Lisp `nil` for a drawer plainly carrying that
spelling.

**And** — the criterion that pins the helper choice — the inheritance case
above is tested **without stubbing the confirmation prompt**: turning swift off
must fire no swift-conflict prompt at all. A test that stubs the prompt to
always-yes passes while the defect is present, so this clause requires the
unstubbed form.

**And** given a heading with **both** `ANKI_DIRECTION: ->` and
`ANKI_INCREMENTAL: t` on,
**When** the toggle runs and the user declines the conflict prompt,
**Then** nothing is written — not `ANKI_SWIFT`, and not `ANKI_NOTE_TYPE`; and
when the user confirms, **both** multiline properties are cleared to the falsy
spelling in the heading's own drawer and `ANKI_SWIFT` is written. Clearing only
one would leave the heading in the state the conflict rule rejects.

**And** the note-type failure message names the card option rather than the
multiline pair, since it now reports for a swift heading too.

**And** given a heading whose `ANKI_NOTE_TYPE` is `imoogi-Basic`,
**When** the toggle runs,
**Then** it reports the type and changes nothing.

**And** `ANKI_SWIFT` is a member of the property-name completion candidates,
where it previously was deliberately absent:

```bash
make test-elisp 2>&1 | tail -5
```

### AC-SW-015 — The new code is paired with a message, both ways (REQ-SW-014)

**Given** the front end's code-to-message table and the hard-coded back-end code
list the pairing test cross-checks against,
**When** the pairing tests run,
**Then** `swift_arrow_missing` is present in both, its message names the problem
and a corrective action including what does not count as an arrow line, and it
contains no stack trace, raw transport error, backtrace, or bare exit code.

**And** the hard-coded list still matches the codes the Go source actually
emits, so the cross-check is not satisfied by editing only the list:

```bash
make test-elisp 2>&1 | grep -i 'imoogi-error-test'
grep -n 'swift_arrow_missing' internal/anki/protocol/protocol.go modules/org/anki/imoogi-error.el tests/anki-error-test.el
```

## Edge Cases

| Case | Expected | Where pinned |
|---|---|---|
| Arrow token with no space on either side (`A:->B`) | An arrow line; sides are `A` and `B` | Corpus row 6, AC-SW-001 |
| Right side is whitespace only | Not an arrow line | Corpus row 9, AC-SW-001c |
| Two tokens on one line | One arrow line; the second token is right-side text, unemphasized | Corpus row 5, AC-SW-001b |
| Arrow in a `#+BEGIN_QUOTE` | Not an arrow line — the stated cost of reusing one block rule | Corpus row 13, AC-SW-001d |
| Arrow in list content, alone in the body | Not an arrow line; entry skipped with `swift_arrow_missing` | Corpus rows 10-11, 21, AC-SW-002 |
| Arrow in list content **beside** a plain arrow line | Only the plain line becomes a card; the list arrow renders on the card as the list item it is, uncarded, with no diagnostic | Pinned here as the specified behavior; `spec.md` § 5 names it and refuses the narrower alternative |
| Hand-written marker spanning the arrow | Not an arrow line; no generated marker anywhere on it | Corpus row 18, AC-SW-005 |
| Unclosed `{{cN::` above an arrow line | The opening spans to the body's end, so nothing below it is an arrow line | Corpus rows 19 and 24, AC-SW-005 |
| Arrow line whose first token is at column zero | Not an arrow line — the left side holds the first token | Corpus row 20, AC-SW-001b |
| Arrow in a table row, a heading, a keyword line, or a comment line | Not an arrow line; each renders unchanged | Corpus row 22, AC-SW-001e2 |
| Prose immediately below an arrow line | No break emitted; the prose renders exactly as today | Corpus row 23, AC-SW-011 |
| Empty remaining body | No arrow line; skipped with `swift_arrow_missing` | AC-SW-002 |
| Body is one `#+BEGIN_EXTRA` block carrying an arrow | Skipped; the arrow never leaves the extra field | AC-SW-008 |
| Dangling `#+END_EXTRA` between two arrow lines | Both lines are still arrow lines; the marker renders as visible literal text on its own line | Pinned here; the shape belongs to backlog card t16 |
| A hand-written marker on **both** sides of every arrow line | Nothing wrapped; entry renders rather than being skipped | AC-SW-005 |
| Left side ends in a bare URL before the token | go-org linkifies part of it; pre-existing renderer behavior, unconstrained | `plan.md` DD-1 |
| Multi-byte sides | Wrapped correctly; the scan never cuts inside a character because it splits on the token, not on a byte count | Corpus rows 1-2, AC-SW-001 |
| Right side is a single non-breaking space | Not an arrow line — the trim is Unicode-aware. An ASCII-only trim would admit it and wrap a blank | AC-SW-001c |
| Fixed-width line (`: A :-> B`) | Not an arrow line; renders inside `<pre class="example">` as today | Corpus row 25, AC-SW-001e2 |
| Footnote definition (`[fn:1] …`) | Not an arrow line; renders to nothing, as today | Corpus row 26, AC-SW-001e2 |
| Arrow inside a `\begin{…}` environment | Not an arrow line; the pairing rule covers both delimiter forms | Corpus row 30, AC-SW-001e3 |
| One-line `\begin{align} A :-> B \end{align}` | **Is** an arrow line — measured, the one-line form is an ordinary paragraph | AC-SW-001e3's converse |
| Arrow inside an inline construct (link, code, verbatim, emphasis, author's snippet, inline footnote) | Composes, and the output is broken in the way § 5 measures per construct. Pinned, not repaired | Corpus rows 31-32, AC-SW-001g |

## Quality Gate Criteria

| Gate | Threshold | Command |
|---|---|---|
| Go tests | exit 0 | `go test ./... -count=1` |
| `orgdoc` coverage | ≥ 100.0% (the `08f7504` baseline) | `go test -cover ./internal/anki/orgdoc` |
| `planner` coverage | ≥ 92.8% (the `08f7504` baseline) | `go test -cover ./internal/anki/planner` |
| Lint | 0 NEW issues over baseline | `make lint` |
| Format | exit 0 | `make fmt-check` |
| Cross-platform build | exit 0 both | `go build ./...` and `GOOS=windows GOARCH=amd64 go build ./...` |
| Elisp tests | 0 unexpected; count ≥ 435 | `make test-elisp` |
| Full local CI | exit 0 | `make ci-local` |

## Definition of Done

- [ ] AC-SW-001 through AC-SW-015 all PASS, each with its deciding command's
      verbatim output cited.
- [ ] Every quality gate above meets its threshold, measured against the
      `08f7504` baseline in `plan.md` § A.2.
- [ ] The two pre-existing golden corpora are unmodified; `git diff --stat` over
      `internal/anki/orgdoc/testdata/` shows only the new swift corpus.
- [ ] The three amendable test sites of `plan.md` § A.4 are green, and none of
      them was made green by deleting an assertion.
- [ ] No file on `plan.md` § A.3's PRESERVE list is modified, including the
      other-actor files.
- [ ] No git command that writes ran against the shared checkout.
- [ ] `README.md`'s Anki section documents the swift card.
- [ ] Every corpus row 18 through 30 is exercised by a test that fails if its
      defect returns — for the zero-arrow-line rows, asserting **both** the zero
      count and byte-identical composition, since a count alone passes on a
      scanner that found zero for the wrong reason. Rows 31 and 32 are
      exercised by a test asserting the measured broken output § 5 discloses.
- [ ] `progress.md` §E.2 and §E.3 carry the run-phase evidence and the
      audit-ready signal.

## Residual Risk

Recorded rather than checked, because the authority for each is a running Anki
collection and AnkiConnect is not connected in this environment. Each is
inherited from the same disposition the three earlier cards took.

1. **Number gaps, including a body with no `c1` at all.** A body of `:->` lines
   yields only odd numbers (`c1`, `c3`, `c5`); a body whose only line is `:<-`
   yields `c2` and **no `c1`**. Anki is understood to generate one card per
   distinct number present, so both should be harmless, but neither has been
   observed against a real collection. The second is the sharper case and is
   named because the positional rule makes it reachable from a one-line body.
2. **The `swift-arrow` stylesheet rule inside Anki's webview.** Verified as
   valid CSS, not as rendered output on a card. This risk **shrank** at plan
   audit iteration 1: the line-separation behaviour no longer depends on CSS at
   all — REQ-SW-011.1's inline break carries it — so a stylesheet that failed to
   load would cost only the arrow's styling, not the card's correctness.
3. **`<b>` inside a cloze deletion.** The emphasis sits between two markers
   rather than inside one, which is the safer arrangement, but multi-element
   HTML on an Anki cloze card is the risk the design record § 5 already flags as
   needing real-Anki verification.
4. **Existing `ANKI_SWIFT` headings change behavior on the next sync.** The
   property has been writable and gate-passing since `6f3ae6c`, and until now it
   selected nothing: a cloze-style heading carrying `ANKI_SWIFT: t` has been
   synchronizing as an ordinary cloze note all along. After this SPEC each such
   heading takes one of two new paths. One whose body carries an arrow line has
   its `Text` field rewritten with markers, emphasis, and an inline break, so its
   hash changes and it is reported **updated** once — correctly, since its card
   genuinely changed. One whose body carries no arrow line is **skipped** with
   `swift_arrow_missing`, so its existing note stays in the collection but stops
   receiving updates until the heading is corrected.

   A **third** path was identified by the plan audit and is closed rather than
   accepted: a heading whose arrow sits inside a hand-written marker would have
   had its card corrupted, not merely rewritten. REQ-SW-005.1 now makes such a
   line not an arrow line, so the heading either takes one of the two paths
   above on its remaining lines or is skipped. The earlier wording here — that
   an arrow-bearing swift heading "is reported updated once — correctly, since
   its card genuinely changed" — was false for that shape and is withdrawn.

   How many headings this reaches is a property of the user's own files and
   cannot be measured here, which is why it sits in this section rather than in
   a criterion. The direction is safe — no note is deleted and no review history
   is lost — but the first sync after this SPEC is not a no-op for anyone who
   adopted the property early.
