# Implementation Plan — SPEC-ANKICARD-004

Ordered by decision-reversibility. § B carries the design decisions, hardest to
reverse first; § F carries the milestones, mechanical work last. A reviewer with
limited attention should spend it on § B, and within § B on DD-1 through DD-4
and DD-9.

DD-9 was added at plan audit iteration 1 and is placed next to the marker rules
it belongs with rather than at the end, so § B's order is by topic and
reversibility, not by number. DD-3 and DD-4 were rewritten in the same
iteration; both now record the decision they replace.

## § A Context

### A.1 What already exists

Commit `08f7504` (SPEC-ANKICARD-003) landed the multiline renderer and, with it,
every mechanism this SPEC needs except the arrow rule itself. This SPEC consumes
it and adds no second copy of any of it:

| Surface | Where | This SPEC's use |
|---|---|---|
| `RenderWithOptions`, dispatching on card kind and delegating to `Render` | `internal/anki/orgdoc/multiline.go` | Extended with one branch. No third entry point. |
| `CardOptions` | same | Gains the swift option. |
| `splitLines`, `blockRegions`, `readBullet`, and `scanAnswerList`'s list-continuation rule | same | Reused for the line scan, the block exclusion, and the list-content exclusion. The continuation rule is what a line-local bullet test misses (DD-4). |
| `clozeSafeText`, `hasClozeMarker`, `wrappable`, `highestClozeNumber` | same, and `orgdoc.go` | Reused verbatim for brace safety, the marker predicate, and the numbering base. |
| `containerClassLine`'s mechanism (`#+ATTR_HTML: :class …`) | same | **NOT reused** — it is a column-zero insertion, which DD-3 measures terminates a list. The class rides the emphasis element instead. |
| `splitExtraBlocks` | `internal/anki/orgdoc/extra.go` | Composition runs after it, on the remaining body. |
| `readCardOptions`, `renderError` | `internal/anki/planner/multiline.go` | Extended: one field read, one error case. |
| `validateCardOptions`, `readBoolean`, `isClozeStyle` | `internal/anki/planner/card_options.go` | Relied on. Read-only; no rule added. |
| `imoogi-anki--ensure-cloze-type` | `modules/org/24-anki.el` | Reused by the new command for the note-type contract. Its failure message is generalized to stop naming multiline options. |
| `imoogi-anki--write-card-option` | same | **NOT reused** — it runs the swift-conflict check unconditionally, which is backwards for the swift toggle's OFF path (DD-10). Either the note-type step is factored out of it, or it gains a parameter naming which conflict check to run. |
| `imoogi-props-resolve-{swift,direction,incremental}` | `modules/org/anki/imoogi-props.el` | Read by the new command. |
| `render-golden.json`, `multiline-golden` corpus, `TestRenderSignatureIsUnchanged` | `internal/anki/orgdoc` | Left byte-identical. Extended by a third corpus. |

### A.2 Baseline at `08f7504`

Measured by the delegating session, not by this plan: `go test ./... -count=1`
exit 0; coverage `orgdoc` 100.0%, `planner` 92.8%, `hashing` 100.0%, `model`
98.4%; `make lint` 0; `make fmt-check` 0; `make ci-local` 0; `make test-elisp`
exit 0 reporting `Ran 435 tests, 433 results as expected, 0 unexpected, 2
skipped`. `acceptance.md`'s **Quality Gate Criteria** table carries the
non-regression thresholds over this baseline.

### A.3 PRESERVE list

Do not modify, on any milestone:

- `internal/anki/orgdoc/testdata/render-golden.json` and the multiline golden
  corpus recorded beside it — the byte-identity evidence of the two earlier
  SPECs. Re-recording either destroys what it proves.
- Three assertions, which must stay byte-identical because each is the evidence
  for a claim an earlier SPEC makes about not changing something:
  `TestRenderSignatureIsUnchanged` and
  `TestHashCallSiteTakesFourArguments`, which must keep compiling against the
  unwidened entry point and the four-argument hash; and
  `TestOptionFreeRequestLogGolden`, which holds the no-op guarantee.
- `internal/anki/planner/card_options.go` — the validation gate and its value
  parsers. This SPEC reads them and adds no rule to them.
- `internal/anki/hashing` — the hash input set.
- `modules/org/anki/imoogi-{props,scan,process}.el` — resolution, scan, and
  serialization are complete as SPEC-ANKICARD-002 left them.
- `.moai/specs/SPEC-ANKICARD-001/`, `-002/`, `-003/`.
- **Another actor's files in this checkout**, named explicitly because they are
  modified in the working tree and are not this SPEC's:
  `modules/org/29-org-roam.el`, `tests/org-roam-test.el`,
  `modules/general/05-transient.el`, `tests/module-layout-test.el`,
  `modules/project/04-projects.el`, `tests/workspace-bridge-test.el`,
  `CLAUDE.md`, and everything under `vendor/elpa/org-roam-*`.

### A.4 Explicitly amendable — three test sites, not one

The brief named one test that this SPEC breaks. There are **three**, and all
three break for the same reason: they set the swift option on an entry whose
body carries no arrow line and expect that entry to be accepted. Until this
SPEC, swift had no rendering meaning and every such entry sailed through.

| Site | What it does | Why it breaks |
|---|---|---|
| `applyHashProbeOptions` / `TestHashIsUnchangedByCardOptions` (`internal/anki/planner/baseline_golden_test.go`) | Sets `Swift: "t"` over the body `"With a body."` and expects the add branch | No arrow line → `swift_arrow_missing` → skipped, so the probe's `Fatalf` fires |
| `TestCardOptionValueRecognition`, row `{"swift truthy", "swift", "t", false}` (`internal/anki/planner/card_options_test.go`) | Runs `optionEntry(nil, nil, "t")` over `multilineBody` and expects acceptance | Same: `multilineBody` carries a marker and a list, but no arrow |
| `TestSwiftWithMultilineOptionIsRejected`, rows `swift with falsy incremental`, `swift with falsy direction`, `swift alone` (same file) | Three non-conflicting rows with swift on, expecting acceptance | Same |

The second and third share one cause and one fix: the shared fixture
`multilineBody`. Its own docstring records that SPEC-ANKICARD-003 gave it the
answer list for exactly this reason — "so an entry using it reaches the gate's
downstream neighbours under a MULTILINE option too". Extending it with an arrow
line is the same move, and it is the last one the series needs: after this SPEC
every card kind has a rendering meaning and the fixture satisfies all of them.

The first needs a new shape, not a new option — see DD-5.

A **fourth** site does not break but must be extended, and is listed here so it
is not missed for being green:
`TestMultilineFalsyOptionsAreByteIdenticalToNoOptions`
(`internal/anki/orgdoc/multiline_golden_test.go`) names the option fields
explicitly and asserts the rendered output gains no `children-list` container.
Adding a swift field leaves it compiling and passing on the field's zero value,
so nothing fails — which is exactly why it would otherwise be overlooked. It
gains the swift field explicitly and a matching assertion that the output gains
no `swift-arrow` emphasis and no inline break, so the no-option claim covers all
three options rather than two of three.

## § B Design Decisions

### DD-1 — Arrow detection is a pre-render text pass

**Decision.** Arrow lines are found in Org source before go-org runs, and the
markers and the emphasis are written into that source.

**Why not post-render.** Measured against the vendored go-org v1.9.1, the
renderer escapes both angle brackets in ordinary text:

```
IN:  "Tokyo :-> Japan\n"          OUT: <p>Tokyo :-&gt; Japan</p>
IN:  "Tokyo :<- Japan\n"          OUT: <p>Tokyo :&lt;- Japan</p>
IN:  "Tokyo :<-> Japan\n"         OUT: <p>Tokyo :&lt;-&gt; Japan</p>
```

A post-render scanner would therefore have to match `:-&gt;`, and would be
matching a string whose relation to the source is an escaping rule rather than
an identity. Everything SPEC-ANKICARD-003 DD-1 says about post-render scanning
applies here unchanged, and one thing more: swift's spans are the two flanks of
a token, so a post-render scanner would also have to know which HTML element
boundaries it may not cross — a question the source-level form does not raise.

**Why pre-render works.** Measured, a cloze marker passes through go-org
untouched on an arrow line, including with multi-byte content:

```
IN:  "{{c2::도쿄}} @@html:<b>:-></b>@@ {{c1::일본}}\n"
OUT: <p>{{c2::도쿄}} <b>:-></b> {{c1::일본}}</p>
```

**A consequence worth stating.** Because detection is a source pass and go-org
runs afterwards, go-org may still transform the sides. Measured,
`see http://x:->y` has `http://x:-` linkified before the `>` — so a line whose
left side ends in a bare URL produces a card with an odd link in it. This is
pre-existing go-org behavior on that text, not something composition
introduces, and no requirement constrains it.

### DD-2 — The arrow's emphasis is an Org export snippet

**Decision.** The token is replaced with `@@html:<b>…</b>@@` carrying the
token's escaped text.

**Measured alternatives, and why each fails.** Raw HTML in the source is
escaped, so the tag shows on the card as characters:

```
IN:  "Tokyo <b>x</b> Japan\n"     OUT: <p>Tokyo &lt;b&gt;x&lt;/b&gt; Japan</p>
```

Org's own bold emits the wrong element:

```
IN:  "Tokyo *bold* Japan\n"       OUT: <p>Tokyo <strong>bold</strong> Japan</p>
```

Post-render substitution is DD-1's rejected approach in a second costume.

**What the snippet does.** Measured, its content passes through verbatim, and
the backend name is matched without regard to case:

```
IN:  "Tokyo @@html:<b>:-></b>@@ Japan\n"     OUT: <p>Tokyo <b>:-></b> Japan</p>
IN:  "Tokyo @@html:<b>:-&gt;</b>@@ Japan\n"  OUT: <p>Tokyo <b>:-&gt;</b> Japan</p>
IN:  "Tokyo @@HTML:<b>x</b>@@ Japan\n"       OUT: <p>Tokyo <b>x</b> Japan</p>
```

Both escaped and unescaped tokens render; `spec.md` REQ-SW-007.2 picks the
escaped form because the snippet's output is raw HTML.

**One measured hazard, and the bound that answers it.** An unrecognized backend
renders to **nothing at all** — `@@latex:<b>x</b>@@` produces `<p>Tokyo  Japan</p>`
— so a typo in the backend name would delete the arrow silently. More
seriously, the snippet bypasses escaping entirely, so any user text placed
inside one reaches the card as markup. REQ-SW-007.3 bounds the construct to the
three fixed tokens for both reasons; the emitted snippet is a constant chosen
from a three-element table, never assembled from the line's text.

**Interaction with the block exclusion, checked.** An `@@html:` snippet written
inside a `#+BEGIN_SRC` or `#+BEGIN_EXAMPLE` block would show on the card as
literal characters, because go-org renders those interiors verbatim. It cannot
happen, and the reason is in the other rule: REQ-SW-001.5 excludes block
interiors from being arrow lines, so composition never writes anything there.
The two rules are safe together and neither is safe alone — noted here because a
reader checking this decision on its own would not see the other.

**Interaction with the existing math pass, checked.** `transformMath` walks
`<`…`>` spans and copies markup through opaquely, and its protection-depth
counter moves only for `pre` and `code`. A `<b>` in the rendered output is
therefore copied unchanged and protects nothing, which is what is wanted.

### DD-3 — The layout fix is inline; composition inserts no line

**Decision (revised after plan audit iteration 1).** An inline line break is
appended to an arrow line whose successor is also an arrow line, the emphasis
element carries the `swift-arrow` class, and composition inserts no line of its
own. The `#+ATTR_HTML` container line this decision previously prescribed is
abandoned.

**The defect being fixed.** Measured, consecutive arrow lines are **one**
paragraph carrying a literal newline:

```
IN:  "도쿄 :-> 일본\n서울 :-> 한국\n"
OUT: <p>도쿄 :-&gt; 일본\n서울 :-&gt; 한국</p>
```

HTML collapses that newline to a space, so without a fix the card shows two
cards' worth of text run together on one line. This is the most likely body
shape there is, so leaving it unaddressed would ship a broken card as the
default case.

**Why the container was wrong.** The attribute line is written at column zero,
and measured it **terminates a list** when the arrow sits on an indented
continuation line:

```
IN (before)  "- 수도\n  도쿄 :-> 일본\n"
OUT          <ul><li>수도\n도쿄 :-&gt; 일본</li></ul>          list intact

IN (after)   "- 수도\n#+ATTR_HTML: :class swift-list\n  도쿄 :-> 일본\n"
OUT          <ul><li>수도</li></ul><p class="swift-list">도쿄 :-&gt; 일본</p>
```

One list item holding two lines becomes a truncated item plus a sibling
paragraph. That is a structural change to authored content — worse than the
uncarded-but-visible limitation § 5 discloses, and undisclosed. A control run
isolates the cause to the attribute line rather than to the markers:

```
IN   "- 수도\n  도쿄 @@html:<b>:-&gt;</b>@@ {{c1::일본}}\n"
OUT  <ul><li>수도\n도쿄 <b>:-&gt;</b> {{c1::일본}}</li></ul>   list intact
```

The container had a second defect of its own: it fixed where the run **opens**
and never said where it closes. Measured, the paragraph runs to the next blank
line, so a non-arrow line below an arrow line was drawn inside the styled
container — a rendering change to content REQ-SW-001.7 promises renders as it
does today.

**Why inline works.** Measured, the break rides the same export-snippet
mechanism as the emphasis, renders deterministically, and restructures nothing
— including inside a list item:

```
IN   "A @@html:<b>:-&gt;</b>@@ {{c1::B}}@@html:<br>@@\nC @@html:<b>:-&gt;</b>@@ {{c3::D}}\n"
OUT  <p>A <b>:-&gt;</b> {{c1::B}}<br>\nC <b>:-&gt;</b> {{c3::D}}</p>

IN   "- 수도\n  도쿄 @@html:…@@ {{c1::일본}}@@html:<br>@@\n  서울 @@html:…@@ {{c3::한국}}\n"
OUT  <ul><li>수도\n도쿄 <b>:-&gt;</b> {{c1::일본}}<br>\n서울 <b>:-&gt;</b> {{c3::한국}}</li></ul>
```

**Why the break is conditional on an arrow successor.** Appending it wherever
an arrow line is followed by any non-blank line would move a following prose
line onto its own line — the same defect the container had at its closing end,
in a smaller costume. Restricting it to an arrow-line successor makes
REQ-SW-001.7 true unqualified: every non-arrow line renders exactly as today.

**The styling hook survives without a container.** Measured, a class on the
emphasis element passes through the snippet intact:

```
IN   "A @@html:<b class=\"swift-arrow\">:-&gt;</b>@@ B\n"
OUT  <p>A <b class="swift-arrow">:-&gt;</b> B</p>
```

So the stylesheet keeps a target and the base-stylesheet precedent is honoured,
with no structural insertion. What is given up is a class on the arrow
**lines** as a group; nothing in this SPEC needs one, and adding one later
belongs on the note type's template rather than in composition.

**Two properties gained by the change, worth naming.** The card is now correct
without any stylesheet rule at all — the raw `Text` field carries the break, so
a browser preview or a third-party viewer shows the lines apart, which the CSS
form did not. And the residual risk about a `white-space`-style declaration
inside Anki's webview disappears, because no such declaration is needed.

### DD-4 — An arrow line is paragraph text, stated definitionally

**Decision (revised after plan audit iteration 1).** An arrow line must be a
line the renderer's own parser reads as ordinary paragraph text, derived from
the parser's rules rather than matched against a list of known line shapes. The
former pair of exclusions — block interior, list item — is replaced by that one
positive rule.

**Why the exclusions were not enough.** They were an enumeration, and an
enumeration misses what nobody thought of. Two classes were missed.

A list **continuation** line carries no bullet, so a line-local bullet test
admits it while go-org reads it as list content. Measured, `- 수도` followed by
an indented `도쿄 :-> 일본` is one list item holding two lines. Under the
abandoned container mechanism this was the D12 corruption (DD-3); under the
inline mechanism nothing breaks, but the line still becomes a card inside a
list item, which is neither what the exclusion intended nor what § 5 discloses.

Three further shapes match the expression and are not paragraph text. Measured
in this SPEC's own re-probe, not carried over from the audit:

```
IN "| A :-> B |\n"      OUT <table><tbody><tr><td>A :-&gt; B</td></tr></tbody></table>
IN "* A :-> B\n"        OUT <h2 id="headline-1">A :-&gt; B</h2>  (plus nav)
IN "#+TITLE: A :-> B\n" OUT <h1 class="title">A :-&gt; B</h1>
IN "# A :-> B\n"        OUT (empty)
```

The table row is the sharpest: the right side is `B |`, so wrapping it swallows
the cell delimiter into a marker. The comment line is the quietest: it renders
to **nothing at all**, so an arrow composed there yields a note whose only
markers are invisible, which Anki then refuses as a cloze note with no
deletion — a skip with a diagnostic pointing at Anki rather than at the line.

**Why definitional rather than a longer list.** Exactly SPEC-ANKICARD-003
REQ-ML-001.3's reasoning, applied to a different fact. A list pins the SPEC to
one parser version and goes stale silently; a definitional rule with an
informative, version-pinned list is falsifiable and survives a parser upgrade.

**The first draft of this decision proved the point against itself.** It ended
with "only the table, heading, keyword, and comment shapes are new, and each is
a single line-level rule" — a closed enumeration of four, in the same decision
whose whole argument is that enumerations miss what nobody thought of. Two
further shapes were then measured outside it:

```
IN ": A :-> B\n"        OUT <pre class="example">\nA :-&gt; B\n</pre>
IN "[fn:1] A :-> B\n"   OUT ""            (renders to nothing at all)
```

The fixed-width line reaches the block interior's hazard without a block; the
footnote definition reaches the comment line's silent hazard. The enumeration is
struck and no replacement list is offered.

**The implementable route, measured.** Parsing a single line and reading the
parser's own node classification answers correctly for every context-free shape:

```
"A :-> B"        -> org.Paragraph      "| A :-> B |"      -> org.Table
"* A :-> B"      -> org.Headline       "#+TITLE: A :-> B" -> org.Keyword
"# A :-> B"      -> org.Comment        ": A :-> B"        -> org.Example
"[fn:1] A :-> B" -> org.FootnoteDefinition
```

Recorded as **one route measured to work**, not as the mandated implementation.
go-org's public AST carries no source positions, so a position-mapped route is
not available and the SPEC does not ask for one.

**The route has a hole, and it is not in the audit's account of it.** The
per-line parse is context-blind, which is why the scanner is still needed for
the context-dependent classes — and there are **three**, not two. Measured:

```
"  도쿄 :-> 일본"  alone -> org.Paragraph ;  after "- 수도"      -> org.List
"x :-> y"          alone -> org.Paragraph ;  inside #+BEGIN_SRC -> org.Block
"A :-> B"          alone -> org.Paragraph ;  inside \begin{align} -> (LatexBlock interior)
```

A multi-line `\begin{align}` environment is `org.LatexBlock`, and go-org passes
its interior through **raw** — measured, a snippet composed there shows as
literal characters exactly as in a source-block interior. The existing pairing
rule matches `#+BEGIN_`/`#+END_` only, so it does not see this delimiter, and
the per-line parse calls the interior line a paragraph. REQ-SW-001.5 therefore
widens the pairing to the `\begin{…}` form. The one-line
`\begin{align} … \end{align}` is genuinely a paragraph and stays admitted.

**Why list content is excluded at all.** A bullet line's content begins at an
offset the parser computes, which REQ-ML-001.3 makes definitional and entangles
with three consumptions. The sharpest collision is measured:

```
IN:  "- Tokyo :: Japan :-> x\n"
OUT: <dl><dt>Tokyo</dt><dd>Japan :-&gt; x</dd></dl>
```

The description term's `::` and the arrow both claim the line. Whichever wins,
the other's author is surprised. `spec.md` § 5 names what the exclusion costs,
refuses the narrower variant the audit offered, and states why.

**Why the block rule is reused rather than narrowed.** Measured, the three
block kinds differ:

```
#+BEGIN_SRC go     → <div class="src src-go">…<pre>x :-&gt; y</pre>…
#+BEGIN_EXAMPLE    → <pre class="example">x :-&gt; y</pre>
#+BEGIN_QUOTE      → <blockquote><p>x :-&gt; y</p></blockquote>
```

A marker inside the first two shows its braces on the card as literal text; a
marker inside the third would work. Narrowing the exclusion to the verbatim
kinds would put a second `#+BEGIN_`/`#+END_` pairing rule in one package, and a
divergence between the two would be invisible until a card rendered wrong. One
rule, one cost, stated.

### DD-9 — A hand-written marker spanning the arrow needs an extent scan

**Decision (added after plan audit iteration 1).** A line whose first arrow
token lies inside a hand-written marker's span is not an arrow line, decided by
a marker-**extent** scan rather than by the existing presence predicate.

**The defect.** REQ-SW-005's per-side guard cannot see a marker that spans the
arrow, because each flank holds only a fragment of it. Composing
`{{c1::도쿄 :-> 일본}}` under the other rules gives left `{{c1::도쿄` (contains
a marker prefix, not wrapped) and right `일본}}` (contains none, wrapped at
`c2`), producing:

```
{{c1::도쿄 @@html:<b>:-&gt;</b>@@ {{c2::일본} } }}
```

Anki's non-greedy close then yields **one** card whose visible text is
`{{c2::일본} }`. The author's deletion is destroyed. `imoogi-anki-cloze-region`
writes exactly this shape when the user selects a whole line, and `ANKI_SWIFT`
inherits, so the shape is reachable on headings marked before the option
existed — a regression, not a new-authoring edge case.

**Why an extent scan.** The existing predicate matches a marker's opening
prefix and answers "is there a marker somewhere", not "does this offset sit
inside one". Measured with a span scan that opens at `{{cN::` and closes at the
first `}}` — Anki's own rule, not a second one:

```
"{{c1::도쿄 :-> 일본}}"    spans=[[0 25]]        arrow@13  inside=true   -> excluded
"{{c1::도쿄}} :-> 일본"    spans=[[0 14]]        arrow@15  inside=false  -> arrow line
"도쿄 :-> {{c1::일본}}"    spans=[[11 25]]       arrow@7   inside=false  -> arrow line
"{{c1::A}} :-> {{c2::B}}"  spans=[[0 9],[14 23]] arrow@10  inside=false  -> arrow line
```

**The unclosed opening.** An `{{cN::` with no `}}` produces no Anki card on its
own, so it is tempting to treat it as no span. That is wrong here: a generated
marker written below it supplies the `}}` it lacks, bringing the author's
dangling opening to life around content they never marked. It is therefore
treated as spanning to the end of the remaining body. SPEC-ANKICARD-003 carries
the same shape unaddressed on the multiline path; closing it there is not this
SPEC's to do, and the asymmetry is disclosed rather than silently introduced.

### DD-5 — The hash probe changes shape, not option

**Decision.** The probe carries every option field in its **falsy** spelling,
and gains a sibling asserting that an **on** option does change the hash.

**Why the old shape cannot survive.** `applyHashProbeOptions` documents its own
requirement: the option it sets must be valid, non-conflicting, reach the add
branch, and have **no rendering meaning**. Its docstring names the exit
condition in capitals — "WHEN t15 GIVES SWIFT A RENDERING MEANING, THIS PROBE
NEEDS THE SAME TREATMENT AGAIN — and at that point no card option will be
rendering-free, so the probe will need a different shape rather than a different
option." This SPEC is that point.

**The new shape.** The falsy spelling is present on the wire — a non-nil pointer
to the string `"nil"` — and resolves off, so the entry passes the gate, renders
through the unwidened path byte-identically, and reaches the add branch. The
probe's claim survives intact: a card-option **field** carried on the entry does
not reach `hashing.Hash` as an input of its own. It stays discriminating,
because an implementation that added an option field to the hash's inputs would
hash a nil pointer differently from a pointer to `"nil"`.

**What it stops covering, and the sibling that covers it.** The old probe also
implied that an option being **on** left the hash alone. That implication is now
false by design: an on option changes the rendered field value, and the rendered
value is one of the four hash inputs and always was. So the probe gains a
sibling asserting that an on swift option over an arrow-bearing body produces a
**different** hash from the same entry without it.

**The pair is not sufficient; the contract is a trio** (plan audit iteration 1,
finding D4, accepted). The two halves are individually necessary, and the
reshaping is right, but together they still admit one wrong implementation:
feeding the **resolved boolean** to the hash while ignoring the wire field
passes half one (both sides resolve off, so both hash alike) and passes half two
(the two sides are supposed to differ). What excludes it is the compile-time
assertion that the hash call site takes exactly its four existing arguments —
`TestHashCallSiteTakesFourArguments`, which already exists and which § A.3
preserves, and which no new parameter can survive.

So the three legs are: falsy-versus-absent identity excludes hashing the wire
field; on-versus-absent difference excludes ignoring the option entirely; and
the four-argument compile assertion excludes introducing a new parameter for it.

**Three evasions, not every evasion.** The claim is deliberately pitched there.
Folding the resolved value into one of the four **existing** arguments — into
the sorted tags, say — passes all three legs while REQ-SW-012.2 forbids it. The
route is low-reachability (the hashing package is on the § A.3 PRESERVE list and
marked unchanged), so it is recorded rather than guarded; but stating the trio
as exhaustive would be the same overclaim, one level up, that the pair was
struck for. The narrowing and all three legs belong in
the helper's own docstring, where the next reader of the probe will be, not only
here.

### DD-6 — Numbering is positional, and the hand-written offset applies to the formula

**Decision.** The i-th arrow line numbers its right side `c(base + 2i − 1)` and
its left side `c(base + 2i)`, where `base` is the highest number any
hand-written marker in the entry carries, or zero.

**Why positional rather than dense.** A side the token leaves unwrapped still
consumes its number. Packing numbers over the wrapped sides only would make a
line's numbers depend on the tokens of the lines above it, so inserting one
`:<-` line near the top would renumber every card below it — and Anki keys a
card's review history to its cloze number, so the user would lose it.

**Why the offset applies to the whole formula.** SPEC-ANKICARD-003 REQ-ML-006.2
starts generated numbering above the author's highest marker, for a reason that
binds here identically: a generated marker colliding with a hand-written one
destroys the author's card. Adding `base` to both halves of the formula rather
than to the first number alone is what keeps the positional property intact —
the gap between a line's two numbers, and between consecutive lines, is
unchanged.

### DD-7 — No blank-run collapse

**Decision.** Swift composition does not collapse blank-line runs.

SPEC-ANKICARD-003 REQ-ML-009.2 collapses them because a removed supplementary
block leaves a gap wide enough to split one authored answer **list** in two,
silently dropping every answer after it. Swift reads independent lines, so a
wider gap yields more paragraphs and loses nothing — measured:

```
IN:  "A :-> B\n\ntext\n\nC :-> D\n"
OUT: <p>A :-&gt; B</p><p>\ntext</p><p>\nC :-&gt; D</p>
```

Both arrow lines are still arrow lines. Declining to reuse the collapse is a
smaller surface rather than a gap: the failure it guards against is unreachable
here, and the collapse carries a cost of its own (it merges two deliberately
adjacent authored lists) that swift would pay for no benefit.

### DD-8 — The editor command is in scope, against the card's own file list

**Decision.** REQ-SW-013 delivers the swift toggle and the completion candidate.

The backlog card t15 lists its files as `internal/anki/orgdoc/swift.go +
imoogi-error.el + README` — no Elisp command. SPEC-ANKICARD-003, however,
excluded both the swift command and the property's completion candidacy with the
words "Both belong with t15", and its `imoogi-anki-property-names` docstring
says so in the code. Honouring the card's file list would leave both owned by
nobody. The command is also nearly free: `imoogi-anki--write-card-option`
already performs the note-type contract, and the multiline commands already
carry a swift-conflict check whose mirror is what the swift command needs.

### DD-10 — The swift command does not reuse the card-option write helper

**Decision (added after plan audit iteration 1).** The new command reaches the
note-type contract through `imoogi-anki--ensure-cloze-type` directly, not
through `imoogi-anki--write-card-option`.

**Why.** That helper runs the swift-conflict check unconditionally, before the
note-type check:

```elisp
(when (and (imoogi-anki--resolve-swift-conflict)
           (imoogi-anki--ensure-cloze-type))
  (org-set-property property value) t)
```

`imoogi-anki--resolve-swift-conflict` resolves `ANKI_SWIFT` with inheritance
and, when it is on, prompts whether to turn swift off. For the multiline
commands that is exactly right. For the swift toggle it is backwards on the
path `spec.md` REQ-SW-013.2 specifies: turning swift **off** happens precisely
when the resolved value is on, so every off would first be asked whether to turn
swift off, and a user who declined could never turn it off at all.

**Two shapes are acceptable** and the choice is the implementer's: factor the
note-type step out of the write helper so each command composes its own
conflict check, or give the write helper a parameter naming which check to run.
What is not acceptable is calling it as it stands.

**The OFF path also escapes the note-type contract** (audit iteration 2, N5).
REQ-SW-013.3 now binds that contract to the ON direction only. Turning swift off
writes the falsy spelling whatever the note type is and creates no note type:
otherwise a heading carrying `imoogi-Basic` with an inherited swift-on could
never opt out through the command, which is the heading the validation gate is
already rejecting and telling the user to fix. The shape is inherited — the
incremental toggle's off path runs the same route — so this SPEC resolves it for
its own command and leaves the existing ones alone.

**Two smaller corrections ride along.** The note-type helper's failure message
reads "멀티라인 옵션을 달 수 없습니다", which is wrong for a swift heading and is
generalized to name the card option rather than the multiline pair. And the ERT
case for the off path must **not** stub the confirmation prompt to always-yes:
a stubbed prompt passes while the prompt still fires, which is the defect
itself.

## § C Pre-flight

Run before any code change, as one parallel batch:

```bash
git rev-parse HEAD && git branch --show-current
go build ./... && GOOS=windows GOARCH=amd64 go build ./...
go test ./... -count=1
go test -cover ./internal/anki/...
make lint ; make fmt-check
make test-elisp
grep -rn "swift_arrow_missing\|REQ-SW-" internal/ modules/ tests/ || echo "namespace free"
```

The last line must report the namespace free; a hit means this SPEC's identifier
space is already in use and the plan is wrong before it starts.

## § D Constraints

1. Never modify the A.3 PRESERVE list. The other-actor files there belong to a
   concurrent session and touching them loses their work.
2. Run no git command that writes to the shared checkout — no `add`, no
   `commit`, no branch change, no `stash`. Where a commit is wanted, work in a
   worktree.
3. Stage by explicit pathspec. `git add -A`, `git add .`, and `git commit -a`
   are forbidden in this checkout for the same reason as constraint 1.
4. Conventional Commits, `feat(SPEC-ANKICARD-004): M{N} <subject>`, with the
   `🗿 MoAI` trailer. Never `--no-verify`, never `--amend` on a pushed commit.
5. Add no rule to `validateCardOptions`. The gate's fixed order is
   SPEC-ANKICARD-002's, and REQ-SW-010 places the new code outside it.
6. Add no parameter to `hashing.Hash` and no second renderer entry point.
7. Place nothing inside an Org export snippet but the three fixed arrow tokens.
8. Match each file's existing comment language: the Go packages are commented in
   English, `modules/org/24-anki.el` and `tests/anki-*.el` in Korean.
9. Insert no line into a body during composition. Every construct written is
   inline (`spec.md` REQ-SW-011.3, DD-3).
10. Decide "is this offset inside a marker" with an extent scan, never with the
    presence predicate (DD-9).

## § E Self-Verification

Report each item per the five-section evidence format (Claim / Evidence /
Baseline-attribution / Gaps / Residual-risk), naming the exact command, its
verbatim output, and the HEAD the evidence was captured against.

- **E1** — the AC matrix from `acceptance.md`, binary PASS/FAIL, one row per
  criterion with its deciding command and that command's actual output.
- **E2** — cross-platform build: `go build ./...` and
  `GOOS=windows GOARCH=amd64 go build ./...`, both exit 0.
- **E3** — coverage per package against the A.2 baseline; `orgdoc` must not
  fall below 100.0%.
- **E4** — the three byte-identity corpora, each reported separately:
  `render-golden.json` and the multiline corpus unmodified (`git diff --stat`
  on those paths, empty), the swift corpus newly recorded.
- **E5** — lint and format status, NEW issues distinguished from the A.2
  baseline.
- **E6** — `make test-elisp`, with the test count compared against the A.2
  baseline of 435.
- **E7** — the RED failing output captured **before** each GREEN, verbatim.
- **E8** — any blocker, as a structured report. Never a question to the user.

## § F Milestones

Priority-ordered, decisions first. Each milestone is one commit.

### M1 — The arrow rule (highest change likelihood)

`internal/anki/orgdoc/swift.go` and its tests. The scanner, the numbering, the
side selection, the marker rules including the extent scan, the emphasis, and
the typed arrow-missing error. Reuses `splitLines`, `blockRegions`,
`readBullet`, `scanAnswerList`'s list-continuation rule, `clozeSafeText`,
`wrappable`, and `highestClozeNumber` from `multiline.go` without copying them;
the marker-extent scan, and the table / heading / keyword / comment line
classification, are the only genuinely new pieces.

Covers REQ-SW-001 through REQ-SW-007, REQ-SW-008.2.

### M2 — The dispatch and the pipeline

`CardOptions` gains the swift option; `RenderWithOptions` gains its branch and
its totality rule; the planner's option reader and render-error mapping gain
their one case each; `protocol` gains the code.

Covers REQ-SW-008.1, REQ-SW-008.3, REQ-SW-009, REQ-SW-010.

### M3 — Non-interference

The swift golden corpus; the reshaped hash probe and its new sibling; the
extended shared option fixture; the three amendable test sites of A.4 brought
green, and A.4's fourth site extended to name the swift option. Re-run the two
preserved corpora and show them unmodified.

Covers REQ-SW-012.

### M4 — Presentation

The `swift-arrow` rule in `internal/anki/model/assets/base.css`, and whatever
whole-file assertions that stylesheet already carries re-run. The inline break
and the class itself land in M1 with the rest of composition; this milestone is
the stylesheet alone.

Covers REQ-SW-011.2.

### M5 — Front end

`modules/org/anki/imoogi-error.el` gains the message; `tests/anki-error-test.el`
gains the code on both sides of its pairing, including the hard-coded back-end
list; `modules/org/24-anki.el` gains the toggle, its binding, its transient
entry, the mirror conflict check that clears every on multiline option, the
generalized note-type failure message, and the completion candidate; ERT
coverage for each, including an OFF-path case that does **not** stub the
confirmation prompt (DD-10).

Covers REQ-SW-013, REQ-SW-014.

### M6 — Documentation (most mechanical)

`README.md`'s Anki section: the property, what an arrow line is and what is not
one, the three tokens, the numbering, and how many cards a heading yields.

## § G Anti-Patterns

1. **Scanning rendered HTML for an arrow.** DD-1. The token is escaped there.
2. **Writing `<b>` into Org source.** DD-2. It renders as `&lt;b&gt;`.
3. **Assembling an export snippet from the line's text.** REQ-SW-007.3. The
   snippet's content is raw HTML.
4. **Copying the bullet grammar or the block pairing into `swift.go`.** A.1.
   Both exist; a second copy diverges silently.
5. **Adding a third renderer entry point.** REQ-SW-009.1. Delegation is what
   makes the no-option path structural.
6. **Adding a fourth rule to `validateCardOptions`.** REQ-SW-010.
7. **Re-recording `render-golden.json` or the multiline corpus to make a test
   pass.** A.3. The recording is the evidence.
8. **Solving the hash probe by finding another rendering-free option.** DD-5.
   There is none left; the shape must change.
9. **Packing cloze numbers densely over the wrapped sides.** DD-6. It renumbers
   cards under insertion and Anki loses their histories.
10. **Reusing the blank-run collapse "for consistency".** DD-7. It costs
    something and buys nothing here.
11. **Letting a list arrow through "since it is obviously intended".** DD-4.
    The description term's `::` claims the same line, and a continuation line
    carries no bullet to test.
12. **Enumerating line shapes instead of classifying as paragraph text.** DD-4.
    The enumeration missed four shapes on its first pass.
13. **Inserting a line at column zero to attach a class.** DD-3, REQ-SW-011.3.
    It terminates a list.
14. **Deciding "inside a marker" with the presence predicate.** DD-9. It
    answers a different question.
15. **Calling `imoogi-anki--write-card-option` from the swift toggle.** DD-10.
    Its conflict check is backwards for this command.
16. **Stubbing the confirmation prompt to always-yes in the OFF-path test.**
    DD-10. The stub passes while the prompt still fires.
17. **Touching another actor's files.** A.3, § D constraint 1.

## § H Cross-References

- `spec.md` — the requirements these milestones implement.
- `acceptance.md` — the Given-When-Then criteria and the corpus.
- `.moai/specs/SPEC-ANKICARD-003/{spec,plan,acceptance}.md` — the mechanisms
  § A.1 reuses, and the go-org evidence DD-1 and DD-4 build on.
- `.moai/reports/anki-card-types-plan-20260920.md` § 2.2 and § 2.3 — the
  user-confirmed generation rules and decision C.
- `.moai/reports/plan-audit/SPEC-ANKICARD-003-plan-audit{,-r2,-r3}.md` — the
  three audit iterations whose measured go-org findings this plan does not
  re-derive.
