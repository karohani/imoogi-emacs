# SPEC Review Report: SPEC-ANKICARD-004

Iteration: 3/3 (final iteration of the retry loop)
Verdict: **FAIL**
Overall Score: **0.75** (harmonic mean; Tier M PASS threshold is ≥ 0.80)
STOP signal: **not raised.** Iteration 3 scored 0.75, the same as iteration 2's
0.75, so the score did not regress. The iteration cap is reached, so this
report is also the **escalation report** the Retry Loop Contract requires (see
§ Escalation).

Reasoning context ignored per M1 Context Isolation. The audit is scoped to the
iteration-2 defect delta (N1-N6), a regression check over every iteration-1 and
iteration-2 finding, and the material v0.1.3 introduced. Measurements were taken
in a scratch Go module outside the checkout
(`/private/tmp/.../scratchpad/r3`), against `go-org v1.9.1` from the module
cache (`/Users/jay/go/pkg/mod/.../go-org@v1.9.1/org/`), at worktree HEAD
`7d98ec0`. The SPEC files under audit are the uncommitted v0.1.3 revision. No
git command that writes was run. `git status --short` was identical before and
after the one test run (`make test-elisp`, with `TMPDIR` in the scratchpad).

---

## Verdict in one paragraph

All six iteration-2 findings are closed as they were written, and no
iteration-1 or iteration-2 finding has reopened. Most of the new material holds
up. The SPEC-ANKICARD-005 propagation is accurate. All six new corpus rows
reproduce exactly against go-org. The 13-case `parseOne` switch the SPEC cites
is real, and the § 8 diff claim and the `7d98ec0` baseline both re-measure
exactly. The verdict is FAIL because the revision's **context-dependent half
repeats the defect it struck from the context-free half**. REQ-SW-001.6 now says
that at v1.9.1 "the remaining constructs **are** the three", and that the list
was "derived from [the switch], not collected by probing". I measured a fourth
construct governed by that same switch: a `#+RESULTS:` attached to a
`#+BEGIN_SRC … :exports code|none`, which renders the following paragraph to
nothing (R3-1). Separately, two of the three mirrored rules are written in a
form that differs from the parser's rule. The footnote rule, as written,
under-excludes in the silent-failure direction, and the property-drawer rule
over-excludes (R3-2). The N5 fix split the note-type gate by direction but left
the second gate on the same OFF path unsplit (R3-3). These three findings belong
to the class iterations 1 and 2 named: **a stated predicate that does not match
the obligation above it.** A fourth, smaller blocking finding is pre-existing:
AC-SW-008 asserts a counterfactual no command can observe (R3-4). It holds
Testability at 0.75, and it is load-bearing for the score (see § Sensitivity).

---

## Must-Pass Results

- **[PASS] MP-1 REQ number consistency.** `grep -c '^#### REQ-SW-' spec.md` → `14`;
  headings run `REQ-SW-001` … `REQ-SW-014` (`spec.md`:L381-973), contiguous and
  zero-padded. `grep -o '^### AC-SW-0[0-9][0-9]' acceptance.md | sort | uniq -d`
  → no output.
- **[PASS] MP-2 GEARS format compliance** (judged on the **requirement layer**
  only). Each of the 14 headings carries exactly one pattern tag: `[Where]` ×2
  (REQ-SW-001, -002), `[Ubiquitous]` ×10, `[Ubiquitous — negated]` ×2 (REQ-SW-005
  at L631, REQ-SW-012 at L842). For example, L382: "Where an entry is a swift
  entry, the back end shall take its cards from the **arrow lines**". The
  Given-When-Then entries in `acceptance.md` are the verification layer and are
  graded under Group 4, not here.
- **[PASS] MP-3 YAML frontmatter validity.** `spec.md`:L2-15 carries all 12
  canonical fields with correct types: `version: "0.1.3"` (quoted),
  `status: draft`, `created: 2026-09-21`, `updated: 2026-10-12`, `priority: P2`,
  `lifecycle: spec-anchored`, and `tags` as a comma-separated string. No rejected
  alias is present. The optional `tier: M` and `depends_on` are also present.
- **[N/A] MP-4 language neutrality.** The SPEC is scoped to this repository's Go
  packages and Emacs Lisp modules. It is not template-bound content.
- **[PASS] MP-5 D7 cross-SPEC reconciliation.** The SPEC references
  SPEC-ANKICARD-001/-002/-003 (`status: in-progress`) and SPEC-ANKICARD-005
  (`status: completed`). None is retired, superseded, or archived, so no
  reconciliation obligation fires, and the D7 verb emits no BLOCKING line. The
  design record `.moai/reports/anki-card-types-plan-20260920.md` exists.
- **[PASS] MP-6 D8 cross-platform discipline.** `grep -c syscall` = 0 in each of
  spec.md, plan.md, and acceptance.md. Auto-PASS per D8-4.
- **[PASS / N/A] MP-7 clarification gate.** `grep -rn '\[NEEDS CLARIFICATION'`
  over the SPEC directory → no output. That is PASS on `plan.md`. `research.md`
  does not exist (correct for Tier M), so the criterion is N/A on that artifact.

No must-pass criterion fails. **The FAIL rests on the aggregate score being
below the Tier M threshold, and the score is held there by four blocking
findings (R3-1 to R3-3 in Clarity, R3-4 in Testability).**

---

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.50 | 0.50 — "Multiple requirements require interpretation. A reasonable engineer might implement them differently than intended." | Three loci. In each, a reasonable engineer follows the explicit statement and builds something other than the stated obligation. **R3-1**: `spec.md`:L541-548 obligates mirroring "every construct that can make a later line something other than in-place paragraph text", then states "At go-org v1.9.1 the remaining constructs are the three". Measured, there are four. `plan.md`:L722-724 (M1) tells the implementer to build "the three mirrored … rules". **R3-2**: `spec.md`:L499-501 and `plan.md`:L405-408 state the footnote continuation as a line-level blank-pair rule, which is not the parser's rule (measured). `spec.md`:L506-508 and `plan.md`:L410-412 state the property-drawer rule without the parser's precondition (measured). **R3-3**: REQ-SW-013.4 (`spec.md`:L954-967) states the mirror conflict check for "the command" with no direction. This is the N5 shape on the second gate of the same OFF path. |
| Completeness | 1.00 | 1.0 | HISTORY (L18), Overview/Purpose (L279-291), Scope (L293), Glossary (L314), Requirements (L354), Relationship (L987), Out of Scope (L1026: seven `### Out of Scope — <topic>` H3 headings, each with specific `-` bullets, matching § 1.2's "the seven exclusions"), Constraints (L1170), Brownfield Delta (L1201), Dependencies (L1284), Traceability (L1317). Frontmatter is complete. The Tier M trio is present, plus `progress.md`. |
| Testability | 0.75 | 0.75 — "One AC is not precisely binary-testable but is measurable with minor interpretation." | No weasel word: `grep -nio 'appropriate\|adequate\|reasonable\|properly\|as needed\|if necessary\|sufficiently\|good enough' acceptance.md` → no output. Every criterion carries a deciding command (`grep -c '^```bash'` = 15), and all 15 `go test -run` patterns are anchored (`^…$`, L135-447). The one AC that falls short is **AC-SW-008** (R3-4). Its first clause (`acceptance.md`:L282-284), "the `Back Extra` field **would have carried** that content had the entry rendered", is a counterfactual about an entry the same clause skips, so no field exists to observe and the cited commands cannot decide it. AC-SW-013's docstring clause and AC-SW-015's message-content clause are decided by reading. They do **not** lower the band, because the 1.0 rubric allows "a human tester can determine PASS or FAIL". |
| Traceability | 1.00 | 1.0 | Every REQ is named in `acceptance.md`. Per-REQ hit counts: 001→11, 002→1, 003→1, 004→1, 005→2, 006→1, 007→2, 008→1, 009→1, 010→2, 011→3, 012→2, 013→2, 014→1. All 15 AC headings name an existing REQ. No orphan AC and no uncovered REQ. Counts are 14/15, inside the Tier M ceiling of 16. |

Harmonic mean: `4 / (1/0.50 + 1/1.00 + 1/0.75 + 1/1.00)` = `4 / 5.333` =
**0.75**. The Tier M threshold is **≥ 0.80**
(`spec-workflow.md` § SPEC Complexity Tier, L141 and L329-330).

**Sensitivity, stated so the escalation is not misread.** The score is 0.05
below the threshold, and **either dimension alone is enough to move it.**

- Testability: fixing R3-4 alone, with Clarity unchanged at 0.50, gives
  `4/(2+1+1+1)` = **0.80**. That meets the threshold on score while R3-1 to
  R3-3 are still open.
- Clarity: fixing R3-1 to R3-3, with Testability unchanged at 0.75, gives
  `4/(1+1+1.333+1)` = 0.92.
- Both: fixing all four gives 1.0.

R3-4 is therefore load-bearing for the verdict, and that is why it is
classified blocking rather than optional (see R3-4). The user should read the
FAIL as **four small blocking defects, one of which is a one-sentence AC
rewrite**, not as a SPEC that is far from passing.

### Why the score held at 0.75 rather than rising

The revision did close everything iteration 2 asked for, and each closure is
verified below. Clarity stayed at 0.50 because the new context-dependent half
reproduced N1's shape one layer down. HISTORY (L37-46) identified the defect
correctly ("the context-dependent half was itself a closed list of three — the
same defect one layer down"), widened the list from three to five constructs,
and then asserted the five complete. It did so in the normative sub-clause
(L546-548) and with a derivation claim (L558-564) that the measurement below
falsifies. The SPEC's own anti-pattern 12 (`plan.md`:L794-797) records this
lesson three times over.

---

## Closure table — iteration-2 findings (N1-N6)

Each row was verified against the current text, not carried forward.

| # | Sev / class (iter 2) | Status | Evidence |
|---|---|---|---|
| **N1** | major / blocking | **CLOSED as written. The same class recurs in new material; see R3-1 and R3-2.** | The closed four-shape list is gone. REQ-SW-001.5's informative list (L512-519) now reads "these **include**", and it adds the fixed-width line and the footnote definition (corpus rows 25 and 26). REQ-SW-001.6 (L528-533) requires the classification to come "from the parser's own classification rules … never from a list of named line shapes", and says the two halves are "neither … a closed set". The per-line parse is recorded as one route in `plan.md` DD-4 (L352-367) and is no longer a `shall`. No text implies a position-mapped route; L555-556 says one is unavailable. I re-confirmed the context-free half: the shapes rows 22 and 25-29 name are classified by their own line. |
| **N2** | major / blocking | **CLOSED** | `### Out of Scope — Arrows inside an inline construct` (L1103-1144) carries a six-row measured table and a stated reason for disclosure over repair, and records the rejected render-twice alternative. Constraint 7 is qualified to "block-level" (L1190-1199). REQ-SW-011.3 is qualified (L836-840). REQ-SW-001.7 is stated at block level, with the inline carve-out (L577-587). REQ-SW-005's head clause is scoped to the cloze marker (L635-641). The glossary entry is restated (L334-338). Corpus rows 31 and 32 and AC-SW-001g pin the broken output deliberately. |
| **N3** | minor / blocking | **CLOSED** | All five sites are corrected. § 1.2 (L297-299) says "the arrow's own class and its stylesheet rule". § 4 (L1003-1010) has a named non-reuse entry: "Deliberately NOT reused — the container-attribute mechanism". § 8 (L1294-1295) says "deliberately **not** consumed". AC-SW-011's heading (L361) says "the arrow is styleable". Residual Risk 4 (L627) says "markers, emphasis, and an inline break". A grep for residual `container` wording finds only historical or negating uses. |
| **N4** | minor / blocking | **CLOSED** | REQ-SW-012.4 (L873-889) says "the three together exclude the three evasions each leg is aimed at". It requires the docstring to state "three rather than … every route", and it names the fold-into-an-existing-argument route as not excluded. This is mirrored in `plan.md` DD-5 (L536-548) and AC-SW-013 (L434-440), which fails a docstring that claims "every way". |
| **N5** | minor / blocking | **CLOSED for the note-type gate. A sibling gate on the same path is open; see R3-3.** | REQ-SW-013.3 (L911-938) binds the note-type contract to the ON direction. The OFF path writes the falsy spelling whatever the note type, and creates no note type. The justification against the tree holds: I read `card_option_needs_cloze` (`modules/org/anki/imoogi-error.el`:L72), and it says "switch it off for this heading alone by giving it the value `nil` in its own drawer". AC-SW-014 gained both required cases: the `imoogi-Basic` heading with inherited swift-on and the prompt unstubbed (L489-498), and the no-note-type OFF case (L500-504). |
| **N6** | minor / optional | **CLOSED** | Corpus rows 27-29 (`acceptance.md`:L50-52) cover the heading, keyword, and comment shapes. AC-SW-001e2 (L91-96) cites them. |

### Regression check — iteration-1 findings

| # | Status | Evidence (current text) |
|---|---|---|
| D1 (critical) | still closed | REQ-SW-005.1 (L643-668): span = Anki's non-greedy extent, an unclosed opening runs to the body's end, an extent scan is required. Corpus rows 18, 19, and 24. Row 24 still discriminates body-wide from line-local reading (AC-SW-005, L208-215). |
| D2 | still closed | REQ-SW-001.3 (L398-411). Corpus row 20. |
| D12 | still closed | REQ-SW-011.3 (L824-840) forbids line insertion. AC-SW-011 requires `grep -n '#+ATTR_HTML' …swift.go` to return nothing (L386, L391). |
| D3 | still closed (dissolved) | REQ-SW-011.1's arrow-successor condition (L805-817). Corpus row 23. |
| D4 | still closed (see N4) | — |
| D5 | still closed for the swift-conflict prompt; see R3-3 for the mirror check | REQ-SW-013.3 (L940-950); AC-SW-014's unstubbed clause (L466-470). |
| D9 | still closed | REQ-SW-013.4 (L954-962) says "clear **every** multiline option that is on". |
| D6, D7, D8, D10, D11, §5 wording | still closed | Corpus count measured at 38 = "Thirty-eight rows" (L16). `plan.md`:L56-57 points at the Quality Gate table. Constraint 6 is at L1185-1189. The Unicode trim is at L412-422. AC-SW-007's test-based bound is at L263-270. § 5 L1048-1053 still says "Nothing is dropped". |

**No iteration-1 or iteration-2 defect has reopened.**

---

## New material — what was checked, and what held

| Claim in v0.1.3 | Result | Evidence |
|---|---|---|
| Corpus rows 33-38 | **all six reproduce exactly** | Measured renders: row 33 `""`; row 34 `<p>\n도쿄 :-&gt; 일본</p>`; row 35 `<p>\begin{align}\n도쿄 :-&gt; 일본</p>`; row 36 `<p>문맥</p>`; row 37 `<p>\begin{align} A :-&gt; B \end{align}</p>`; row 38 `<p>A :-&gt; B</p>`. Row 30 is raw (`\begin{align}\nA :-&gt; B\n\end{align}`). |
| `parseOne` is "a finite switch of thirteen cases" | **true** | `org/document.go` `parseOne`: list, table, beginBlock, beginLatexBlock, result, beginDrawer, text, example, horizontalRule, comment, keyword, headline, footnoteDefinition = 13. |
| LaTeX rule: line-anchored, same-name close compared case-insensitively, an unclosed opening opens nothing | **true for the name** | `org/block.go`:L32-33 `(?i)^(\s*)\\begin{([^}]+)}(\s*)$`; `parseLatexBlock` (L104-118) returns `0, nil` when unclosed. The `(?i)` also covers the `\begin` keyword itself (see R3-2 note c). |
| § 5 interleaving: env-first agrees, quote-first diverges, raw-first does not diverge (parser side) | **parser side true** | env-first → `\begin{align}\n#+BEGIN_QUOTE\n\end{align}\n<p>A :-&gt; B</p>…`; quote-first → `<p>#+BEGIN_QUOTE</p>\n\begin{align}\n#+END_QUOTE\nA :-&gt; B\n\end{align}`; src-first → `…</pre>…<p>A :-&gt; B</p>…`. The scanner side depends on composition; see optional O-2. |
| SPEC-ANKICARD-005 propagation: split → **check** → compose → gate → render | **true** | `RenderWithOptions` (`multiline.go`:L590-600) and `Render`'s cloze branch (`orgdoc.go`:L81-84) both call `splitExtraBlocksChecked`. `planner/multiline.go`:L63-64 maps `*ExtraBlockUnbalancedError`. REQ-AKX-004 (SPEC-ANKICARD-005 `spec.md`:L175-178) runs the check "immediately after the split and before every other render-time diagnostic". REQ-SW-010's "can never co-occur" argument (L792-797) follows. SPEC-ANKICARD-005 is `status: completed`. |
| § 8: `git diff 08f7504 7d98ec0 -- multiline.go` touches only `composeMultiline`'s comment and `RenderWithOptions` | **true** | Two hunks: L475 (the `composeMultiline` doc comment) and L589 (the `RenderWithOptions` comment plus the checked-split call). |
| Baseline at `7d98ec0` | **true, re-measured by this audit** | `go test -count=1 -cover ./internal/anki/...` → orgdoc 100.0%, planner 92.8%, hashing 100.0%, model 98.4%. `make test-elisp` → exit 0, `Ran 645 tests, 634 results as expected, 0 unexpected, 11 skipped`. |
| Anchored `go test -run` patterns; "the eight pre-existing names were confirmed" | **true** | All 15 `go test -run` lines (`acceptance.md`:L135-447) use `'^Name$'` or `'^(A\|B)$'`. Each of `TestRenderSignatureIsUnchanged`, `TestRenderGoldenCorpus`, `TestMultilineGoldenCorpus`, `TestGatePrecedesRendering`, `TestSwiftWithMultilineOptionIsRejected`, `TestHashIsUnchangedByCardOptions`, `TestHashCallSiteTakesFourArguments`, and `TestCardOptionValueRecognition` resolves to exactly one `func` under `internal/`. |
| N5: note-type check only on the ON path | **sound for that gate** | See the N5 row. The sibling gate is R3-3. |
| AC-SW-002's zero-row list | **consistent** | "rows 7 through 13, 18 through 22, 24 through 30, 33, and 36" equals exactly the corpus rows recording `0`. |

---

## Defects Found (structured defect-list)

### R3-1 — The context-dependent set is asserted complete at v1.9.1, and a fourth construct exists

`spec.md`:L546-548 and L558-564; `plan.md`:L381-391 and L722-724 — Severity: **major** — Class: **blocking**

This is N1's shape one layer down, and the closest thing in this audit to
stagnation. The normative sub-clause states the obligation generally at
L541-543 ("mirroring the parser's own opening-and-continuation rule for
**every** construct that can make a later line something other than in-place
paragraph text"). It then scopes that obligation for the pinned version at
L546-548:

> "At go-org v1.9.1 the remaining constructs **are** the three sub-clause 5
> measures: the LaTeX environment, the footnote definition, and the property
> drawer."

L558-564 claims the list is complete by derivation: "that dispatch is a finite
switch of thirteen cases, read for this revision. The informative list … is
derived from it, not collected by probing." `plan.md`:L384-387 says the
read-out is "**exactly** the block (its verbatim kinds …), the list, the LaTeX
environment, the footnote definition, and the property drawer". It adds that
`#+RESULTS:` "measured, still render[s] it as an in-place `<p>`" (L387-389).
M1 (L722-724) then tells the implementer that the new pieces are "the **three**
mirrored opening-and-continuation rules".

**The `beginBlock` case governs one more later line.** `parseBlock` calls
`parseSrcBlockResult` for a SRC block (`org/block.go`:L96-99). That function
skips blank lines and attaches the next `#+RESULTS:` node to the block
(L120-129). `WriteBlock` then writes that result only when
`:exports` is neither `code` nor `none` (`org/html_writer.go`:L182-184).
Measured:

```
IN  "#+BEGIN_SRC sh :exports code\necho\n#+END_SRC\n\n#+RESULTS:\n도쿄 :-> 일본\n"
OUT "<div class=\"src src-sh\">…<pre>\necho\n</pre>…</div>\n"          <- arrow line gone
IN  "#+BEGIN_SRC sh :exports none\necho\n#+END_SRC\n#+RESULTS:\n도쿄 :-> 일본\n"
OUT ""
IN  "#+BEGIN_SRC sh :exports code\n…\n#+RESULTS:\n도쿄 :-> 일본\n서울 :-> 한국\n"
OUT (code only; both arrow lines gone — the result node is the whole paragraph)
IN  "#+BEGIN_SRC sh\necho\n#+END_SRC\n#+RESULTS:\n도쿄 :-> 일본\n"
OUT "…</div>\n<p>도쿄 :-&gt; 일본</p>\n"                                 <- converse: in place
IN  "#+RESULTS:\n도쿄 :-> 일본\n"
OUT "<p>도쿄 :-&gt; 일본</p>\n"                                         <- the case plan.md measured
```

The arrow line reads as a paragraph on its own, sits outside every block
interior, is not list content, and is not in any of the three newly mirrored
constructs. An implementation built to M1 therefore admits it, composes a marker
into it, and the marker renders to nothing. Where it is the entry's only arrow
line, that is the failure L521-527 calls "the worst property": "a note Anki
refuses as a cloze note with no deletion, with nothing announcing it". The
plan's `#+RESULTS:` measurement was correct for the standalone keyword. The
derivation missed that the same keyword is also consumed inside the `beginBlock`
case, under a header-argument condition.

Reachability is low but real. `:exports code` is a common org-babel header, and
`#+RESULTS:` is written by `C-c C-c`. The usual result shape is a fixed-width
`: …` line, which is already excluded context-free. A `:results raw` or
`:results drawer` result can carry paragraph text.

**Required fix.** In REQ-SW-001.6, replace "the remaining constructs are the
three" with a statement that does not assert completeness. Then do one of two
things: (a) add the SRC block's attached result (`#+RESULTS:` after a
`#+BEGIN_SRC` whose `:exports` is `code` or `none`, blank lines between allowed)
as a fourth mirrored construct; or (b) decide explicitly to over-exclude every
result attached to a SRC block, and state that cost. Either way, correct
`plan.md` DD-4's "exactly" (L384) and its `#+RESULTS:` sentence (L387-389), and
change M1's "the three" (L722-724). Add two corpus rows: the `:exports code`
body above → `0`, and the default-exports converse → `1`. Also soften L558-564:
"derived from the switch" was falsified once, so the text should say the list
was read from the switch, not that the reading was exhaustive.

### R3-2 — Two of the three mirrored rules are written differently from the parser's rule, one in the silent-failure direction

`spec.md`:L498-511 and L1085-1090 (§ 5 "Wholly nested shapes agree"); `plan.md`:L405-412 and L420-422 — Severity: **major** — Class: **blocking**

REQ-SW-001.6 obligates the scanner to *mirror* each parser rule. The SPEC and
the plan then write two of those rules down in a form that is not the parser's
rule. The plan's "How each new one is mirrored" (L393-412) is what an
implementer will code from. No corpus row distinguishes the stated rule from the
parser's rule.

**(a) Footnote continuation, as stated, under-excludes. This is the dangerous
direction.** `spec.md`:L499-501 and `plan.md`:L405-408 say a definition
"continues until the second of two consecutive blank lines, a headline, or
another footnote-definition line". That is a rule over body lines. The parser's
rule is a rule over *parsed children*: `parseFootnoteDefinition` calls
`parseMany` (`org/footnote.go`:L22-32), so a nested block or environment
consumes its own lines, blank ones included, before the footnote's stop
condition is checked again. Measured:

```
IN  "[fn:1] 메모\n#+BEGIN_SRC go\n\n\n#+END_SRC\n도쿄 :-> 일본\n"      OUT ""
IN  "[fn:1] 메모\n\\begin{align}\n\n\n\\end{align}\n도쿄 :-> 일본\n"   OUT ""
IN  "[fn:1] 메모\n#+BEGIN_QUOTE\n\n\n#+END_QUOTE\n도쿄 :-> 일본\n"     OUT ""
```

Under the stated rule, the definition ends at the second blank line *inside* the
block. `blockRegions` marks only the block's interior, so `도쿄 :-> 일본` after
`#+END_SRC` is admitted, composed, and rendered to nothing. That is the same
silent outcome as R3-1. A code sample with a blank-line pair between two
functions is an ordinary shape. This also falsifies the general claim at
`spec.md`:L1085-1086, "Wholly nested shapes agree with the parser under the
mirrored rules". Here a SRC block wholly inside the footnote's parser extent
disagrees. That sentence was measured only for env/SRC nesting.

**(b) The property drawer, as stated, over-excludes.** `spec.md`:L506-508 says "A
drawer named `PROPERTIES` renders nothing at all". `plan.md`:L410-412 says the
drawer "opens on `^\s*:NAME:\s*$` and closes at `^\s*:END:\s*$`; only the name
`PROPERTIES` suppresses its interior". In the parser, `parsePropertyDrawer`
(`org/drawer.go`) returns `0, nil` in three cases: any interior line fails
`^(\s*):(\S+):(\s+(.*)$|$)`, any interior token is not text or begin-drawer
(a blank line counts), or no `:END:` is reached. In each case the
`:PROPERTIES:` line falls back to plain text and nothing is suppressed. Measured:

```
IN  "문맥\n:PROPERTIES:\n도쿄 :-> 일본\n:END:\n"
OUT "<p>문맥</p>\n<p>:PROPERTIES:\n도쿄 :-&gt; 일본</p>\n<p>:END:</p>\n"    <- in place
IN  "문맥\n:PROPERTIES:\n:X: 도쿄 :-> 일본\n\n:END:\n"                       <- one blank line
OUT "<p>문맥</p>\n<p>:PROPERTIES:\n:X: 도쿄 :-&gt; 일본</p>\n<p>:END:</p>\n"
IN  ":NOTES:\n:PROPERTIES:\n:X: A :-> B\n:END:\n"                           <- inside a drawer
OUT "<p>:PROPERTIES:</p>\n<p>:X: A :-&gt; B</p>\n"
```

By REQ-SW-001.5's own definition, each of these lines is an arrow line. The
stated rule excludes them. AC-SW-001e3 (`acceptance.md`:L110-112) treats exactly
this as a failure: "They are what fail an implementation that excludes more
than the parser does". The direction is safe (a withheld card, or
`swift_arrow_missing` if the line is alone), which is why this half alone would
be minor.

**(c) Smaller variances, recorded with (a) and (b) because they share the fix.**
`plan.md`:L405 writes the footnote opening as `^\[fn:LABEL\]` with LABEL
unspecified. The parser's opening is `^\[fn:([\w-]+)\](\s+(.+)|\s*$)`: an ASCII
label, followed by whitespace or end of line. Measured, `[fn:메모] 노트` and
`[fn:1]노트` are **not** definitions, and the line after each renders in place.
A Korean label is plausible in this project. Any looser LABEL over-excludes. The
LaTeX regex is `(?i)` on the `\begin` keyword as well as on the name: measured,
`\BEGIN{align}`…`\END{align}` is a raw environment. The SPEC states only that the
name is compared case-insensitively. This under-excludes, but nobody writes
`\BEGIN`, so it carries little weight.

**Required fix.** State each mirrored rule as the parser's rule.
- Footnote: nested blocks, environments, and lists consume their own lines, so
  a blank pair inside them does not end the definition. Give the opening as
  `^\[fn:[\w-]+\]` followed by whitespace or end of line.
- Property drawer: suppresses only when every interior line is a property line
  (`:KEY:` or `:KEY: value`) up to an `:END:`, with no blank line. Otherwise
  the `:PROPERTIES:` line is plain text and nothing is suppressed.
- LaTeX: `\begin` and `\end` themselves are case-insensitive.

Mirror these in `plan.md` DD-4 (L398-412). Narrow `spec.md`:L1085-1086 to the
nestings actually measured. Add two corpus rows: the footnote-plus-SRC-blank-pair
body → `0`, and `:PROPERTIES:`⏎`도쿄 :-> 일본`⏎`:END:` → `1`.

### R3-3 — REQ-SW-013.4's mirror conflict check does not distinguish the OFF path

`spec.md`:L954-967; `acceptance.md`:L472-478 — Severity: **minor** — Class: **blocking**

The N5 fix gave the note-type gate a direction. The second gate on the same OFF
path is still undirected. REQ-SW-013.4:

> "Where the heading resolves **any** multiline option to an on value, the
> command shall report the conflict and shall write nothing unless the user
> confirms. On confirmation it shall clear **every** multiline option that is on"

This applies to "the command", and REQ-SW-013.1 makes the command a toggle. The
rationale (L964-967) is about the ON direction only: "writing swift beside an
active direction would create exactly the entry SPEC-ANKICARD-002's conflict
rule rejects". The trigger clause is unconditional. An implementer who mirrors
the existing `imoogi-anki--resolve-swift-conflict`, which is unconditional
(`modules/org/24-anki.el`, called from `imoogi-anki--write-card-option` before
every write), will fire the check on OFF.

The heading this reaches is the one SPEC-ANKICARD-002's gate rejects with
`card_option_conflict`: swift on, typically inherited, beside an on direction.
That code's message (`modules/org/anki/imoogi-error.el`:L66) says "Remove one of
the two … give it the value `nil` in the heading's own drawer". A user who wants
to keep the multiline card toggles swift **off** and is asked whether to clear
`ANKI_DIRECTION`. Declining writes nothing, so the user cannot opt out of swift
through the command. Confirming clears the very option they meant to keep. This
is iteration-1 D5's failure, reached through the mirror check rather than the
swift-conflict check. It is also the argument L926-932 uses to justify N5's
resolution: "binding the contract to the OFF path too would make the command
refuse the very fix the diagnostic prescribes". AC-SW-014's both-on case
(L472-478) does not state the heading's swift state, so it tests only ON, and
no clause pins OFF. Severity is minor for the same reason N5 was: the shape is
inherited from the existing commands.

**Required fix.** One clause in REQ-SW-013.4 binding the mirror check to the ON
direction. On OFF, write the falsy `ANKI_SWIFT` and touch no multiline option.
Add an AC-SW-014 case: a heading with `ANKI_DIRECTION: ->` in its own drawer and
`ANKI_SWIFT` inherited on; toggle, unstubbed. Then the heading's drawer carries
`ANKI_SWIFT: nil`, `ANKI_DIRECTION` is still `->`, and no prompt fires.

### R3-4 — AC-SW-008's first clause asserts a counterfactual that no command can observe

`acceptance.md`:L279-284, against L7-9 — Severity: **minor** — Class: **blocking**

> "**Then** it is skipped with `swift_arrow_missing` … — and the `Back Extra`
> field **would have carried** that content had the entry rendered."

The entry is skipped by the same clause, so no `Back Extra` field exists to
observe, and neither cited command
(`go test -run '^TestSwiftPipelineOrder$'`, `…'^TestSwiftUnpairedExtraMarkerIsReported$'`)
decides a counterfactual. The test author has to choose an observation, for
example calling the checked split directly. Two test authors can make different
choices.

`acceptance.md`:L7-9 states the criterion this violates: "Each criterion is
binary: it passes or it fails, and **the deciding command's output says
which**." That makes it blocking under M6's definition (a criterion the
document actually states). It is also what holds Testability at 0.75, and so,
by the arithmetic above, it is load-bearing for the verdict. It is pre-existing
and was not raised in iterations 1 or 2. It is raised now because an auditor
that leaves a verdict-deciding defect unclassified would be absorbing it into
the score silently.

**Required fix (one sentence).** Restate the clause as something a test can
observe: the checked split's `extra` output for that body carries the arrow
line, and the composed remaining body carries none.

### O-2 — § 5's interleaving outcomes depend on how the rules are composed, and the SPEC fixes no composition

`spec.md`:L541-546 (reuse of the `#+BEGIN_` pairing) with L1083-1101; `plan.md`:L420-422 — Severity: **minor** — Class: **optional**

§ 5 states its agree and diverge outcomes for "mirrored rules applied in one
sequential pass". The reused `blockRegions` (`internal/anki/orgdoc/multiline.go`:L183)
is a whole-body function that computes block interiors independently of
anything else. Composing it with an independent LaTeX scan takes a union of
regions, not a sequential pass. Reasoned from that code (not measured against
an implementation, since none exists), the union inverts § 5's outcomes:

- env-first: `#+BEGIN_QUOTE`…`#+END_QUOTE` pairs across the environment and
  excludes `A :-> B`. Over-excludes, where § 5 says it agrees.
- src-first: the independent scan opens at the `\begin{align}` inside the SRC
  interior. Over-excludes, where § 5 says it does not diverge.
- quote-first: excluded by the environment scan. Agrees with the parser, where
  § 5 discloses a divergence.

Union is the safer composition (its only divergence is over-exclusion), but the
SPEC should say which composition its § 5 statements describe. This is optional
because every shape involved is one § 5 itself says "no author writes on
purpose".

---

## Escalation (iteration 3 of 3 — Retry Loop Contract)

**Defect history across the three iterations.**

| Iteration | Score | Blocking findings | Their shape |
|---|---|---|---|
| 1 | 0.75 | D1, D2, D4, D5, D9, D12 (+ D3 dissolved) | a predicate narrower than the obligation above it (D1, D2, D12), plus an overclaim of sufficiency (D4) |
| 2 | 0.75 | N1, N2, N3, N4, N5 | N1: a closed context-free list ("only four"); N2: a line-level rule for an inline hazard; N4: an overclaim; N5: a gate without a direction |
| 3 | 0.75 | R3-1, R3-2, R3-3, R3-4 | R3-1: a closed context-dependent list ("are the three"); R3-2: rules written differently from the parser's; R3-3: a second gate without a direction; R3-4: a pre-existing counterfactual clause in an AC |

**Stagnation.** No individual defect appears unchanged across all three
iterations. Every finding from iterations 1 and 2 was closed, and none reopened.
The **class** has appeared in every iteration, and R3-1 and R3-3 are each the
predecessor fix applied to one instance and not to its sibling (the
context-free list to the context-dependent list; the note-type gate to the
conflict gate). That is the pattern this report flags for the user. Revisions
fix the cited instance. They do not re-check the sibling surfaces the fix's own
reasoning reaches.

**What the remaining defects cost to fix.** All four blocking findings have
narrow, text-level fixes and no design change. R3-1 needs one sentence plus one
mirrored construct (or an explicit over-exclusion) and two corpus rows. R3-2
needs three rule statements to be corrected and two corpus rows. R3-3 needs one
clause and one AC case. R3-4 needs one sentence. No requirement or criterion
count changes. Fixing R3-4 alone would put the score at exactly 0.80 while
R3-1 to R3-3 stay open, so a score-only reading of a fourth pass would not tell
the user whether the substantive defects were fixed. Reachability
of every blocking defect is low: org-babel results under `:exports code|none`,
footnote definitions followed by blank-pair code blocks, and a swift OFF toggle
on an already-conflicting heading.

**Options the orchestrator presents to the user** (per contract; this agent
does not ask):

1. **PASS-with-debt.** Accept v0.1.3 and carry R3-1 to R3-4 as named debt into
   `/moai run`. In that case, put R3-1 and R3-2(a) into the M1 delegation as
   explicit rules, because both produce a silent Anki rejection. Also give
   AC-SW-008's test a concrete observation (R3-4).
2. **Targeted fix, then a delta-only re-audit.** This needs an explicit user
   override of the cap. The fixes are small enough that a fourth pass scoped to
   R3-1 to R3-4 would be cheap.
3. **Scope reduction.** Not indicated. The defects do not come from the SPEC
   being too large.

---

## Could not verify

1. **Anki-side consequences** of R3-1 and R3-2(a) (a note refused for having no
   deletion). The rendered fields were measured. Anki's behavior is reasoned
   from the SPEC's own statement and Anki's cloze rule. AnkiConnect is not
   reachable here.
2. **O-2 and R3-2(a)'s scanner-side outcome.** These are reasoned from
   `blockRegions`'s code and from the rule as stated. No swift scanner exists
   yet to measure.
3. **R3-3.** Reasoned from the requirement text, the existing helper's code
   (`modules/org/24-anki.el`, `imoogi-anki--resolve-swift-conflict` and
   `imoogi-anki--write-card-option`), and the diagnostic's message. No
   implementation exists.
4. **`make lint`, `make fmt-check`, `make ci-local`, `go test ./... -count=1`,
   and the Windows cross-build** were not run. The parts of § A.2's baseline I
   did measure (coverage for four packages, and the Elisp count of 645) match
   exactly.
5. **Whether a sixth context-dependent construct exists.** I read all 13
   `parseOne` cases, the keyword dispatch (`NAME`, `SETUPFILE`, `INCLUDE`,
   `LINK`, `MACRO`, `CAPTION`, `ATTR_HTML`), and the HTML writer's per-node
   suppressions, and found only R3-1's. `#+CAPTION:` wraps in place
   (`<figure><p>…</p>`), `#+INCLUDE:` of a missing file leaves the next line in
   place, and a horizontal rule does not touch the next line. That is a reading,
   which is the method R3-1 shows can miss something. It is not a proof.

---

## Recommendation

FAIL at 0.75 against the Tier M threshold of 0.80. Four findings are blocking
and one is optional. Iteration 3 is the last, so the orchestrator escalates to
the user (§ Escalation). If fixes are authorized, apply them in this order:

1. **R3-1** (`spec.md` REQ-SW-001.6 L546-548 and L558-564; `plan.md` DD-4
   L381-391 and M1 L722-724). Remove "are the three". Add the SRC-attached
   result under `:exports code|none` as a fourth mirrored construct, or as an
   explicitly stated over-exclusion. Add two corpus rows.
2. **R3-2** (`spec.md` L499-511 and L1085-1086; `plan.md` L405-412). Restate the
   footnote, property-drawer, and LaTeX rules as the parser's rules: nested
   constructs consume their own blank lines; the property drawer needs an
   all-property interior; `\begin` is case-insensitive. Narrow § 5's "wholly
   nested" claim. Add two corpus rows.
3. **R3-3** (`spec.md` REQ-SW-013.4). Bind the mirror conflict check to the ON
   direction. Add the inherited-swift-plus-own-direction OFF case to AC-SW-014,
   unstubbed.

4. **R3-4** (`acceptance.md` AC-SW-008, L282-284). Restate the counterfactual
   "would have carried" as an observation of the checked split's `extra`
   output.

Optional: **O-2** (say which rule composition § 5's outcomes assume).

**What is strong and should not be disturbed.** Every iteration-1 and
iteration-2 finding is closed, verified against current text. The
SPEC-ANKICARD-005 propagation is precise and checked against code. The six new
corpus rows reproduce exactly, and the converse rows (34, 35, 37, 38) do real
work against over-exclusion. The § 5 interleaving disclosure is honest on the
parser side. The baseline was genuinely re-measured, as were the anchored test
names. The disposition of N5 is right, and so is its use of the
`card_option_needs_cloze` message as evidence. R3-3 asks for the same reasoning
to be applied one gate further.
