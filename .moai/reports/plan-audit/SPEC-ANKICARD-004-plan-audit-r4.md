# SPEC Review Report: SPEC-ANKICARD-004

Iteration: 4 (operator-authorized exception to the 3-iteration cap; decision
board ruling `d-20261012T131704Z-eab1`, scope `card:t15`). Delta-only. No
further iteration exists.
Verdict: **PASS**
Overall Score: **0.92** (harmonic mean; Tier M PASS threshold is ≥ 0.80)
STOP signal: **not raised.** 0.92 > 0.75 (iteration 3).

Reasoning context ignored per M1 Context Isolation. The audit covers three
things: closure of R3-1 to R3-4 and O-2 at every site r3 named; a regression
check over N1-N6 and the iteration-1 findings; and new defects in the v0.1.4
delta. Measurements were taken in a scratch Go module outside the checkout
(`/private/tmp/.../scratchpad/r4`), against `go-org v1.9.1` from the module
cache (`/Users/jay/go/pkg/mod/github.com/niklasfasching/go-org@v1.9.1/org/`),
using the renderer's own call shape (`org.New().Parse(…, "./").Write(org.NewHTMLWriter())`,
`orgdoc.go`:L136). Worktree HEAD is `7d98ec0`, unchanged from iteration 3, and
the SPEC files under audit are the uncommitted v0.1.4 revision. No writing git
command was run.

---

## Verdict in one paragraph

All four iteration-3 blocking findings are closed at every site r3 named, and
the optional O-2 is taken. Each was checked against the current text, and
every measurable claim was measured. Corpus rows 39-42 reproduce exactly. All
nine source-block-result edges the SPEC states reproduce. The footnote,
property-drawer, and LaTeX rules now match go-org's code
(`footnote.go`:L13 and L22-32, `drawer.go`:L62-82, `block.go`:L32-33). The
author's deliberate departure from r3's R3-2 wording is **correct on
measurement**: a list or an ordinary drawer holding a blank pair ends the
footnote definition. go-org's code agrees, because `parseListItem` and
`parseDrawer` honour the parent stop and `parseBlock` and `parseLatexBlock` do
not. r3's required-fix wording was wrong on that point. No N1-N6 or
iteration-1 finding has reopened. The delta brings **one new instance of the
recurring class** (R4-1). `spec.md` states the footnote rule correctly: none of
the three end conditions is tested inside a nested block or environment. But
`plan.md`'s mirror mechanism skips only blank pairs, and I measured the silent
case that difference produces. It is a one-sentence fix in a plan sentence
that has a correct twin, and it is reachable only in a contrived shape. It
puts Clarity at 0.75, and the aggregate is still 0.92.

---

## Must-Pass Results

- **[PASS] MP-1 REQ number consistency.** `grep -c '^#### REQ-SW-' spec.md` →
  `14`. The headings run `REQ-SW-001` … `REQ-SW-014` (`spec.md`:L446-1135),
  contiguous and zero-padded. `grep -o '^### AC-SW-0[0-9][0-9]' acceptance.md | sort | uniq -d`
  → no output, and there are 15 AC headings.
- **[PASS] MP-2 GEARS format compliance** (judged on the **requirement layer**
  only). The v0.1.4 delta changes no requirement heading or trigger. There are
  14 headings, each with one tag: `[Where]` ×2 (L446, L732),
  `[Ubiquitous — negated]` ×2 (L774, L985), and `[Ubiquitous]` ×10. The new
  REQ-SW-013.4 text keeps the sub-clause case-split form that § 3 permits
  (L427-430: "Numbered sub-clauses are case-splits on that trigger's operand").
  Its "Where the command is turning the option on" / "off" pair splits the
  trigger's operand and adds no second trigger. The Given-When-Then entries in
  `acceptance.md` are the verification layer and are graded under Group 4.
- **[PASS] MP-3 YAML frontmatter validity.** `spec.md`:L1-15 has all 12
  canonical fields: `id`, `title`, `version: "0.1.4"` (quoted),
  `status: draft`, `created: 2026-09-21`, `updated: 2026-10-12`, `author`,
  `priority: P2`, `phase`, `module`, `lifecycle: spec-anchored`, and `tags`
  (comma-separated string). No rejected alias is present. `tier: M` is present.
- **[N/A] MP-4 language neutrality.** The SPEC is scoped to this repository's Go
  and Emacs Lisp modules, not to template-bound content.
- **[PASS] MP-5 D7 cross-SPEC reconciliation.** It references
  SPEC-ANKICARD-001, -002, and -003 (`status: in-progress`) and
  SPEC-ANKICARD-005 (`status: completed`). None is retired, superseded, or
  archived, so the D7 verb emits no BLOCKING line.
- **[PASS] MP-6 D8 cross-platform discipline.** `grep -c syscall` → 0 in each of
  `spec.md`, `plan.md`, and `acceptance.md`. Auto-PASS per D8-4.
- **[PASS / N/A] MP-7 clarification gate.** `grep -rn 'NEEDS CLARIFICATION'`
  over the SPEC directory → no output (exit 1). That is a PASS on `plan.md`.
  `research.md` is absent, which is correct for Tier M, so the criterion is N/A
  on that artifact.

---

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.75 | 0.75 — "Minor ambiguity in one or two requirements that a reasonable engineer would resolve consistently." | R3-1 to R3-3 are closed (see the closure table). One locus remains, **R4-1**. `spec.md`:L575-581 states the footnote end condition (second blank line, headline, or footnote-definition line) as "tested only **between** the constructs the definition holds, never inside a `#+BEGIN_` block or a LaTeX environment". `plan.md`:L439-444 restates that correctly. `plan.md`:L446-447 then gives the mirror mechanism as "skipping blank pairs that fall inside them", which is narrower. The normative text is right, so an engineer reading `spec.md` resolves it correctly. That makes this the 0.75 band, not 0.50. Following r3's precedent, a plan-side rule narrowing counts against Clarity. |
| Completeness | 1.00 | 1.0 | HISTORY (L18, with v0.1.4 at L20-83), Overview (L344), Scope (L358), Glossary (L379), Requirements (L419), Relationship (L1149), and Out of Scope (L1188: seven `### Out of Scope — <topic>` H3 headings with `-` bullets, matching § 1.2's "seven exclusions" at L369). Also Constraints (L1349), Brownfield Delta (L1380, whose L1394-1398 names the fourth construct), Dependencies (L1465), and Traceability (L1498). Frontmatter is complete. |
| Testability | 1.00 | 1.0 | **AC-SW-008's counterfactual is gone.** `acceptance.md`:L298-301 now reads "the checked split called on that body directly returns a nil error, an `extra` output that carries the arrow line, and a `rest` output that carries none". That is directly observable against the real signature `splitExtraBlocksChecked(body string) (rest, extra string, err error)` (`internal/anki/orgdoc/extra.go`:L92). No weasel word: the `grep -nio 'appropriate\|adequate\|reasonable\|properly\|as needed\|if necessary\|sufficiently\|good enough' acceptance.md` exits 1. Every `go test -run` pattern is anchored (`grep -n "go test -run" acceptance.md \| grep -v "'^"` exits 1). The new AC-SW-014 OFF case (L497-505) is binary: drawer contents plus "no prompt fires", unstubbed. AC-SW-013's docstring clause and AC-SW-015's message clause are decided by reading, which the 1.0 band allows (as r3 recorded). |
| Traceability | 1.00 | 1.0 | Every REQ is named in `acceptance.md`. Per-REQ hit counts: 001→12, 002→1, 003→1, 004→1, 005→2, 006→1, 007→2, 008→1, 009→1, 010→2, 011→3, 012→2, 013→3, 014→1. All 15 AC headings name an existing REQ. The new rows 39-42 are cited by AC-SW-001e3 (L103, L115) and AC-SW-002 (L157-158), and by the Edge Cases table (L588-591). |

Harmonic mean: `4 / (1/0.75 + 1/1.00 + 1/1.00 + 1/1.00)` = `4 / 4.333` = **0.92**.

**Sensitivity.** With R4-1 fixed, Clarity rises to 1.0 and the aggregate to
1.0. If Clarity were scored a band lower (0.50) on R4-1, the aggregate would be
`4/(2+3)` = 0.80, which still meets the threshold. The PASS does not depend on
how R4-1 is banded.

---

## Closure table — iteration-3 findings

| # | Sev / class (iter 3) | Status | Evidence (current text, measured where measurable) |
|---|---|---|---|
| **R3-1** | major / blocking | **CLOSED** | **"Are the three" is gone.** REQ-SW-001.6 (`spec.md`:L670-675) now reads "As read for this revision, the constructs it reaches … beyond the reused three **include** the four sub-clause 5 measures … That list is informative". **The fourth construct is mirrored.** The source-block-result bullet (`spec.md`:L609-633) states the parser's and the writer's rules. The derivation is softened (`spec.md`:L694-705: "The reading is not a proof that the list is complete"), and the third failed draft is recorded (L713-715). **DD-4 and M1 are corrected.** `plan.md` DD-4 says "include" (L392) and records "exactly" as history (L401). It corrects the `#+RESULTS:` sentence (L412-415) and adds the rule (L476-494). M1 now says "these include … a source block's attached result … this list is informative" (`plan.md`:L838-842). The Brownfield Delta agrees (`spec.md`:L1394-1398). **Rows 39 and 40 reproduce.** Measured: row 39 → `…<pre>\necho\n</pre>…</div>\n` (the arrow line is gone, so the count is 0), and row 40 → `…</div>\n<p>도쿄 :-&gt; 일본</p>\n` (count 1). **All nine stated edges reproduce** (L622-633, `plan.md`:L485-494). `:exports none` gives `""`. One blank line after `#+RESULTS:` still suppresses, and two do not (`<p>\n도쿄 :-&gt; 일본</p>`). A `:RESULTS:` drawer is suppressed whole while the line after its `:END:` renders. A non-blank line before `#+RESULTS:` detaches the result, and an `EXAMPLE` block in place of the source block does too. `:exports CODE`, `:EXPORTS code`, no language token, `results`, `both`, and absent `:exports` each render in place. Lowercase `#+begin_src`/`#+results:` suppresses. The code agrees: `parseSrcBlockResult` (`block.go`:L120-129) and `WriteBlock`'s condition `params[":exports"] != "code" && … != "none"` (`html_writer.go`:L181). |
| **R3-2** | major / blocking | **CLOSED** | **(a) Footnote.** `spec.md`:L569-591 states the column-zero opening with an ASCII label, followed by whitespace or end of line. That matches `footnoteDefinitionRegexp` `^\[fn:([\w-]+)\](\s+(.+)\|\s*$)` (`footnote.go`:L13). It also states that the end condition is not tested inside a nested block or environment. Row 41 → `""` (measured). The QUOTE and `\begin{align}` blank-pair variants → `""`, and `[fn:메모] 노트`, `[fn:1]노트`, and ` [fn:1] 노트` each leave the next line in place. `[fn:a-b_1]` alone → `""`. **(b) Property drawer.** `spec.md`:L592-608 adds the all-property-interior, no-blank, `:END:`-reached precondition, which matches `parsePropertyDrawer` (`drawer.go`:L62-82). Measured: row 42 → `<p>문맥</p>\n<p>:PROPERTIES:\n도쿄 :-&gt; 일본</p>\n<p>:END:</p>\n` (in place). A blank line inside → in place. No `:END:` → in place. Lowercase `:properties:`…`:end:` → suppressed. Inside `:NOTES:` → in place. Row 36 is still suppressed. **(c) LaTeX.** `spec.md`:L539-541: "the `\begin` and `\end` keywords as well as the name". Measured: `\BEGIN{align}`…`\END{align}` → raw, and `\end{ALIGN}` closes `\begin{align}`. **§ 5 narrowed.** `spec.md`:L1253-1258: "That claim is no wider than those two shapes." **The author's deviation is accepted on measurement.** `[fn:1] 메모`⏎`- a`⏎⏎⏎`도쿄 :-> 일본` → `<p>\n도쿄 :-&gt; 일본</p>` (in place). The `:NOTES:` drawer holding a blank pair → `…<p>:END:\n도쿄 :-&gt; 일본</p>` (in place). In the code, `parseListItem` and `parseDrawer` call `parentStop` (`list.go`:L102-104, `drawer.go`:L37-39), while `parseBlock`'s and `parseLatexBlock`'s stops ignore it (`block.go`:L75-77, L107-109). r3's wording ("nested blocks, environments, **and lists**") was wrong about lists, and the SPEC is right not to follow it. |
| **R3-3** | minor / blocking | **CLOSED** | REQ-SW-013.4 (`spec.md`:L1097-1129) binds the mirror check to the ON direction ("Where the command is turning the option on and …"). On OFF it requires that the command "shall fire no multiline-conflict prompt, and shall read, clear, or write no multiline option". Mirrored in `plan.md` DD-10 (L743-760) and M5 (L882-887: "runs on the ON path only … one for the mirror check on a heading carrying its own `ANKI_DIRECTION`"). AC-SW-014 gains the r3-specified case unstubbed (`acceptance.md`:L497-505: own `ANKI_DIRECTION: ->`, inherited swift on; then `ANKI_SWIFT` falsy, `ANKI_DIRECTION` still `->`, `ANKI_INCREMENTAL` not created, no prompt). The both-on case now states its direction (L489, "the **on** direction"). Twin check: the note-type gate (13.3, L1054-1067) and the conflict gate (13.4) both carry an explicit direction. No third gate exists on the OFF path in the text. |
| **R3-4** | minor / blocking | **CLOSED** | `acceptance.md`:L298-301 replaces "would have carried" with a direct observation of the checked split's `(rest, extra, err)`. That matches the real signature (`extra.go`:L92). |
| **O-2** | minor / optional | **TAKEN** | REQ-SW-001.6 fixes the composition (`spec.md`:L677-684): block and environment regions are computed independently over the whole body, the footnote and source-result rules read those regions, the property-drawer and list rules are computed on their own, and the excluded set is the union. Mirrored in `plan.md`:L502-524. § 5 is restated for that composition (`spec.md`:L1245-1280). Parser side re-measured: env-first → `…\end{align}\n<p>A :-&gt; B</p>\n<p>#+END_QUOTE</p>`; src-first → `…</div>\n<p>A :-&gt; B</p>\n<p>\end{align}</p>`; quote-first → `<p>#+BEGIN_QUOTE</p>\n\begin{align}\n#+END_QUOTE\nA :-&gt; B\n\end{align}`. Scanner side (reasoned from `blockRegions`, `multiline.go`:L183-214): env-first pairs the quote across the environment's close and excludes the arrow line (over-excludes). Src-first: the independent environment scan opens inside the source interior (over-excludes). Quote-first: the environment region holds the arrow line (agrees). All three match § 5's stated outcomes. |

### Regression check — iteration-2 and iteration-1 findings

| # | Status | Evidence (current text) |
|---|---|---|
| N1 | still closed | REQ-SW-001.5 is still definitional ("shall **not** determine it by matching a list", L492-494), and the informative list says "include" (L634-641). |
| N2 | still closed | REQ-SW-011.3 is qualified "to a line's block-level element" (L980). REQ-SW-001.7 is stated at block level (L720-730). The § 5 inline exclusion is at L1282. |
| N3 | still closed | § 4 "Deliberately NOT reused — the container-attribute mechanism" (L1165); § 8 "**not** consumed (§ 4)" (L1476); AC-SW-011 "the arrow is styleable" (`acceptance.md`:L378); Residual Risk "markers, emphasis, and an inline break" (`acceptance.md`:L657). |
| N4 | still closed | "stated as three rather than as every route" (`spec.md`:L1022; `acceptance.md`:L454). |
| N5 | still closed | REQ-SW-013.3 is ON-only (L1054-1075). AC-SW-014's `imoogi-Basic` OFF case is at L516-525 and the no-note-type OFF case at L527-531. |
| N6 | still closed | Corpus rows 27-29 (`acceptance.md`:L51-53) are cited by AC-SW-001e2. |
| D1, D2, D3, D12 | still closed | Non-greedy span (L288, L792-797). The first-token rule (L463-476) with row 20. Row 23. AC-SW-011's `grep -n '#+ATTR_HTML' internal/anki/orgdoc/swift.go` (`acceptance.md`:L403). |
| D5, D9 | still closed | The off path fires no swift-conflict prompt (L1083-1093). "clear **every** multiline option" (L1101). |
| D6-D8, D10, D11, § 5 wording | still closed | `plan.md`:L63 points at the Quality Gate table. Constraint 6 is at L1364. § 5 "Nothing is dropped" is at L1210. The corpus count is "Forty-two rows" (`acceptance.md`:L16), equal to the rows numbered 1-42. AC-SW-002's zero-row list (L157-158) now adds 39 and 41, the two new zero rows. |

**No iteration-1, -2, or -3 defect has reopened.**

---

## Defects Found (structured defect-list)

### R4-1 — The plan's footnote mirror mechanism skips only blank pairs inside nested regions, while the rule it mirrors suspends all three end conditions there

`plan.md`:L446-447, against its twin `spec.md`:L575-581 and `plan.md`:L439-444 — Severity: **minor** — Class: **blocking** (internal consistency, silent-failure direction)

This is the recurring class: a fix that is right at one site and narrower at its
twin. `spec.md` states the rule correctly:

> "go-org continues a definition across the lines below it until the second of
> two consecutive blank lines, a headline, or another footnote-definition line.
> **That end condition** is tested only **between** the constructs the
> definition holds, never inside a `#+BEGIN_` block or a LaTeX environment
> nested in it" (L575-580)

`plan.md`:L439-444 also says it correctly: the whole stop function is handed to
`parseMany`, and `parseBlock`/`parseLatexBlock` "ignore the parent's stop". But
the sentence that tells the implementer **how to mirror it** narrows it:

> "The scanner mirrors this by reading the block and environment regions and
> **skipping blank pairs** that fall inside them." (L446-447)

A footnote-definition line inside a nested block also does not end the
definition. Measured:

```
IN  "[fn:1] 메모\n#+BEGIN_SRC org\n[fn:2] x\n#+END_SRC\n도쿄 :-> 일본\n"        OUT ""
IN  "[fn:1] 메모\n#+BEGIN_EXAMPLE\n[fn:2] x\n#+END_EXAMPLE\n도쿄 :-> 일본\n"   OUT ""
IN  "#+BEGIN_SRC org\n[fn:2] x\n#+END_SRC\n도쿄 :-> 일본\n"                     OUT "…</div>\n<p>도쿄 :-&gt; 일본</p>\n"
```

The third line is the converse. A `[fn:` line inside a block opens nothing, so
a correct scanner must not open a definition inside a region either. Suppose a
scanner is built to L446-447 and gets that converse right. It ends the
definition at `[fn:2]` and opens nothing there. It then admits `도쿄 :-> 일본`
and composes a marker that renders to nothing. That is the silent failure the
SPEC's own L643-649 calls the worst property. No corpus row pins the shape
either way. (The headline condition cannot arise, because the scan cuts the
body at the next headline.)

Reachability is near nil. It needs a footnote definition that holds an Org
code or example sample with a column-zero `[fn:N]` line, followed by an arrow
line before any blank pair. That is why the severity is minor and the score
impact is one Clarity band. It is still classified blocking under M6, because
it is an internal inconsistency in the silent direction on a mirrored rule. It
is also exactly the shape this iteration was asked to look for.

**Required fix (one sentence in `plan.md`:L446-447).** Replace "skipping blank
pairs that fall inside them" with "suspending every end condition (a blank
pair or a footnote-definition line) inside them, and opening no definition
inside them". Optionally, add the first probe above as a corpus row (→ 0).

### O-1 — The source-block-result rule's extent is elaborated for a paragraph and a drawer only

`spec.md`:L612-626, `plan.md`:L479-482 and L490 — Severity: **minor** — Class: **optional**

The rule is stated correctly and generally: "the **one** construct `parseOne`
reads from the very next line form the result" (`plan.md`:L479-480). Only two
extents are spelled out: a paragraph's, and a `:RESULTS:` drawer "suppressed
whole". go-org's construct extents reach further, and the following were
measured under `:exports code`:

- `#+RESULTS:`⏎`#+NAME: r` (or `#+CAPTION: c`, or `#+ATTR_HTML: :class x`)⏎`도쿄 :-> 일본` → the arrow line is suppressed. The affiliated keyword's node is part of the one construct (`keyword.go`:L59-60, L75-78).
- `#+RESULTS:`⏎`:RESULTS:`⏎`도쿄 :-> 일본`⏎⏎⏎`서울 :-> 한국`, with **no** `:END:` → both arrow lines are suppressed. An unclosed ordinary drawer runs to the end of the body (`drawer.go`:L44-58).
- A nested `#+BEGIN_SRC`…`#+END_SRC`⏎`#+RESULTS:`⏎arrow, as the attached construct → suppressed.

An implementer who reads the paragraph and drawer examples as the whole
specification under-excludes in these shapes. The normative text already
covers them, and none is a shape an author writes with an arrow in it, so this
is optional. The suggested fix is one sentence: the extent is whatever
`parseOne` consumes, including an affiliated keyword's following construct and
an unclosed drawer running to the body's end.

Related, and also optional: `blockRegions` returns only `interior []bool`
(`multiline.go`:L183). An **empty** source block has no interior line, so its
close cannot be located from that output. Measured, `#+BEGIN_SRC sh :exports code`⏎`#+END_SRC`⏎`#+RESULTS:`⏎arrow
→ suppressed. "Reuses `blockRegions` unchanged" plus "reading those regions"
therefore cannot see that block. This is unreachable in practice, and the
implementer will meet it as an implementation detail.

### O-2 — `progress.md` contradicts itself on the iteration-4 authorization

`progress.md`:L18 versus L28-33 — Severity: **minor** — Class: **optional**

The YAML says `next_audit_authorization: granted (decision board …)`. The prose
below it says the pass "is **pending authorization**. Until that is given, these
artifacts are … not re-audited". The prose is stale. (`spec.md` HISTORY L24-25
says "still pending", but that is a dated record and is accurate as of its
writing.)

---

## Could not verify

1. **The scanner side of § 5's interleavings, and of R4-1 and O-1.** These are
   reasoned from `blockRegions`' code and from the rules as written. No swift
   scanner exists to measure. The region arithmetic for all three
   interleavings does match § 5.
2. **Anki-side consequences** of a hidden marker (a refused note, or a missing
   card). These are reasoned from Anki's cloze rule. AnkiConnect is not
   reachable here.
3. **The `7d98ec0` baseline** (`plan.md` § A.2) was **not** re-run this
   iteration. HEAD is unchanged from iteration 3, where r3 re-measured it. That
   measurement is carried forward and is not a fresh one.
4. **Whether an unstubbed confirmation prompt is observable in batch ERT**
   (AC-SW-014's "no prompt fires"). The criterion is binary as written, but the
   harness mechanics are a run-phase matter.
5. **Completeness of the context-dependent list** beyond the four constructs.
   This audit did not re-read all thirteen `parseOne` cases. The SPEC no longer
   claims completeness (L698-705), so the obligation rests on the rule, not on
   the list.

---

## Recommendation

**PASS at 0.92** against the Tier M threshold of 0.80. The must-pass firewall
is clean (MP-1, MP-2, MP-3, MP-5, MP-6, MP-7 PASS; MP-4 N/A).

- **R3-1** is closed. The completeness claim is struck, the fourth construct is
  mirrored with nine measured edges, DD-4 and M1 are corrected, and rows
  39/40 reproduce.
- **R3-2** is closed. All three rules now match go-org's code, § 5 is narrowed,
  and rows 41/42 reproduce. The author's departure on lists and ordinary
  drawers is correct, and r3's wording was not.
- **R3-3** is closed. Both OFF-path gates are directed, and the unstubbed OFF
  case is in AC-SW-014.
- **R3-4** is closed. AC-SW-008 observes `(rest, extra, err)` directly.
- **O-2** is taken. The union composition is stated, and § 5's three outcomes
  follow from it.

**Before or with run-phase entry:** the orchestrator should apply R4-1's
one-sentence `plan.md`:L446-447 correction, or name it explicitly in the M1
delegation to manager-develop. The instruction would be: "the footnote rule
suspends every end condition, and opens no definition, inside a block or
environment region; `[fn:1] 메모`⏎`#+BEGIN_SRC org`⏎`[fn:2] x`⏎`#+END_SRC`⏎`도쿄 :-> 일본` → 0 arrow lines". O-1 is worth one line in the same delegation. O-2 is housekeeping.

**What is strong and should not be disturbed.** Every rule restated in v0.1.4
was written from go-org's code rather than from probes, and every edge it
claims reproduces. The author measured r3's own required-fix wording and
rejected the part that was wrong instead of copying it. The composition is now
stated, and its divergences all fall in the over-exclusion direction, as § 5
says. Across four iterations, every finding is closed and none has reopened.
