# SPEC Review Report: SPEC-ANKICARD-004

Iteration: 2/3
Verdict: **FAIL**
Overall Score: **0.75** (harmonic mean; Tier M PASS threshold is 0.80)
STOP signal: **not raised** — iter2 0.75 equals iter1 0.75, so this is not a
score regression. See § Why the score held.

Reasoning context ignored per M1 Context Isolation. Scoped to the iteration-1
defect delta plus a regression check over those defects, as the Retry Loop
Contract requires for iteration 2+. Measurements were taken in a scratch Go
module outside the checkout
(`/private/tmp/.../scratchpad/sw5`) against `go-org v1.9.1` from the module
cache, at repository HEAD `08f7504`. No git command that writes was run; the
working tree was not modified.

---

## Verdict in one paragraph

All thirteen iteration-1 findings are closed, and closed well — every one
verified individually below, none reopened. The verdict is FAIL because the two
replaced design decisions introduced two new blocking defects of the same
structural class the iteration-1 report identified as this document's cluster:
**a predicate whose scope is narrower than the obligation stated above it.**
One of them (N1) is the fix-introduces-the-failure pattern the brief primed for:
REQ-SW-001.5 adopts a definitional rule *because* an enumeration misses what
nobody thought of, and REQ-SW-001.6 then prescribes a closed enumeration of four
shapes — which I measured misses at least two more, each producing exactly the
failure mode REQ-SW-001.5 names as its reason for existing.

---

## Must-Pass Results

- **[PASS] MP-1 REQ number consistency.** `grep -o 'REQ-SW-[0-9]\{3\}' spec.md |
  sort -u` → `REQ-SW-001 … REQ-SW-014`, contiguous, zero-padded.
  `grep -c '^#### REQ-SW-'` = 14; `sort | uniq -d` over the headings prints
  nothing. Unchanged from iteration 1.
- **[PASS] MP-2 GEARS format compliance** (requirement layer only). All 14
  headings carry exactly one pattern tag: `[Where]` ×2, `[Ubiquitous]` ×10,
  `[Ubiquitous — negated]` ×2 (REQ-SW-005, REQ-SW-012). No Given-When-Then entry
  appears in the requirement layer; the Given-When-Then entries in
  `acceptance.md` are the verification layer and are graded under Group 4.
- **[PASS] MP-3 YAML frontmatter validity.** All 12 canonical fields present
  with correct types at `spec.md`:L2-15. `version: "0.1.1"` quoted,
  `status: draft`, `created`/`updated: 2026-09-21` ISO, `priority: P2`,
  `lifecycle: spec-anchored`, `tags` a comma-separated string. No rejected
  snake_case alias. Optional `tier: M` and `depends_on` also present.
- **[N/A] MP-4 language neutrality.** Single-project SPEC scoped to this
  repository's Go packages and Emacs Lisp modules. Not template-bound content.
- **[PASS] MP-5 D7 cross-SPEC reconciliation.** The three referenced SPECs all
  resolve and all read `status: in-progress`; none is retired, superseded, or
  archived, so no reconciliation obligation fires. The SPEC-ANKICARD-003
  dependency is disclosed with its authorization at `spec.md`:L871-884.
- **[PASS] MP-6 D8 cross-platform discipline.** `grep -c 'syscall'` = 0 across
  `spec.md`, `plan.md`, `acceptance.md`. Auto-PASS per D8-4.
- **[N/A] MP-7 clarification gate.** `grep -rn '\[NEEDS CLARIFICATION'` over the
  SPEC directory returns exit 1. `research.md` does not exist — correct for
  Tier M — so the criterion is N/A on that artifact and PASS on `plan.md`.

No must-pass criterion fails. **The FAIL rests on the aggregate score falling
below the Tier M threshold, not on the firewall.**

---

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.50 | 0.50 — "Multiple requirements require interpretation. A reasonable engineer might implement them differently than intended." | Three loci. **N1**: `spec.md`:L263-267 ("shall **not** determine it by matching a list of known line shapes") against :L297-301 ("only the shapes that scanner has no reason to classify — the table row, the heading, the keyword and comment lines — are new, and they are recognized at the line level"). Both normative; I ran both readings and got different cards. **N2**: REQ-SW-005's head clause (:L371), Constraint 6/7 (:L797-800) and REQ-SW-001.7 (:L314-319) all state an obligation broader than the predicate that implements it. **N5**: REQ-SW-013.3 (:L631) says "the command" without distinguishing the toggle's ON path from its OFF path. |
| Completeness | 1.00 | 1.0 | HISTORY (:18), Purpose (:126), Scope (:138), Glossary (:148), Requirements (:184), Relationship (:662), Out of Scope (:695 — six `### Out of Scope — <topic>` H3 headings, each carrying specific `-` bullets), Constraints (:775), Brownfield Delta (:803), Dependencies (:869), Traceability (:886). Acceptance criteria in `acceptance.md` per the Tier M artifact set. Frontmatter complete. |
| Testability | 0.75 | 0.75 — "One AC is not precisely binary-testable but is measurable with minor interpretation." | No weasel word appears: `grep -nio 'appropriate\|adequate\|reasonable\|properly\|as needed\|if necessary\|sufficiently\|good enough' acceptance.md` → exit 1, no output. All 15 criteria carry a deciding command (`grep -c '^```bash'` = 15). The exception is **AC-SW-001e2** (`acceptance.md`:L78-82): "A heading line, a `#+TITLE:` keyword line, and a `#` comment line likewise yield zero" names three shapes with no corpus row, so the test author must invent the bodies; only the table shape got row 22. Compounded by **N3**: AC-SW-011's heading contradicts its own body. |
| Traceability | 1.00 | 1.0 | All 14 REQs named in `acceptance.md` (per-REQ hit counts: 001→9, 002→1, 003→1, 004→1, 005→2, 006→1, 007→2, 008→1, 009→1, 010→2, 011→3, 012→2, 013→1, 014→1). All 15 ACs name a valid REQ in their heading. No orphan, no uncovered REQ. Counts unchanged at 14/15, both within the Tier M ceiling of 16. |

Harmonic mean: `4 / (1/0.50 + 1/1.00 + 1/0.75 + 1/1.00)` = `4 / 5.333` = **0.75**.
Tier M threshold (`spec-workflow.md` § SPEC Complexity Tier) is **0.80**.

### Why the score held at 0.75 rather than rising

This is the sentence that matters most for routing. The score did not stall
because the revision was ineffective. **The three iteration-1 Clarity
divergences (D1, D2, D12) are genuinely closed and closed well** — each verified
individually in the closure table below, each pinned by a corpus row that
discriminates the specified rule from the reading it replaces. They were
replaced by two new loci, and **one of them (N1) was introduced by the DD-4
fix itself**. Iteration 1's Clarity was 0.50 for three divergences; iteration 2
is 0.50 for three different ones. Testability rose materially (17→24 corpus
rows, AC-SW-007's mechanicity claim replaced by a real test) but is held at 0.75
by a smaller residue.

No STOP signal: 0.75 is not lower than 0.75.

---

## Finding-by-finding closure table (iteration 1)

| # | Sev | Status | Evidence |
|---|-----|--------|----------|
| **D1** | critical | **CLOSED** | REQ-SW-005.1 (`spec.md`:L373-397) makes a line whose first arrow token lies inside a marker's span not an arrow line; span = Anki's non-greedy extent; unclosed opening extends to the body's end; states the presence predicate cannot decide it and an extent scan is needed. Corpus rows 18, 19, 24. **Row 24 does discriminate** as the brief asked: `{{c1::메모`⏎⏎`도쿄 :-> 일본` — line-locally the arrow line carries no marker at all and would compose, so a line-local reading admits it and the body-wide rule rejects it. Rows 18 and 19 are both single-line and cannot make that distinction; AC-SW-005 says so explicitly (`acceptance.md`:L161-167). Residual Risk 4 corrected: the third path is now named and closed (`acceptance.md`:L530-534). |
| **D2** | major | **CLOSED** | REQ-SW-001.3 (`spec.md`:L233-247) settles toward rejection and states why non-greediness does not pin the first token. Corpus row 20. Measured: the rejection catches no legitimate line — see § Mechanical confirmations. |
| **D12** | major | **CLOSED** | The mechanism is gone, not patched. REQ-SW-011.3 (`spec.md`:L534-548) forbids composition inserting any line; REQ-SW-001.5 independently excludes list continuation. Corpus row 21. AC-SW-011 requires `grep -n '#+ATTR_HTML' internal/anki/orgdoc/swift.go` to return nothing. Two independent guards, as the sub-clause says. |
| **D3** | minor | **CLOSED (dissolved)** | With no container there is no run whose end must be stated. Corpus row 23 pins prose immediately below an arrow line; REQ-SW-011.1's conditional break is what keeps it unchanged. |
| **D4** | minor | **CLOSED, with a new overclaim** — see N4 | The trio is named in all four places the finding required: REQ-SW-012.4 (`spec.md`:L582-595), DD-5 (`plan.md`:L421-437), AC-SW-013 (`acceptance.md`:L365-371), and the helper docstring obligation. The three legs are individually necessary and none is redundant. |
| **D5** | major | **CLOSED, with a residue** — see N5 | REQ-SW-013.3 (`spec.md`:L637-650) stops routing through the write helper, requires the note-type helper be usable standalone, leaves the factor-out-vs-parameter choice to the implementer, and requires the off path fire no swift-conflict prompt. DD-10 (`plan.md`:L490-522) carries the reasoning. AC-SW-014 (`acceptance.md`:L406-410) requires the **unstubbed** ERT case. **The OFF path is genuinely unblocked**: the prompt lives only in `imoogi-anki--resolve-swift-conflict`, which the new command no longer calls. **Nothing the helper also did is skipped** — I checked: `imoogi-anki--write-card-option` does `at-heading` → conflict → note-type → `org-set-property` → return t; the existing toggle already calls `(imoogi-anki--at-heading)` itself at `modules/org/24-anki.el:393`, and the property write is separately required by REQ-SW-013.1/.2. The note-type message correction is safe: no Elisp test asserts the string `멀티라인 옵션` (`grep -rn '멀티라인 옵션' tests/` → one comment line only, `tests/anki-commands-test.el:273`). |
| **D9** | minor | **CLOSED** | REQ-SW-013.4 (`spec.md`:L651-660): "**any** multiline option", "clear **every** multiline option that is on", "the report shall name each of them". AC-SW-014 pins the both-on case with the decline and the confirm branches. |
| **D6** | minor | **CLOSED** | `acceptance.md`:L16 reads "Twenty-four rows"; `awk '/^\| # \| Body/,/^## AC Matrix/' acceptance.md \| grep -c '^\| [0-9]'` = 24. |
| **D7** | minor | **CLOSED** | `plan.md`:L44 now points at "`acceptance.md`'s **Quality Gate Criteria** table". |
| **D8** | minor | **CLOSED** | Constraint 6 (`spec.md`:L793-797): "The bound is on composition and not on the card: an export snippet an author writes in a body already reaches the card as raw HTML, and did so before this SPEC." |
| **D10** | minor | **CLOSED** | REQ-SW-001.4 (`spec.md`:L248-258) requires a Unicode-aware trim and carries the non-breaking-space measurement. Pinned in the Edge Cases table. |
| **D11** | minor | **CLOSED** | AC-SW-007 (`acceptance.md`:L215-224) replaces the grep with `TestSwiftSnippetBound` and explicitly withdraws the earlier claim: "the grep below is a convenience for a human reader, not the deciding step". |
| **§5 wording** | minor | **CLOSED** | `spec.md`:L714-721: "Nothing is dropped, and the wording matters because a future reader should not set out to repair a loss that is not occurring." |

**Regression check: no iteration-1 defect reopened.** Each row above was
verified against the current text, not carried forward.

---

## Defects Found (structured defect-list)

### N1 — REQ-SW-001.6 reinstates the enumeration REQ-SW-001.5 exists to replace, and it is incomplete

`spec.md`:L263-267 with L297-301, and `plan.md`:L313-317 — Severity: **major** — Class: **blocking**

This is the fix-introduces-the-failure pattern, in the specific shape the brief
named. REQ-SW-001.5 states the obligation definitionally and forbids the
alternative in normative language:

> "the composer shall derive the classification from the parser's own rules for
> the vendored parser version, and shall **not** determine it by matching a list
> of known line shapes."

REQ-SW-001.6 then prescribes exactly that list:

> "only the shapes that scanner has no reason to classify — the table row, the
> heading, the keyword and comment lines — are new, and they are recognized at
> the line level against the parser's own rules for them."

`plan.md` DD-4 repeats it: "only the table, heading, keyword, and comment shapes
are new, and each is a single line-level rule." Both sub-clauses are `shall`.
An implementer taking .5 builds a general classifier; one taking .6 builds four
rules. They produce different cards.

**The closed four is measurably incomplete.** Two further shapes match the arrow
expression, are not paragraph text, and are not in the list:

```
: A :-> B                     -> <pre class="example">\nA :-&gt; B\n</pre>
[fn:1] A :-> B                -> ""            (renders to nothing at all)
```

Composed under the remaining rules, each produces the exact failure mode
REQ-SW-001.5 cites as its own reason for existing:

```
COMPOSED  ": A @@html:<b class=\"swift-arrow\">:-&gt;</b>@@ {{c1::B}}"
RENDERED  "<pre class=\"example\">\nA @@html:&lt;b class=&#34;swift-arrow&#34;&gt;:-&amp;gt;&lt;/b&gt;@@ {{c1::B}}\n</pre>\n"
```

— the marker's braces *and* the raw snippet text shown on the card as literal
characters, which is the block-interior bullet's stated hazard reached by a line
that is not in a block. And the footnote definition renders to nothing at all,
which is the comment line's hazard — the one the SPEC itself calls "the
quietest" and "the sharpest" — reached by a shape the list does not name. A
`\begin{align}` line is a third (go-org v1.9.1 emits a parse warning and falls
back to plain text at the top level, but the environment's interior is raw
passthrough).

**The definitional rule is implementable, so the fix is cheap.** I measured one
route that works. Parsing a single line and reading its node type classifies
every shape correctly, including the two the list misses:

```
"A :-> B"          -> org.Paragraph          "| A :-> B |"      -> org.Table
"* A :-> B"        -> org.Headline           "#+TITLE: A :-> B" -> org.Keyword
"# A :-> B"        -> org.Comment            ": A :-> B"        -> org.Example
"[fn:1] A :-> B"   -> org.FootnoteDefinition "- A :-> B"        -> org.List
```

It is context-blind, which is exactly why REQ-SW-001.6 is right to keep the
existing scanner for the three context-dependent classes — measured:

```
"  도쿄 :-> 일본"  isolated -> org.Paragraph ;  after "- 수도" -> org.List
"x :-> y"          isolated -> org.Paragraph ;  inside #+BEGIN_SRC -> org.Block
```

So the composed method — the scanner for block pairing, bullets, and list
continuation; a parser-derived classification for the context-free remainder —
is coherent *and* complete. I record the per-line parse as one route I measured
to work, **not** as the mandated implementation; go-org's public AST carries no
source positions (`org.Paragraph` is `struct{ Children []Node }`, `Document.tokens`
is unexported, `Node` is `interface{ String() string }`), so a position-mapped
route is not available and the SPEC should not imply one.

**Required fix:** in REQ-SW-001.6, strike "only" and the closed four; require
the context-free classification to be derived from the parser's node
classification rather than from named shapes. Extend REQ-SW-001.5's
*informative* list with the fixed-width line and the footnote definition. Mirror
both in `plan.md` DD-4. Add corpus rows for `: A :-> B` and `[fn:1] A :-> B`
with `0` arrow lines.

### N2 — The definitional rule is block-level; the corruption class it must prevent is not

`spec.md`:L259-301 with L371 (REQ-SW-005 head), L314-319 (REQ-SW-001.7), L799-800 (Constraint 7) — Severity: **major** — Class: **blocking**

This is the failure direction a positive rule has and an exclusion list does
not, which the brief asked me to hunt. REQ-SW-001.5 classifies a **line**. The
damage composition does is to **inline** constructs, which live inside a line
the parser reads as ordinary paragraph text. The rule therefore admits them.

REQ-SW-005.1 closes exactly one member of this class — the hand-written cloze
marker — with a bespoke extent scan. Five others are open. All measured:

```
A1 link       "[[https://example.com][도쿄 :-> 일본]]"
   block-lvl  <p><a href="…">도쿄 :-&gt; 일본</a></p>        <- paragraph text
   rendered   <p><a href="…">도쿄 <b …>:-&gt;</b> {{c1::일본</a>}}</p>
              the marker straddles </a>; the deletion swallows the close tag

A2 code span  "~도쿄 :-> 일본~"
   rendered   <p><code>도쿄 @@html:&lt;b class=&#34;swift-arrow&#34;…@@ {{c1::일본</code>}}</p>
              the export snippet is shown on the card as literal characters

A3 verbatim   "=도쿄 :-> 일본="
   rendered   <p><code class="verbatim">도쿄 @@html:&lt;b class</code>&#34;swift-arrow&#34;…
              the `=` inside class="…" terminates the verbatim span early

A4 bold       "*도쿄 :-> 일본* 메모"
   rendered   <p><strong>도쿄 <b …>:-&gt;</b> {{c1::일본</strong> 메모}}</p>

A5 author's own snippet  "@@html:<i>도쿄 :-> 일본</i>@@"
   rendered   <p><i>도쿄html:&lt;b class=…      <- nested @@ closes early, text lost

A6 inline footnote       "[fn::도쿄 :-> 일본]"
   rendered   …<sup …>1</sup>}}</p> … <p>도쿄 <b …>:-&gt;</b> {{c1::일본</p>
              the }} lands BEFORE the {{c1:: in the field string
```

A2 is the sharpest for severity: it is byte-for-byte the failure REQ-SW-001.5's
block-interior bullet exists to prevent ("a marker's braces show on the card as
literal text"), reached through a line the same sub-clause admits. A6 is the
sharpest in kind: the field contains an opening with no following close, so Anki
finds no deletion and **refuses the note** — the failure the SPEC itself flags
as the reason the comment line must be excluded.

The SPEC states the obligation these violate in three places. REQ-SW-005's head
clause: "Composition shall not disturb a hand-written marker already present in
the title or the remaining body" — A1 and A4 disturb an authored construct of a
different kind, and the head clause's generalisation is what D1 established
should govern. REQ-SW-001.7: the rule "binds the content's text, its document
order, and **the element it renders inside**" — A1 and A4 move content across an
element boundary. And **Constraint 7's rationale is too strong**: "Every
construct this SPEC writes is inline, so no line's enclosing element can
change." Literally true and beside the point — an inline insertion cannot change
a *line's* element, but it can split an *inline* element and straddle its
boundary, which is what every row above shows.

**Required fix — the proportionate one, not a new scanner.** Do what § 5 already
does for list content: add an `### Out of Scope — Arrows inside an inline
construct` entry naming the class, the measured cost, and where it belongs. A
general inline-extent scan across link, emphasis, code, verbatim and snippet
syntaxes is a second grammar in one package and is not warranted here. Correct
Constraint 7's rationale in the same pass so it stops claiming a property the
measurements contradict. If any single member is judged worth closing now, A2/A3
(code and verbatim spans) are the cheapest, because their delimiters are
single characters a line-local scan can pair.

### N3 — The container abandonment was not propagated; five stale sites, one of them a direct contradiction

`spec.md`:L142, L675, L879; `acceptance.md`:L298, L526 — Severity: **minor** — Class: **blocking**

REQ-SW-011.3 abandons the container attribute line and `plan.md` § A.1 marks the
mechanism **NOT reused**. Five statements still describe the abandoned design:

1. `spec.md`:L142 (§ 1.2 Scope) — in scope is "the arrow-line **container class**
   and its stylesheet rule". There is no container class; the class rides the
   emphasis element.
2. `spec.md`:L675 (§ 4 "**Consumed unchanged**") — lists "the
   container-attribute mechanism REQ-ML-012 proved attaches to a rendered
   element" among surfaces consumed unchanged, and closes "This SPEC defines no
   second copy of any of them." **This directly contradicts REQ-SW-011.3 and
   `plan.md` § A.1.** It is the one site that a reader could act on wrongly.
3. `spec.md`:L879 (§ 8 Dependencies) — again lists "the container mechanism"
   among surfaces "this SPEC consumes".
4. `acceptance.md`:L298 — AC-SW-011's heading reads "The arrow lines render
   **inside the styled container**", while its own body requires a `<br>`, no
   container, and `grep -n '#+ATTR_HTML' … ` returning nothing. A test author
   reads the heading first.
5. `acceptance.md`:L526 (Residual Risk 4) — "its `Text` field rewritten with
   markers, emphasis, and **the container**".

Classified minor rather than major because HISTORY announces the abandonment
explicitly, so a careful reader resolves the conflict consistently; none of the
five produces a divergent implementation on its own.

**Required fix:** correct all five. § 4's entry should move from "Consumed
unchanged" to a named non-reuse with its reason, matching `plan.md` § A.1.

### N4 — The trio's sufficiency claim is stronger than the pair's was, and is also false

`spec.md`:L588-595 with `plan.md`:L433-437 — Severity: **minor** — Class: **blocking**

Iteration 1's D4 was an overclaim of sufficiency. The fix added the third leg
and replaced the claim with a stronger one:

> "the three together exclude **every way** an option could reach the hash"

Counterexample, by reasoning over what each leg actually constrains. An
implementation that folds the **resolved boolean into one of the four existing
arguments** — appending it to the sorted tags, or to the resolved deck string —
passes all three legs:

- Leg 1 (falsy vs absent): both resolve off, so the appended value is identical.
  Same hash. **Passes.**
- Leg 2 (on vs absent): the hashes are supposed to differ, and they do.
  **Passes.**
- Leg 3 (four-argument compile assertion): the argument *count* is unchanged.
  **Passes.**

REQ-SW-012.2 says "The content hash's input set shall gain no member" — the tag
set gained a member; the parameter list did not. Low reachability (the hashing
package is on the PRESERVE list and marked `[UNCHANGED]`), which is why this is
minor rather than major. But the claim as written is the same defect class D4
raised, one level up.

**Required fix — one sentence.** Narrow "every way an option could reach the
hash" to the three evasions the legs actually exclude: hashing the wire field,
ignoring the option entirely, and introducing a new parameter. Mirror in
`plan.md` DD-5 and in the helper docstring the sub-clause requires.

### N5 — REQ-SW-013.3's note-type contract does not distinguish the toggle's OFF path

`spec.md`:L631-636 — Severity: **minor** — Class: **blocking**

D5's fix unblocked the swift-conflict prompt on the OFF path. A second gate on
the same path is unaddressed, because REQ-SW-013.3 states the note-type contract
for "the command" without distinguishing directions, while REQ-SW-013.1 makes
the command a toggle:

> "where it carries any other type the command shall report it and change
> nothing."

Read as binding both directions, a heading carrying `ANKI_NOTE_TYPE:
imoogi-Basic` and an inherited `ANKI_SWIFT: t` cannot have swift turned off by
the command — on precisely the heading SPEC-ANKICARD-002's gate is rejecting,
whose diagnostic tells the user to "remove the option"
(`modules/org/anki/imoogi-error.el:70`). The first branch has the symmetric
oddity: turning an option **off** writes `imoogi-Cloze` onto a heading that
carried no note type.

Severity minor because the shape is **pre-existing and symmetric** — I checked,
and `imoogi-anki-toggle-incremental`'s OFF path runs the same
`imoogi-anki--write-card-option` → `imoogi-anki--ensure-cloze-type` route
(`modules/org/24-anki.el:397`, :350). So this is not a regression this SPEC
introduces; it is an ambiguity the SPEC inherits and does not resolve, on the
exact path D5 was raised about.

**Required fix:** one clause saying whether the note-type contract binds the OFF
direction, and if it does not, what the OFF path does instead. Add the
`imoogi-Basic` + inherited-on case to AC-SW-014.

### N6 — Three of the five non-paragraph shapes have no corpus row

`acceptance.md`:L78-82 — Severity: **minor** — Class: **optional**

AC-SW-001e2 names four shapes and gives only the table a corpus row (row 22).
The heading, the `#+TITLE:` keyword line, and the `#` comment line are asserted
in prose: "A heading line, a `#+TITLE:` keyword line, and a `#` comment line
likewise yield zero; the comment case matters most". The criterion is binary and
testable, but the deciding command's input set is not pinned, so the test author
invents the bodies — and the SPEC's own judgment is that the comment case is the
one that matters most. Row 22's assertion is itself correct: I measured
`| A :-> B |` → `<table><tbody><tr><td>A :-&gt; B</td></tr></tbody></table>`,
and the right side is `B |`, so wrapping would swallow the delimiter as the
note says.

**Required fix:** add three corpus rows. Together with N1's two, that is five,
taking the corpus to 29 and covering every shape the SPEC names.

---

## Judgment on the two replaced design decisions (requested)

### DD-3 — the inline break and the class on the emphasis element

**Disposition: ACCEPTED. Better than what it replaces, on its own merits, and I
found no new hazard in the mechanism itself.**

Judged on its own hazards as the brief required, not on the container being
worse:

**Does the inline break survive every list and block context the corpus covers?**
Answer by construction rather than enumeration, which is the stronger argument:
the break fires only when the successor line is itself an arrow line, and
REQ-SW-001.5 excludes list content and block interiors from being arrow lines at
all. So the break can only ever appear inside an ordinary paragraph. I measured
the list-item case anyway and it is intact —
`<ul><li>수도\n도쿄 <b class="swift-arrow">:-&gt;</b> {{c1::일본}}<br></li></ul>` —
but that is defence-in-depth confirming REQ-SW-011.3's second guard, not a
reachable case.

**Does the class reach a stylesheet selector?** Yes. The precedent's form is
`.card .children-list` (`internal/anki/model/assets/base.css`:114), a descendant
selector under `.card`, which Anki supplies. A `.card .swift-arrow` rule matches
`<b class="swift-arrow">` inside the rendered `Text` field. Measured, the class
survives the snippet intact:
`<p>도쿄 <b class="swift-arrow">:-&gt;</b> {{c1::일본}}</p>`. REQ-SW-011.2 does
not prescribe the rule's declarations, which is correct — presentation is the
implementer's, and AC-SW-011 keeps the two properties that are actually asserted
mechanically (no network resource, nothing derived from a deck name).

**Is "only when the next line is itself an arrow line" precise enough that two
implementations agree on the last line of a run?** Yes, and the three cases are
exhaustive and all resolve to *no break*: (a) the arrow line is the body's last
line — there is no following line; (b) the following line is blank — a blank
line is not an arrow line; (c) the following line is a non-arrow line — not an
arrow line. No "run" concept is needed, which is why D3's question dissolved
rather than being answered. The line sequence is stable because REQ-SW-008.1
fixes the input as the remaining body and REQ-SW-008.2 declines the blank-run
collapse, so no line is inserted or removed between detection and composition.
The SPEC would read more precisely if it said so — one sentence naming the three
cases — but I do not require it: two implementations agree as written.

The two properties DD-3 claims it gains are real. The card is correct with no
stylesheet at all (measured: the `<br>` is in the field), and Residual Risk 2
shrank accordingly.

### DD-4 — the positive definitional rule

**Disposition: the rule is RIGHT; its implementation clause is WRONG. See N1.**

The reframing is correct and its justification is sound. It mirrors
SPEC-ANKICARD-003 REQ-ML-001.3 for the same reason, and the brief's four
confirmed shapes all reproduced here exactly. The informative-list-plus-
definitional-rule form is the right shape for a version-pinned fact.

**Is it implementable?** Yes — an implementer can decide "is this paragraph
text" without guessing, by the composed method measured under N1: the existing
scanner for the three context-dependent classes, a parser-derived node
classification for the context-free remainder. What an implementer *cannot* do
is map a source line to a node through go-org's public AST, because no node
carries a position; the SPEC does not ask for that, and should not start.

**Does it cover all five shapes the brief reproduced?** Yes, as a rule. The
defect is that REQ-SW-001.6 then narrows the implementation to four named
shapes, and **I found a sixth and a seventh** (N1): the fixed-width line and the
footnote definition. Both are outside the four, both are non-paragraph, and both
produce a failure the SPEC names as its reason for the rule. So the answer to
"do not treat my confirmation as closing the question of whether more shapes
exist" is: more exist, the definitional rule already covers them, and the
implementation clause does not.

**The failure direction a positive rule has** is N2, and it is the more
consequential of the two. An exclusion list is narrow-by-default and fails by
omission; a positive rule is broad-by-default and fails by admitting something
it should not. REQ-SW-001.5 admits every line the parser calls paragraph text,
including six inline constructs whose delimiters composition splits.

---

## Mechanical confirmations (measured, not reasoned)

Recorded separately from the reasoning above, per the brief.

1. **Row 24 discriminates body-wide from line-local.** Confirmed by reading
   `acceptance.md`:L161-167 against REQ-SW-005.1. Line-locally the arrow line in
   row 24 carries no marker and would compose; the body-wide span rejects it.
2. **An unclosed opening is counted by the numbering base.** `highestClozeNumber`
   uses `clozeNumberPattern = regexp.MustCompile(`\{\{c(\d+)::`)`
   (`internal/anki/orgdoc/multiline.go`:425, :458-467) — it matches the opening
   and requires no `}}`. Measured: `"{{c5::메모"` → hit `5`. So REQ-SW-005.3's
   base is correct in the presence of the unclosed opening REQ-SW-005.1 newly
   made significant. **A candidate finding I raised and then retracted.**
3. **Arrow lines ABOVE an unclosed opening are safe** (the brief's question).
   Measured:
   `"도쿄 @@html:…@@ {{c6::일본}}\n{{c5::메모\n"` →
   `<p>도쿄 <b class="swift-arrow">:-&gt;</b> {{c6::일본}}\n{{c5::메모</p>`.
   The generated marker closes itself before the author's dangling opening, and
   every line below the opening is excluded by REQ-SW-005.1, so no generated
   `}}` ever follows it. Also checked the same-line variant
   `도쿄 :<- {{c5::일본` (arrow at offset 7, opening at 11, arrow not inside the
   span → an arrow line; the right side carries a marker so REQ-SW-005.2 leaves
   it unwrapped). No corruption on any route.
4. **The D2 rejection catches no legitimate line.** Eleven inputs measured;
   only the two whose first token sits at offset zero are rejected
   (`:-> B :-> C`, `:<- B :-> C`). `키워드 :<- 는 왼쪽, :-> 는 오른쪽` correctly
   yields `L="키워드"`, the **first** token. `C++ :-> D`, `A:->B`, and
   `기호 :-> 를 쓰면 :-> 가 화살표다` all compose correctly. One variant worth
   recording: `" :-> B :-> C"` (leading space) reaches zero arrow lines through
   REQ-SW-001.4's trim rather than through .3's rejection — different clause,
   same correct answer.
5. **`imoogi-anki--at-heading` is not lost by DD-10's replacement.** The existing
   toggle calls it directly (`modules/org/24-anki.el`:393) before reaching the
   write helper, so a swift toggle mirroring it keeps the guard. **A second
   candidate finding raised and retracted.**
6. **No Elisp test asserts the note-type failure message**, so generalizing it is
   safe. `grep -rn '멀티라인 옵션' tests/ modules/` → one comment in
   `tests/anki-commands-test.el`:273 and the message itself.
7. **Corpus count**, **weasel words**, **Out of Scope H3 headings**,
   **traceability**, **frontmatter**, **REQ numbering**, **deciding-command
   count** — all reported under Must-Pass and Category Scores above, each with
   its command.

---

## Could not verify

1. **N2's Anki-side consequences.** The rendered fields are measured; what Anki
   does with them is reasoned from its non-greedy cloze regex and from the
   SPEC's own statement that a note with no deletion is refused. Not observed
   against a running collection — AnkiConnect is not reachable here.
2. **N4.** Reasoning over what each leg constrains, not a measurement. No
   implementation exists to test against.
3. **`make test-elisp`. Declined, as instructed and as at iteration 1.** The
   suite loads modules belonging to another actor: `tests/org-roam-test.el` is
   untracked and carries 3 tests, and `tests/module-layout-test.el` and
   `tests/workspace-bridge-test.el` are modified by that actor. A failure there
   could have nothing to do with this SPEC, so the 435-test / 0-unexpected
   baseline is carried forward as the delegating session's measurement, not
   mine.
4. **`make lint`, `make fmt-check`, `make ci-local`, `go test ./... -count=1`.**
   Not run this iteration. The § A.2 baseline is carried forward unverified.
5. **The `swift-arrow` rule's rendered effect inside Anki's webview.** The
   selector form is verified against the `.card .children-list` precedent; the
   rule does not exist yet and its declarations are unspecified by design.
6. **Whether an eighth non-paragraph shape exists.** I found two beyond the
   brief's five and stopped; the search was not exhaustive, which is itself the
   argument for N1's required fix — a definitional classifier does not need the
   search to be exhaustive, and a four-item list does.

---

## Recommendation

Verdict is FAIL at 0.75 against the Tier M threshold of 0.80. Five findings are
blocking; one is optional. Fix in this order — the first two change what gets
built.

1. **N1** (`spec.md` REQ-SW-001.6, `plan.md` DD-4) — strike the closed four,
   require the context-free classification to derive from the parser's node
   classification, extend the informative list with the fixed-width line and the
   footnote definition, add their two corpus rows. This is the finding the
   DD-4 fix introduced, and it is the one an implementer will act on wrongly.
2. **N2** (`spec.md` § 5, Constraint 7) — add the inline-construct exclusion as
   an Out of Scope entry with its measured cost, mirroring the list-content
   disposition, and correct Constraint 7's rationale. Do **not** add an
   inline-extent-scan requirement; the disclosure is the proportionate move.
3. **N3** (`spec.md` § 1.2, § 4, § 8; `acceptance.md` AC-SW-011 heading,
   Residual Risk 4) — propagate the container abandonment to all five sites.
   § 4's entry moves from "Consumed unchanged" to a named non-reuse.
4. **N5** (`spec.md` REQ-SW-013.3) — say whether the note-type contract binds
   the OFF direction; add the `imoogi-Basic` + inherited-on case to AC-SW-014.
5. **N4** (`spec.md` REQ-SW-012.4, `plan.md` DD-5) — narrow "every way" to the
   three evasions the legs exclude.

Optional, at the orchestrator's discretion: **N6** (three corpus rows for the
heading, keyword, and comment shapes).

**What is strong, and should not be disturbed by the revision.** Every
iteration-1 finding is closed, and the three that mattered are closed by
removing a mechanism rather than patching it — which is why D3 dissolved instead
of needing an answer. The corpus grew from 17 rows to 24 and each new row pins a
measured defect against the reading it replaces, with row 24 doing discriminating
work that rows 18 and 19 structurally cannot. The § 5 refusal of the third option
is argued on three stated reasons rather than on effort, and the strongest of
them — that it would make this card's behaviour depend on a value computed by
another SPEC's rule — is the right reason. DD-3's replacement is better than what
it replaced on its own merits and gains two properties worth the change. The
document's remaining defects are all one shape: a predicate narrower than the
obligation above it. That is the same cluster iteration 1 named, and it is worth
saying to the author that the shape, not the instances, is what to look for.
