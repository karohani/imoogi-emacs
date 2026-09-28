# Implementation Plan — SPEC-ANKICARD-003

Ordered by decision-reversibility. § B carries the design decisions, hardest to
reverse first; § F carries the milestones, mechanical work last. A reviewer
with limited attention should spend it on § B.

## § A Context

### A.1 What already exists

Commit `6f3ae6c` (SPEC-ANKICARD-002) landed the transport and the validation.
This SPEC consumes it and adds no second copy of any of it:

| Surface | Where | This SPEC's use |
|---|---|---|
| `Direction`, `Incremental`, `Swift` on the entry, nullable, verbatim | `internal/anki/protocol` | Read. `nil` means the chain resolved to nothing. |
| `readDirection`, `readBoolean`, `isClozeStyle`, the value constants | `internal/anki/planner/card_options.go` | Read. Trimming and case rules live there; no second parser. |
| `validateCardOptions`, called before the render on both paths | `internal/anki/planner/{planner,migrate}.go` | Relied on. Composition receives only entries that passed it. |
| `splitExtraBlocks` | `internal/anki/orgdoc/extra.go` | Composition runs after it, on the remaining body. |
| `imoogi-props-resolve-{direction,incremental,swift}` | `modules/org/anki/imoogi-props.el` | Read by the new editor commands. |
| `render-golden.json` + `TestRenderSignatureIsUnchanged` | `internal/anki/orgdoc` | Left byte-identical. Extended by a separate corpus. |
| `imoogi-anki--cloze-safe-text` and its pad | `modules/org/24-anki.el` | The brace rule the Go composer mirrors. |

### A.2 Baseline at `6f3ae6c`

Measured by the delegating session, not by this plan: `go test ./... -count=1`
exit 0; coverage `orgdoc` 100.0%, `planner` 92.5%, `hashing` 100.0%, `model`
98.4%; `make lint` 0; `make fmt-check` 0; `make test-elisp` exit 0 reporting
`Ran 422 tests, 420 results as expected, 0 unexpected, 2 skipped`.
`acceptance.md` AC-ML-015 is the non-regression criterion over this baseline.

### A.3 PRESERVE list

Do not modify, on any milestone:

- `internal/anki/orgdoc/testdata/render-golden.json` — the byte-identity
  evidence. Re-recording it destroys what it proves.
- Three assertions, which must stay byte-identical because each is the
  evidence for a claim this SPEC makes about not changing something:
  `TestRenderSignatureIsUnchanged` (`internal/anki/orgdoc/golden_baseline_test.go`),
  which must keep compiling against the unwidened entry point;
  `TestHashCallSiteTakesFourArguments` and `TestOptionFreeRequestLogGolden`
  (`internal/anki/planner/baseline_golden_test.go`), which hold REQ-ML-014's
  hash-input and no-op guarantees.

  **`TestHashIsUnchangedByCardOptions` and its `applyHashProbeOptions` helper,
  in that same planner file, are explicitly amendable** — the file is not
  preserved wholesale. That test builds a probe entry carrying `direction` and
  `incremental` over a body with no list and expects it to reach the add
  branch, which REQ-ML-002 now correctly rejects with
  `multiline_answer_missing`. The helper's own docstring names itself "the ONE
  place `TestHashIsUnchangedByCardOptions` needs to change as the wire
  contract grows", and states that its options are chosen so the entry passes
  the gate and reaches the add branch — so re-choosing them is the change the
  helper was written for. The probe's subject is the hash, not the card kind,
  so any valid non-conflicting option set serves; `swift` alone is the natural
  choice, being the one option with no rendering meaning until card t15.
- `internal/anki/planner/card_options.go` — the validation gate and its value
  parsers. This SPEC reads them and adds no rule to them.
- `internal/anki/hashing` — the hash input set.
- `modules/org/anki/imoogi-{props,scan,process}.el` — resolution, scan, and
  serialization are complete as SPEC-ANKICARD-002 left them.
- `.moai/specs/SPEC-ANKICARD-001/`, `.moai/specs/SPEC-ANKICARD-002/`.
- The uncommitted working-tree changes present at plan time (`CLAUDE.md`,
  `modules/project/04-projects.el`, `tests/workspace-bridge-test.el`) and every
  untracked path under `.moai/`.

## § B Design Decisions

### DD-1 — Composition is pre-render, at the Org source level

**Decision.** The generated markers are written into the Org source before
go-org runs, not into the rendered HTML afterward.

**Why not post-render.** Two measured facts make HTML-level wrapping wrong, not
merely awkward. Both were probed against the vendored go-org v1.9.1.

A nested list renders **inside** its parent item, so the parent's closing tag
comes after the nested list's:

```
IN:  "- Tokyo\n  - Kanto\n- Osaka\n"
OUT: <ul>
     <li>
     <p>Tokyo</p>
     <ul>
     <li>Kanto</li>
     </ul>
     </li>
     <li>Osaka</li>
     </ul>
```

Finding a depth-one `<li>` in that output requires tracking tag depth across
the whole fragment — which is writing an HTML parser, the thing the post-render
approach was supposed to avoid.

Worse, a description list emits **no `<li>` at all**:

```
IN:  "- term :: definition\n"
OUT: <dl>
     <dt>
     term
     </dt>
     <dd>definition</dd>
     </dl>
```

A post-render `<li>` scanner does not mis-handle this shape; it silently misses
it, producing an unwrapped card with no signal. Silent misses are the failure
mode this SPEC can least afford, because the user's evidence is a card that
simply does not ask the question.

**Why pre-render works.** Measured: a marker written into Org source passes
through go-org untouched, in a plain item, in a nested parent item, in an
ordered item, and inside a description list's definition half.

```
IN:  "- {{c1::Tokyo}}\n  - Kanto\n- {{c2::Osaka}}\n"
OUT: <ul><li><p>{{c1::Tokyo}}</p><ul><li>Kanto</li></ul></li><li>{{c2::Osaka}}</li></ul>
```

No escaping, no reflow, no renumbering. A fourth argument reinforces it: the
existing `hasClozeMarker` gate already reads **raw source**, so composing
before the gate lets that function be reused unchanged on the composed
fragment, which is what makes `spec.md` REQ-ML-009.3 a one-line consequence
rather than a new code path.

**The container rides the same mechanism.** `spec.md` REQ-ML-012 needs the
answer list wrapped in an element carrying `children-list`, which post-render
would mean injecting an attribute into a tag. Pre-render it is one prepended
Org line, and measured, it attaches to **all three** list forms:

```
IN:  "#+ATTR_HTML: :class children-list\n- term :: def\n"
OUT: <dl class="children-list">…</dl>

IN:  "#+ATTR_HTML: :class children-list\n1. A\n2. B\n"
OUT: <ol class="children-list"><li>A</li><li>B</li></ol>

IN:  "#+ATTR_HTML: :class children-list\n- A\n  - sub\n- B\n"
OUT: <ul class="children-list"><li><p>A</p><ul><li>sub</li></ul></li><li>B</li></ul>
```

That the description form carries the class too is what keeps REQ-ML-001.3 from
producing an unstyled card. A `#+BEGIN_…` special block was measured as the
alternative and rejected: go-org truncates the block name at a hyphen and
appends its own suffix, so `#+BEGIN_CHILDREN-LIST` yields
`<div class="children-block">` — the wrong class, and an extra wrapper.

**Mechanism, and the probe that constrains it.** `org.ListItem` in v1.9.1
carries `Bullet`, `Status`, `Value`, `Children` — and **no source position**.
The AST therefore cannot map an item back to the source line that produced it,
so composition cannot be "parse, then edit the source at the item's offset".

The mechanism is a source-level scanner whose bullet grammar mirrors go-org's
own (`^(\s*)([+*-])(\s+(.*)|$)` and the ordered form, whose terminators are
both `.` and `)`), with the answer list's base indentation taken from its first
bullet line and any more-indented line attributed to the preceding item. Four
rules the bare grammar does not carry, each measured:

- **Block interiors are skipped.** go-org does not read a bullet line inside a
  block as a list item, so the scanner must not either. `#+BEGIN_SRC` and
  `#+BEGIN_EXAMPLE` interiors render as `<pre>`, and an `#+BEGIN_EXAMPLE` body
  of bullets yields **no list at all**. A scanner that counted them would take
  the answer list from inside a code sample and inject a cloze marker into it.
- **A list nested inside a block is not top level.** `#+BEGIN_QUOTE` with
  bullets renders `<blockquote><ul>…</ul></blockquote>` — a real list, but not
  one at the body's top level.
- **A blank-line run does not end the list.** One blank line between items
  keeps one list; two split it. Composition collapses runs to one before the
  scanner runs (DD-4), so after that step no blank line ends a list, and the
  scanner ends the answer list at the first line that is neither a bullet at
  the base indent nor a continuation of the preceding item.
- **The base indent comes from the first bullet line, not from column zero.**
  An indented first bullet (`  - A`) is a top-level list to go-org and must
  not be read as nested.

The parse is still used, for two things the scanner must not guess: whether
the first list is description-kind (the AST exposes `DescriptiveListItem` as a
distinct type) and how many top-level items it has.

**The cross-check that makes this falsifiable.** For every body in the test
corpus, the scanner's item count must equal the parse's top-level item count.
A disagreement means the scanner and the renderer read the same document
differently, which is exactly the class of bug a hand-rolled scanner invites.
Assert it as a property test over the corpus, not as a runtime check — at
runtime there is no correct action to take on disagreement, and silently
rendering unwrapped is the outcome DD-1 rejects post-render for.

**Reversibility.** High cost to reverse. Every other decision below assumes
source-level composition.

### DD-2 — A nested sub-item is not wrapped

**Decision.** A generated marker covers an answer item's own content and stops
before its nested children (`spec.md` REQ-ML-001.1).

**The alternative, and why it is rejected.** Extending the marker over the
children means the closing `}}` lands after the nested list. Measured:

```
IN:  "- {{c1::Tokyo\n  - Kanto}}\n"
OUT: <ul><li><p>{{c1::Tokyo</p><ul><li>Kanto}}</li></ul></li></ul>
```

The marker survives, but it now spans `</p><ul><li>` — intervening markup
inside a cloze deletion. Whether Anki renders that acceptably is unknown here
and the design record § 5 independently lists "multi-line HTML inside a cloze"
as a medium risk requiring a real-Anki check this SPEC does not perform.
Shipping a shape whose correctness is unverified, in order to hide slightly
more text, is the wrong trade.

**Consequence, stated rather than hidden.** A sub-item elaborating its answer
stays visible on the question side. `spec.md` § 5 records this as a deliberate
exclusion so the next reader does not file it as a bug.

**Reversibility.** Moderate. Changing it later changes rendered output for
entries that have nested answers, which re-hashes those notes.

### DD-3 — The wrapped span excludes whatever prefix the parser reads first

**Decision.** For `- term :: definition`, the answer is `definition`
(`spec.md` REQ-ML-001.3).

**Why not the whole item.** go-org splits a description item at the first
` :: `, and its splitter regexp is `\s::(\s|$)`. Wrapping the whole item puts
the generated marker's own `::` on the wrong side of that split. Measured:

```
IN:  "- {{c1::term :: definition}}\n"
OUT: <dl><dt>{{c1::term</dt><dd>definition}}</dd></dl>
```

The marker is torn in half across `<dt>` and `<dd>`. Anki would render an
unclosed marker literally — a visibly broken card.

**Why not exclude description items entirely.** Definition-only wrapping is
measured clean:

```
IN:  "- term :: {{c1::definition}}\n"
OUT: <dl><dt>term</dt><dd>{{c1::definition}}</dd></dl>
```

and it is the semantically obvious reading: the `::` form already states which
half is the answer. Excluding the form would mean an author who wrote their
answers in the most explicit Org shape available gets no card.

**A hazard checked and found absent.** Because the splitter requires
whitespace before `::`, a generated `{{c1::Tokyo}}` in a **plain** item does
not accidentally turn that list into a description list. Measured:
`- {{c1::Tokyo}}` renders as `<ul><li>{{c1::Tokyo}}</li></ul>`.

**The same hazard, two more shapes.** The audit found that the rule above was
scoped to the one shape this SPEC's original probe set reached. Two more carry
a parser-consumed prefix, and both re-measured here:

A **checkbox** item is worse than the description case, because the marker's
opening is destroyed rather than merely displaced:

```
IN : "- {{c1::[ ] A}}\n"
OUT: <ul><li class="unchecked">::[ ] A}}</li></ul>
```

go-org reads the status before the item content and consumes `{{c1` with it.
Wrapping after the checkbox is clean, for both states:

```
IN : "- [ ] {{c1::A}}\n"   OUT: <ul><li class="unchecked">{{c1::A}}</li></ul>
IN : "- [X] {{c1::A}}\n"   OUT: <ul><li class="checked">{{c1::A}}</li></ul>
```

A **plain sibling inside a description-kind list** has no definition half at
all. Classification is per list, not per item, so one `::` anywhere makes every
sibling descriptive and the plain one is emitted with a literal `?` term:

```
IN : "- term :: def\n- plain\n"
OUT: <dl><dt>term</dt><dd>def</dd><dt>?</dt><dd>plain</dd></dl>
```

Wrapping such an item whole is measured clean:

```
IN : "- term :: {{c1::def}}\n- {{c2::plain}}\n"
OUT: <dl><dt>term</dt><dd>{{c1::def}}</dd><dt>?</dt><dd>{{c2::plain}}</dd></dl>
```

**A fourth consumption, and why the v0.1.1 wording was still wrong.** The
iteration-2 audit found the ordered-list **counter cookie**, which the v0.1.1
text's "three shapes exhaust the cases" denied. Re-measured here:

```
IN : "1. [@5] A\n2. B\n"          OUT: <ol><li value="5">A</li><li>B</li></ol>
IN : "1. {{c1::[@5] A}}\n"         OUT: <ol><li value="5">:[@5] A}}</li></ol>
IN : "1. [@5] {{c1::A}}\n"         OUT: <ol><li value="5">{{c1::A}}</li></ol>
IN : "- [@5] {{c1::A}}\n"          OUT: <ul><li>[@5] {{c1::A}}</li></ul>
```

The cookie is consumed on ordered lists only — the unordered row leaves it in
the text — and on every ordered terminator (`1.`, `1)`, `a.` all measured).
Stacked with a status it still wraps clean:
`1. [@5] [ ] {{c1::A}}` → `<li value="5" class="unchecked">{{c1::A}}</li>`.

**The model itself was wrong, not just the count.** Reading the vendored
parser rather than probing it: `listItemValueRegexp` and `listItemStatusRegexp`
are **unanchored**, and the parser then slices `content[len("[@] ")+len(m[1]):]`
and `content[len("[ ] "):]` — a fixed byte count **from position zero**,
wherever the match occurred. So there is often no leading prefix at all:

```
IN : "- Tokyo [ ] is big\n"           OUT: <li class="unchecked">o [ ] is big</li>
IN : "- Tokyo :: a [ ] b\n"           OUT: <dt class="unchecked">o</dt><dd>a [ ] b</dd>
```

Four characters vanish, and the second row shows it reaches a description
term too. This is a **pre-existing go-org defect** — the first row is what the
tree renders today, with or without this SPEC — and fixing it is out of scope.
What is in scope is that the v0.1.1 phrase "structural prefix … ahead of that
content" describes a parser that does not exist. An implementer who read it
literally would search for a leading token, find none, wrap whole, and produce
`<li class="unchecked">::Tokyo [ ] is big}}</li>` — measured.

Only the description term is position-aware: it slices at its own match index
(`content[:m[0]]`, `content[m[1]:]`).

**The offset is in bytes, and on Korean content that is destructive.** The
iteration-3 audit raised this and the damage is worse than an ASCII example
suggests. Re-measured:

```
IN : "- 서{{c1::울특별시 [ ] 큼}}\n"  OUT: <li class="unchecked">{c1::울특별시 [ ] 큼}}</li>
IN : "- 서울특별시 [ ] 큼\n"          OUT: <li class="unchecked">��특별시 [ ] 큼</li>   invalid UTF-8
IN : "- [ ] {{c1::서울특별시}}\n"     OUT: <li class="unchecked">{{c1::서울특별시}}</li>   clean
```

The first row loses the marker's opening brace: the four removed bytes are the
three bytes of `서` plus one of `{`. The second is worse — the cut lands inside
a character and go-org emits invalid UTF-8. The third shows the correctly
composed form is clean, which is what makes this a composition obligation
rather than an unavoidable loss. This repository's own notes are Korean, so an
ASCII-only corpus would exercise the single case that does not arise here; M1
carries a multi-byte row accordingly.

**Two corrections to the informative list.** Both measured, and both were
over-claims of consumption rather than under-claims, so neither weakened the
obligation:

```
IN : "1. [@5] Tokyo :: big\n"   OUT: <dl><dt>[@5] Tokyo</dt><dd>big</dd></dl>
IN : "- Plain\n- Term :: Def\n" OUT: <ul><li>Plain</li><li>Term :: Def</li></ul>
```

The cookie is gated on the parser's list **kind**, not on the terminator: one
`::` makes the list descriptive and the cookie then survives in the term. And
`listKind` reads `d.tokens[i]` — the list's **first** token — so kind is
decided by the first item, not by a `::` appearing anywhere. The plain-sibling
shape is reachable whenever the first item carries the `::`, which is the
ordering the earlier probe happened to use.

**Why the rule is now definitional rather than enumerated.** The audit's
required fix was to enumerate four prefixes instead of three. That is the
weaker fix: a later go-org consuming a fifth would silently invalidate the
enumeration, and the same defect would be filed again. `spec.md` REQ-ML-001.3
instead makes the obligation definitional — the span begins where the parser
begins the item's content — prohibits deriving that offset by searching for a
token, and keeps the three-item list as **informative** and pinned to v1.9.1.
A fifth shape then needs no edit to the requirement.

**Why one rule rather than several.** Every failure is the same failure — a
marker opening that lands before the offset the parser starts content at. One
rule covers the cookie, the status, the description term, and the
position-blind case, rather than a sub-clause accreting per shape as each is
discovered.

**Why the count-equality cross-check cannot catch these.** For a checkbox list
the scanner and the parse agree on the item count; only the wrapped span
differs. The cross-check is a real falsifier for miscounting and blind to
mis-spanning, which is why these shapes are named in the M1 corpus (§ F) as
span assertions rather than left to it.

**Reversibility.** Moderate.

### DD-4 — The answer list is the first list only

**Decision.** Answers come from the first top-level list of the remaining
body; a later list is ordinary content (`spec.md` REQ-ML-001.4).

**Why.** "The answers" is one group. A body with a list, a paragraph, and a
second list has two constructs, and promoting both to answers would wrap text
the author wrote as commentary. The `children-list` container is also one
element (DD-6), which a two-list reading would have to either duplicate or
stretch across intervening prose.

**The case this decision actually decides.** Two blank lines separate one
authored list into two, measured:

```
IN:  "- A\n\n\n- B\n"
OUT: <ul><li>A</li></ul>
     <ul><li>B</li></ul>
```

That shape is reachable without the author ever typing two blank lines. A
supplementary block sitting between two answers leaves exactly it once the
split has run — measured on the real function:

```
BODY:  "- A\n\n#+BEGIN_EXTRA\nnote\n#+END_EXTRA\n\n- B\n"
REST:  "- A\n\n\n- B"
RENDERS: <ul><li>A</li></ul><ul><li>B</li></ul>
```

So `B` would **not** be an answer, and an author who interleaved a note
between two answers would get a one-answer card with no signal.

**The v0.1.0 disposition was wrong, and this replaces it.** That version
rejected two fixes and accepted the loss, mitigated by a README paragraph. The
two rejections stand and are kept below. What was missing is a third fix that
escapes both objections, which the audit supplied.

**Rejected: read the answer list before the split.** Supplementary content
would become an answer, which `spec.md` REQ-ML-009.1 forbids outright. Still
rejected.

**Rejected: normalize the residue inside `splitExtraBlocks`.** That function
is on the path of **every** cloze entry, and its byte-identical no-block return
is load-bearing. Normalizing there would change rendered output for every
existing note carrying a supplementary block, re-hashing entries that are not
even multiline — the mass-spurious-update outcome REQ-ML-014 exists to
prevent. Still rejected.

**Adopted: collapse the gap inside composition.** Composition already rewrites
Org source, and it runs **only for a multiline entry**. Collapsing a blank-line
run to a single blank line there therefore touches no other entry and re-hashes
nothing outside this SPEC's own new surface — which is precisely the objection
that sank the previous fix, and it does not reach this one. Measured, the
collapse produces exactly the wanted result:

```
IN:  "- A\n\n- B\n"     OUT: <ul><li>A</li><li>B</li></ul>
```

One list, both answers. `spec.md` REQ-ML-009.2 carries the obligation and
`acceptance.md` AC-ML-009d asserts the end-to-end shape.

**The collapse skips block interiors, and the v0.1.1 version did not.** The
iteration-2 audit found that an unqualified collapse strips blank lines from a
`#+BEGIN_SRC` or `#+BEGIN_EXAMPLE` block sitting in the question context.
Re-measured, the rendered output genuinely differs:

```
BEFORE "#+BEGIN_SRC text\nline1\n\n\nline2\n#+END_SRC\n\n- A\n"
AFTER  "#+BEGIN_SRC text\nline1\n\nline2\n#+END_SRC\n\n- A\n"
identical = false      (same for #+BEGIN_EXAMPLE)
```

That is a silent content change — an author's code sample reformatted with no
diagnostic and no report signal — which is the same failure shape D10 was
filed against, reintroduced by D10's own fix. The correction was already
written two paragraphs away and simply not carried across: DD-1 requires the
**scanner** to skip block interiors, and M1 lists the collapse and the scanner
as two separate deliverables, so the collapse inherited nothing. Both now
carry the same exclusion. `acceptance.md` AC-ML-009f asserts a block's
interior renders identically with and without the collapse.

**An unterminated block: decline, do not guess.** A depth-counting skip has no
closing marker to count, so "inside a block" is undefined. Measured, go-org
does not fail the document — it logs a parse error, treats the `#+BEGIN_` line
as plain text, and renders the rest normally, so the blank-line run survives
and the two lists stay split:

```
IN : "#+BEGIN_SRC text\n- A\n\n\n- B\n"
OUT: <p>#+BEGIN_SRC text</p><ul><li>A</li></ul><ul><li>B</li></ul>
```

`spec.md` REQ-ML-009.2 therefore has the collapse decline to act on such a run.
The cost is a benefit not delivered on malformed input, which is why the audit
classed this optional and why nothing renders worse than it does today. The
alternative — guessing the block's extent on input the parser itself could not
read — risks the silent content change the exclusion exists to prevent, which
is the worse direction to err.

**Not adopted: a non-skipping advisory.** The advisory mechanism exists and is
tested, and it was the audit's other candidate. It is unnecessary once the
collapse lands: there is no longer a loss to advise about. Reporting an
advisory for a card that now contains every answer the author wrote would be
noise, and an advisory is the weaker fix in any case — it tells the user their
content was dropped instead of not dropping it.

**What this does to DD-4's own scope.** The collapse narrows when this decision
bites. A blank-line run no longer separates one authored list into two, so
"first list only" now applies where genuine content sits between the lists — a
paragraph, a block — which is the case where a second list really is
commentary. Measured: `- A\n\nNote.\n\n- B\n` still renders two lists, and
that is correct. `spec.md` § 5 records the narrowed exclusion.

**Cost.** An author wanting two answer groups writes two headings. Recorded in
`spec.md` § 5 as an exclusion rather than left implicit.

**Reversibility.** Moderate.

### DD-5 — A span already carrying a hand-written marker is not wrapped

**This decision replaces the v0.1.0 one, which was unimplementable.** The
earlier version let a hand-written marker sit inside a wrapped answer and
required it to survive byte-for-byte. `spec.md` REQ-ML-007.1 separates every
run of two closing braces inside wrapped content, so it would have rewritten
that marker's `}}` into `} }` and destroyed it. The audit measured the
collision on the repository's own rule: `imoogi-anki--cloze-safe-text` turns
`Tokyo is {{c4::big}}` into `Tokyo is {{c4::big} }`. The two criteria could
not both hold.

**Exempting the inner marker from the brace rule does not rescue it.** This is
the load-bearing point, and it is an **Anki** constraint rather than a
rendering one. go-org passes the nested form through untouched — re-measured
here:

```
IN : "- {{c5::Tokyo is {{c4::big}}}}\n"
OUT: <ul><li>{{c5::Tokyo is {{c4::big}}}}</li></ul>
```

so nothing in the render path objects. Anki's cloze pattern is non-greedy: the
outer `{{c5::` closes at the **inner** marker's `}}`, leaving a trailing `}}`
in the card text and a deletion that ends in the wrong place. Cloze markers
cannot nest in Anki, and no placement of the brace rule changes that.

**Decision.** A span that already carries a hand-written marker is not
wrapped. It reaches the renderer byte-for-byte as the author wrote it. The
rule binds an answer item and the title alike (`spec.md` REQ-ML-006.1).

**Why not reject the entry with a diagnostic.** Rejection was the other option
the audit named. It costs the author the whole card until they edit it, in
exchange for information they already have: they typed the marker, and they
can see it in the buffer. Not wrapping costs nothing — the item is still
clozed, by the author's own marker, so it still produces a card. The multiline
options still wrap every other answer. Nothing is destroyed and nothing is
silent.

**What survives from the old decision.** The offset rule, correctly scoped:
generated numbering for the spans that *are* wrapped still begins above the
highest hand-written number anywhere in the entry, so a generated marker can
never collide with one the author wrote elsewhere. That is the same convention
`imoogi-anki--next-cloze-number` uses, so both sides of the system count alike.

**A consequence, stated.** An entry whose every answer item carries a
hand-written marker composes with nothing wrapped. It is not rejected: its
answers are present and already clozed, so the card works. `acceptance.md`
AC-ML-006d pins it.

**Reversibility.** Low cost while unreleased; high after, since a numbering
change re-hashes every multiline note.

### DD-6 — A new entry point, and a separate golden corpus

**Decision.** The options reach the renderer through a **new** entry point.
The existing `Render(noteType, title, body)` keeps its signature and its
behavior, and the multiline byte-identity cases go in their own golden file
(`spec.md` REQ-ML-010, REQ-ML-014.3).

**Why not widen the existing entry point.** SPEC-ANKICARD-002 shipped
`TestRenderSignatureIsUnchanged`, a compile-time assertion that the entry point
took no card-option parameter. Widening it makes that test fail to compile.
That test is not an obstacle to route around — it is the mechanism by which
SPEC-ANKICARD-002's byte-identity claim stays honest, and it should keep
passing for entries that have no options. A second entry point satisfies both:
old callers and the old proof are untouched, new callers pass options.

**The new entry point delegates rather than reimplements.** Both production
call sites move to it (REQ-ML-010), so once that lands the only remaining
callers of the old entry point would be its own tests — and a corpus
exercising a function no production code reaches proves nothing about
production. The audit caught this and asked for a criterion. A criterion alone
is the weaker half of the fix: the new entry point is therefore required to
call the old one for a non-multiline entry (`spec.md` REQ-ML-010.1), which
makes byte-identity structural rather than merely measured, and leaves the
criterion guarding that the delegation is actually there. `acceptance.md`
AC-ML-013e asserts it against the existing corpus and names the planner test
that catches a divergence through the real run.

**Why a separate golden file.** Adding multiline cases to `render-golden.json`
requires regenerating it, and a regenerated golden proves nothing about the
cases it already held — the run that would have caught a regression is the run
that re-blesses it. The existing file stays byte-identical, verified by
`git diff --exit-code` against `6f3ae6c` (`acceptance.md` AC-ML-013a), and the
new cases get their own file with its own recording environment variable.

**Why the migration path must pass options too.** `isClozeStyle` accepts stock
`Cloze`, so a stock-`Cloze` heading can be a valid multiline entry. If
`migrate.go` rendered it without its options, the migrated note would carry an
unwrapped Text while the ordinary path computes a wrapped one — the very next
synchronization reports an update for content the user never changed. This is
the re-examination SPEC-ANKICARD-002 § 5 assigned to the first rendering card,
and the answer is yes. `acceptance.md` AC-ML-010b is the check.

**Reversibility.** Low. It is an internal API shape with two call sites.

### DD-7 — Two editor commands, and what they do about swift

**Decision.** A direction command offering the three arrows plus an explicit
clear, and a separate incremental toggle (`spec.md` REQ-ML-013.1).

**Why two rather than one.** Direction is a three-value choice and incremental
is a boolean. A single prompt carrying both asks about incremental on every
direction edit, and the lone direction change is the common one. The repository
already contains both shapes — `imoogi-anki-set-deck` reads a value with
completion, `imoogi-anki-mark-cloze` is a per-outcome command — so neither
choice is novel; the split follows the shape of the data.

**Off writes the falsy spelling, it does not delete.** All three properties
inherit. Deleting `ANKI_INCREMENTAL` from a heading under a file-level
`#+PROPERTY: ANKI_INCREMENTAL t` re-exposes the inherited value, so the toggle
would appear not to work. Writing the falsy spelling into the heading's own
drawer is the documented opt-out and is what SPEC-ANKICARD-002 recognizes on
all three properties precisely so this is uniform.

**Swift blocks the write rather than being silently cleared.** A heading that
resolves swift truthy and then gains a direction is exactly the entry
SPEC-ANKICARD-002's conflict rule rejects — and the user would not learn that
until the next sync. Three options were considered: write anyway (defers a
known failure), write and clear swift silently (a change the user did not ask
for), or report and require confirmation. The third is chosen: declining writes
nothing at all, including the note type, so a declined command leaves no trace.

**Why `ANKI_SWIFT` stays out of property completion.** SPEC-ANKICARD-002 § 5
placed each property name's completion candidacy with the command that writes
it. This SPEC writes two of the three; t15 writes the third.

**Reversibility.** High. Elisp command shape is the cheapest thing here to
change.

## § C Pre-flight

Run before any edit, and cite the output:

```bash
git rev-parse HEAD && git branch --show-current
go build ./... && GOOS=windows GOARCH=amd64 go build ./...
go test ./... -count=1 2>&1 | tail -5
go test -cover ./internal/anki/... 2>&1 | tail -10
make lint 2>&1 | tail -5
git diff --exit-code 6f3ae6c -- internal/anki/orgdoc/testdata/render-golden.json && echo GOLDEN-CLEAN
```

The last line establishes that the byte-identity evidence is unmodified at the
start, so a later failure is attributable to this SPEC's work.

## § D Constraints

1. Every file in § A.3's PRESERVE list is untouched.
2. No file outside `spec.md` § 7's delta table is modified.
3. Composition is source-level and pre-render (DD-1). A post-render HTML scan
   is not an acceptable implementation of any requirement here.
4. No second parser for the option values. `readDirection` and `readBoolean`
   in `card_options.go` are the only ones.
5. Conventional Commits, scope `SPEC-ANKICARD-003`. Never `--no-verify`, never
   `--amend`, never a force push.
6. Stage by explicit pathspec. The working tree carries unrelated uncommitted
   and untracked changes; `git add -A` would sweep them in.
7. No user prompt from the implementing agent. A blocker is a structured
   report to the orchestrator.

## § E Self-Verification

Report each item as Claim / Evidence (verbatim command and output) /
Baseline-attribution (this run, this tree, HEAD SHA) / Gaps / Residual-risk.

| # | Item | Command |
|---|---|---|
| E1 | AC-ML-001..015 binary matrix | the command each `acceptance.md` section names |
| E2 | Cross-platform build | `go build ./...` and `GOOS=windows GOARCH=amd64 go build ./...` |
| E3 | Per-package coverage vs § A.2 | `go test -cover ./internal/anki/...` |
| E4 | Existing corpus unmodified | `git diff --exit-code 6f3ae6c -- internal/anki/orgdoc/testdata/render-golden.json` |
| E5 | Lint and format, NEW vs baseline | `make lint`, `make fmt-check` |
| E6 | Elisp suite, 0 unexpected | `make test-elisp` |
| E7 | Scanner-vs-parse item-count cross-check (DD-1) | the property test over the corpus |
| E8 | RED output captured before GREEN, per milestone | the milestone's test command, run before its implementation |

**Run-phase review note — the delegation is unasserted.** REQ-ML-010.1
requires the new entry point to call the existing one for a non-multiline
entry, and no criterion asserts the call directly: a reimplementation that
matched the corpus byte-for-byte would satisfy AC-ML-013e while violating the
requirement. The iteration-2 audit raised this and classed it consequence-free,
which is right — the observable contract is byte-identity and AC-ML-013e holds
it. It is recorded here so the run-phase reviewer checks the call site by
reading rather than expecting a test to catch it. Adding a criterion that
asserts an internal call would test the implementation rather than the
behavior, which is why one is not added.

## § F Milestones

Priority-ordered. Each lands as its own commit with its tests.

### M1 — Answer identification (Priority High)

The blank-line collapse, the scanner, the parse-backed description-kind and
item-count reads, and the cross-check property test. No wrapping yet; the
deliverable is a function that says which source spans are answers.

Requirements: REQ-ML-001, REQ-ML-009.2. Criteria: AC-ML-001.

This is first because DD-1's mechanism is the one thing that could still prove
unworkable, and everything downstream assumes it.

**The corpus is enumerated here, not left to the implementer.** The
cross-check is only as good as what it runs on, and a corpus chosen by whoever
writes the scanner tends to contain the shapes that scanner already handles.
These **sixteen rows** carry **eighteen shapes** — two rows pair a shape with
its contrast — and all eighteen are required members. Each stresses a
different axis, and each is measured against go-org v1.9.1 rather than
assumed:

| Shape | What it discriminates |
|---|---|
| `#+BEGIN_SRC text\n- x\n#+END_SRC\n- real\n` | Parser sees one item; a grammar-only scanner sees two |
| `#+BEGIN_EXAMPLE\n- x\n#+END_EXAMPLE\n` | Parser sees **no** list; a naive scanner sees one and wrongly skips `multiline_answer_missing` |
| `#+BEGIN_QUOTE\n- inner\n#+END_QUOTE\n` | A real list, but not at top level — existence versus top-level-ness |
| `- A\n\n- B\n` and `- A\n\n\n- B\n` | One blank keeps one list, two split it; after the collapse both must read as one |
| `- A\n\nNote.\n\n- B\n` | Genuine content still separates, so the collapse did not over-reach |
| `- [ ] A\n- [X] B\n` | Counts agree, spans differ — the cross-check is blind here by construction, so this row is a **span** assertion |
| `- term :: def\n- plain\n` | Counts agree, item kinds diverge — also a span assertion |
| `1) A\n2) B\n` | The `)` terminator is an ordered list too |
| `  - A\n  - B\n` | Indented first bullet is top level, not nested |
| `- A\n+ B\n` | go-org merges a bullet change into one list where Org-mode would not; pinned so the divergence stays deliberate |
| `1. [@5] A\n2. B\n` | The ordered counter cookie is consumed ahead of content — a third **span** assertion, and one the `::[` signature does not catch |
| `- Tokyo [ ] is big\n` | Position-blind consumption: no leading token exists, yet the parser removes four characters. A leading-token search wraps whole and corrupts |
| `#+BEGIN_SRC text\nx\n\n\ny\n#+END_SRC\n\n- A\n` | The collapse must not reach inside a block; rendered output identical with and without it |
| `- A\n- B\n\n\n- C\n- D\n` | An author's deliberate two-list body merges into one four-item list — the cost the collapse takes, pinned so it stays deliberate |
| `- 서울특별시 [ ] 큼\n` and `- [ ] {{c1::서울특별시}}\n` | The offset is in **bytes**: a rune-rounded composer misplaces the marker, and the parser's own cut yields invalid UTF-8. The multi-byte twin of the position-blind row, and the case an ASCII corpus cannot reach |
| `#+BEGIN_SRC text\n- A\n\n\n- B\n` | Unterminated block: the collapse declines, the lists stay split, and the document still renders |

The **four** span assertions — the checkbox row, the mixed-description row,
the counter-cookie row, and the position-blind row — are called out because
the count-equality property cannot fail on any of them: the scanner and the
parse agree on how many items there are and disagree only on what part of each
is the answer. Those four rows assert the wrapped span directly. The
position-blind row is the one that fails an implementation built by searching
for a leading token, which is the reading `spec.md` REQ-ML-001.3 now forbids
outright.

### M2 — Composition (Priority High)

Direction, default direction, incremental numbering, the hand-written offset,
and brace safety. Produces the composed Org fragment.

Requirements: REQ-ML-003..REQ-ML-008. Criteria: AC-ML-003..AC-ML-008.

### M3 — Pipeline and diagnostics (Priority High)

The new entry point, the split-compose-gate-render order, both call sites, the
new protocol code, and the planner's error mapping.

Requirements: REQ-ML-002, REQ-ML-009, REQ-ML-010, REQ-ML-011.
Criteria: AC-ML-002, AC-ML-009, AC-ML-010.

### M4 — Editor commands (Priority Medium)

The direction command, the incremental toggle, the note-type contract, the
swift confirmation, the key bindings, the transient entries, the completion
candidates, and ERT coverage.

Requirements: REQ-ML-013. Criteria: AC-ML-012.

### M5 — Presentation and pairing (Priority Medium)

The `children-list` container and its stylesheet rule; the diagnostic message
and the pairing assertion over the enlarged code set.

Requirements: REQ-ML-012, REQ-ML-015. Criteria: AC-ML-011, AC-ML-014.

### M6 — Non-interference evidence and documentation (Priority Low)

The separate multiline golden corpus, the falsy-options identity case, the hash
input-set assertion, the `git diff` check, the full-baseline comparison, and
the README Anki section.

Requirements: REQ-ML-014. Criteria: AC-ML-013, AC-ML-015.

Mechanical and last by design: it records evidence about work the earlier
milestones did, so it cannot be written before them and carries no design
decision of its own.

## § G Anti-Patterns

1. **Scanning rendered HTML for `<li>`.** DD-1 rejects it with measured
   evidence. A description list has no `<li>`, and the miss is silent.
2. **Regenerating `render-golden.json`.** It converts the byte-identity proof
   into a tautology. AC-ML-013a is the guard.
3. **Widening the existing entry point.** Breaks
   `TestRenderSignatureIsUnchanged` at compile time, which is that test working
   as designed, not an obstacle.
4. **Skipping the migration call site.** Produces a spurious update on the very
   next sync (DD-6), and the symptom appears far from the cause.
5. **A second option-value parser.** `card_options.go` owns trimming and case.
   A second copy drifts the first time one is edited.
6. **Adding a fourth rule to the validation gate.** `multiline_answer_missing`
   is a render-time condition; placing it in the gate would require composing
   before validating, inverting the order SPEC-ANKICARD-002 fixed.
7. **Deleting a property to turn an option off.** Inheritance re-exposes the
   ancestor's value (DD-7).
8. **Asserting brace safety by reading the code.** AC-ML-007a states it as an
   output property — no `}}` between the generated marker's opening and its own
   closing — which is checkable on the rendered string.
9. **Trimming the M1 corpus to the shapes the scanner already passes.** The
   sixteen rows in § F M1, carrying eighteen shapes, are required members. Dropping one because it is
   awkward removes the only falsifier for the axis it stresses, and the four
   span assertions in particular cannot be replaced by the count-equality
   property.
10. **Wrapping an item that already carries a hand-written marker.** Anki
    cannot nest cloze markers, so the outer one closes at the inner one's
    braces (DD-5). The item is left unwrapped; it is not renumbered and the
    entry is not rejected.
11. **Finding the answer-content offset by searching for a leading token.**
    Two of go-org's three consumptions are position-blind, so a `[ ] ` in the
    middle of an item still removes the item's first four bytes and no leading
    token exists to find (DD-3). Mirror the parser's own arithmetic.
12. **Rounding the offset to a rune boundary.** The parser removes bytes and
    does not round, so a rounded composer places its marker somewhere the
    parser does not start content. On Korean content this loses the marker's
    opening brace outright (DD-3).
13. **Letting the blank-line collapse reach inside a block.** It reformats an
    author's code sample silently (DD-4). The scanner and the collapse carry
    the same block-interior exclusion.

## § H Cross-References

- `.moai/specs/SPEC-ANKICARD-002/{spec,plan,acceptance}.md` — the transport and
  validation this SPEC consumes; its § 5 holds the two deferrals DD-6 and
  `spec.md` REQ-ML-004 discharge.
- `.moai/reports/anki-card-types-plan-20260920.md` § 2.2 — the confirmed
  generation rules; § 2.3 decisions B, C, and F; § 5 the multi-line-HTML risk
  DD-2 weighs.
- `internal/anki/planner/card_options.go` — the value parsers and the gate.
- `internal/anki/orgdoc/extra.go` — the split composition runs after.
- `modules/org/24-anki.el` — the edit-time contract DD-7 mirrors.
