# SPEC Review Report: SPEC-ANKICARD-003
Iteration: 1/3
Verdict: FAIL
Overall Score: 0.63 (harmonic mean; Tier M PASS threshold 0.80)

Reasoning context ignored per M1 Context Isolation. The audit reads
`spec.md`, `plan.md`, `acceptance.md` (Tier M artifact set), and verifies
every claim it can against the tree at HEAD `6f3ae6c`.

All seven must-pass criteria PASS. The FAIL is driven by the rubric scores,
not by the M5 firewall. Three confirmed internal contradictions and two
mechanically-confirmed unspecified shapes put Clarity and Testability at
0.50 each.

## Must-Pass Results

- **[PASS] MP-1 REQ number consistency** — `REQ-ML-001`..`REQ-ML-015`,
  fifteen `#### REQ-ML-` headings, fifteen distinct numbers, contiguous, no
  gaps, no duplicates, uniform three-digit zero-padding. Verified by
  `grep -o 'REQ-ML-[0-9]*' spec.md | sort -u` (15 values, 001..015) against
  `grep -c '^#### REQ-ML-' spec.md` (15).

- **[PASS] MP-2 GEARS format compliance** — judged against the **requirement
  layer** (`REQ-XXX` entries in `spec.md` § 3) only, per M3 § Scope. The
  Given-When-Then entries in `acceptance.md` are the verification layer and
  are graded under Group 4, not here. Each of the fifteen carries exactly one
  GEARS trigger:
  - `[Where]` — REQ-ML-001 (spec.md:L124), REQ-ML-002 (L146), REQ-ML-008 (L215)
  - `[Ubiquitous]` — REQ-ML-003 (L161) "The resolved direction shall select…",
    REQ-ML-004 (L169), REQ-ML-005 (L180), REQ-ML-006 (L191), REQ-ML-007 (L202),
    REQ-ML-009 (L224), REQ-ML-010 (L243), REQ-ML-011 (L256), REQ-ML-012 (L266),
    REQ-ML-013 (L279), REQ-ML-015 (L318)
  - `[Ubiquitous — negated]` — REQ-ML-014 (L305) "The back end shall not
    change what a **non**-multiline entry renders to"
  The generalized `<subject>` substitution (back end, composition, editor,
  resolved direction) is permitted at Score 1.0. Numbered sub-clauses are
  case-splits on the single trigger's operand and introduce no second
  modality, as spec.md:L108-111 states and inspection confirms. One label
  inconsistency is recorded as D9 below; it does not break the pattern.

- **[PASS] MP-3 YAML frontmatter validity** — all 12 canonical fields present
  with correct types at spec.md:L2-14: `id: SPEC-ANKICARD-003`, `title`
  (quoted string), `version: "0.1.0"` (quoted semver), `status: draft` (in
  the 8-value enum per `spec-frontmatter-schema.md:L22`), `created:
  2026-09-20`, `updated: 2026-09-20` (both ISO `YYYY-MM-DD`), `author: jay`,
  `priority: P2`, `phase`, `module`, `lifecycle: spec-anchored`, `tags`
  (comma-separated string). No rejected snake_case alias (`created_at`,
  `updated_at`, `labels`, `spec_id`) appears. Optional `tier: M` and
  `depends_on: [SPEC-ANKICARD-002]` are additive.

- **[N/A] MP-4 Section 22 language neutrality** — N/A: single-project SPEC.
  The SPEC targets this repository's own Go packages (`internal/anki/*`) and
  its own Elisp modules (`modules/org/anki`). It specifies no multi-language
  tooling surface and enumerates no language-specific analyzer, so the
  16-language enumeration obligation does not bind. N/A auto-passes.

- **[PASS] MP-5 D7 cross-SPEC reconciliation** — the body references exactly
  `SPEC-ANKICARD-001` and `SPEC-ANKICARD-002`. Both directories exist under
  `.moai/specs/`. `grep '^status:'` returns `in-progress` for both. Neither is
  `retired`, `superseded`, or `archived`, so no reconciliation clause is
  required and no BLOCKING finding is emitted.

- **[PASS] MP-6 D8 cross-platform discipline** — `grep -c syscall` returns 0
  for `spec.md`, `plan.md`, `acceptance.md`, and `progress.md`. D8-4
  auto-PASS.

- **[PASS] MP-7 clarification gate** — `grep -rn '\[NEEDS CLARIFICATION'` over
  `spec.md`, `plan.md`, `acceptance.md`, `progress.md` returns no match.
  `research.md` is absent, which is correct for Tier M.

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.50 | 0.50 — multiple requirements require interpretation | Five shapes a reasonable engineer would implement differently: checkbox items (D1), a plain sibling in a description list (D3), block-interior bullet lines against DD-1's stated grammar (D4), the title's number under incremental `<->` (D6), and whether "wrapped content" in REQ-ML-007.1 includes an existing marker's braces (D2). spec.md:L128-145, L191-200, L202-213; plan.md:L136-155 |
| Completeness | 0.75 | 0.75 — one surface sparse, frontmatter complete | Every required section is present and substantive: HISTORY (L18), Overview/Purpose (L33), Scope (L48), Requirements (L101), Out of Scope (L357, six `### Out of Scope — <topic>` H3 sub-headings each carrying specific `-` bullets), Constraints (L411), Brownfield Delta (L428), Dependencies (L497), Traceability (L520). `acceptance.md` carries the AC layer. Deduction: the Edge Cases table (acceptance.md:L396-415), offered expressly so "a reader can check the list is complete", omits two shapes reachable from REQ-ML-001.2 and REQ-ML-001.3 — checkbox items and a mixed description list |
| Testability | 0.50 | 0.50 — several criteria cannot decide what they claim | AC-ML-006a is unsatisfiable as written (D2, acceptance.md:L156-162); AC-ML-014's deciding command names a file with no pairing assertion (D5, acceptance.md:L372); AC-ML-013's named guard becomes non-load-bearing under REQ-ML-010 (D7, acceptance.md:L354); AC-ML-007c names an Elisp function that does not apply the pad it asserts (D8, acceptance.md:L189-198). No weasel words found: `grep -i 'appropriate\|adequate\|reasonable\|proper'` over acceptance.md returns only prose outside criteria |
| Traceability | 1.00 | 1.0 — complete both directions | Every `REQ-ML-001`..`REQ-ML-015` is named in at least one AC heading (`grep '^### AC-ML-'` shows the mapping at acceptance.md:L37,75,102,121,135,154,176,203,214,251,275,297,330,357,375). Every AC heading names a requirement that exists. No orphaned AC, no uncovered REQ. The deliberate off-by-one from AC-ML-011 onward (AC-ML-011↔REQ-ML-012 … AC-ML-014↔REQ-ML-015) is explicit in each heading, not drift |

Aggregate: harmonic mean of (0.50, 0.75, 0.50, 1.00) = 4 / (2 + 1.333 + 2 + 1)
= **0.632**. Tier M PASS threshold is 0.80 (`spec-workflow.md:L329-330`).
The arithmetic mean, 0.688, is also below threshold; the verdict does not
turn on the choice of mean.

---

## Verification Performed (what was confirmed mechanically)

A throwaway probe was written to `internal/anki/orgdoc/`, run against the
vendored go-org v1.9.1 (`go.mod:5`), and deleted. `git status --short
internal/` returns empty and `go build ./...` exits 0 after removal.

**Every go-org claim in the SPEC is confirmed byte-for-byte.** The author's
recorded evidence is honest; nothing in DD-1, DD-2, DD-3, or DD-4 was
overstated or misquoted.

| Claim | plan.md | Probe result |
|---|---|---|
| Nested list renders inside its parent `<li>` | L65-76 | CONFIRMED verbatim |
| Description list emits `<dl>/<dt>/<dd>`, no `<li>` | L84-92 | CONFIRMED verbatim |
| Marker survives go-org in plain / nested / ordered / description-definition | L99-106 | CONFIRMED all four |
| `#+ATTR_HTML: :class children-list` attaches to `<ul>`, `<ol>`, and `<dl>` | L119-128 | CONFIRMED all three |
| `#+BEGIN_CHILDREN-LIST` yields `<div class="children-block">` | L132-134 | CONFIRMED verbatim |
| Marker spanning nested children spans `</p><ul><li>` | L168-171 | CONFIRMED verbatim |
| Whole-item description wrap tears across `<dt>`/`<dd>` | L196-199 | CONFIRMED verbatim |
| Definition-only wrap is clean | L206-209 | CONFIRMED verbatim |
| A generated `{{c1::Tokyo}}` does not turn a plain item into a description item | L216-219 | CONFIRMED (see § Hazard dispositions) |
| Two blank lines split one list into two | L237-241 | CONFIRMED verbatim |
| `splitExtraBlocks` residue is `"- A\n\n\n- B"` and renders as two lists | L248-251 | CONFIRMED on the real function |
| `org.ListItem` carries no source position | L136-139 | CONFIRMED by reading the vendored type |

**Pipeline ordering (brief item 6) is confirmed.** `validateCardOptions` is
called before `orgdoc.Render` on both paths: `planner.go:286` → `:291`, and
`migrate.go:199` → `:208`. REQ-ML-011's "fourth by construction" claim holds
mechanically. `migrate.go:193-198` additionally documents the two early
returns (dry-run, non-candidate) that precede the gate, so the gate's
position is stable under the new code.

**Diagnostic-code surface is confirmed.** All codes the binary emits are in
one const block, `protocol.go:51-92`. The front-end table
`modules/org/anki/imoogi-error.el` pairs them. The two-directional assertion
lives in `tests/anki-error-test.el` — `imoogi-error-test-go-codes-are-subset-of-table`
(L51), `imoogi-error-test-table-entries-are-all-paired-with-a-go-constant`
(L101) — and `imoogi-error-test-codes-match-source-file` (L57) cross-checks
the hardcoded list against `protocol.go` by regexp, so a new code that is
added to Go but not to the Elisp list fails. REQ-ML-015 is implementable as
written; AC-ML-014's deciding command is not (D5).

**Non-interference anchor is confirmed.** `TestRenderSignatureIsUnchanged`
exists at `golden_baseline_test.go:136` and asserts
`func(noteType, title, body string) (map[string]string, error)`. DD-6's
reading of it is correct. `internal/anki/orgdoc/testdata/render-golden.json`
is the corpus AC-ML-013a pins.

**§ 8 does not reintroduce the run-phase gate.** SPEC-ANKICARD-002's own
audit (`SPEC-ANKICARD-002-plan-audit.md`, finding D1) required removing a
`spec.md` clause that gated the run phase on the predecessor's closure.
SPEC-ANKICARD-003 § 8 (spec.md:L497-513) mirrors SPEC-ANKICARD-002 § 8
(spec.md:L1-27 of that file) faithfully: "Proceeding is authorized", "The
dependency is process, not code availability", "The consequence is recorded
rather than mitigated". No gate clause is present. CLEAN.

**Design record § 2.2 was checked for contradiction.** The confirmed
generation rules read: `->` wraps answer items as `{{c1::…}}`, `<-` wraps the
title as `{{c2::…}}`, `<->` both; incremental gives each item its own number.
REQ-ML-003, REQ-ML-005, and REQ-ML-006.2 reproduce this. Decision F ("답 없는
Multiline → `multiline_answer_missing`") matches REQ-ML-002. Decision B
("본문 최상위 목록 항목") matches REQ-ML-001. Decision C matches the
per-answer-note exclusion. **No silent contradiction found.** One derived
inconsistency is recorded as D6.

---

## Explicit answer on the two dispositioned hazards (brief item 3)

### 3a — The description-item hazard: disposition ACCEPTED

The decision to wrap the definition half alone is **correct, and the
specification of it is precise enough to implement for the shape DD-3
actually considers.** Both halves of the evidence reproduce verbatim.

The inverse hazard is **genuinely absent, not assumed.** I checked it
independently rather than taking DD-3's word. go-org's description splitter
requires whitespace before `::`; a generated marker's colons are always
preceded by a digit (`c1::`, `c10::`), so no generated marker can introduce a
split point. Confirmed three ways:

- `- {{c1::Tokyo}}` → `<ul><li>{{c1::Tokyo}}</li></ul>` — stays a plain list.
- `- term:: definition` (no space) → `<ul><li>term:: definition</li></ul>` —
  the whitespace requirement is real, not folklore.
- There is no reachable case of a plain item whose own content carries ` :: `,
  because go-org would already have classified that list as description-kind,
  at which point REQ-ML-001.3 governs.

Where the disposition falls short is a shape DD-3 did not consider: a
**mixed** list, where one item uses `::` and a sibling does not. Recorded as
D3.

### 3b — The `#+BEGIN_EXTRA`-between-answers silent loss: disposition REJECTED

**The call is not sound as it stands, and this is the finding the brief was
right to single out.** The reasoning is coherent and the two rejected fixes
are correctly rejected against real constraints — but the alternatives list
is short by at least one, and the one that is missing survives both
objections the author raised.

What is right: reading the answer list before the split would let
supplementary content become an answer, which REQ-ML-009.1 forbids outright.
Normalizing inside `splitExtraBlocks` would re-hash every existing note
carrying a supplementary block, including non-multiline ones — I confirmed
`splitExtraBlocks` is on the path of every Cloze entry (`orgdoc.go:73`) and
that its byte-identical no-block return is load-bearing (`extra.go:22-27`).
Both rejections stand.

What is missing: **composition-local gap normalization.** Composition already
rewrites Org source, and it runs **only for multiline entries** (REQ-ML-009,
spec.md:L224). Collapsing a blank-line run inside the composition step
therefore touches no non-multiline entry and re-hashes nothing outside this
SPEC's own new surface — it escapes the exact objection the author used to
reject normalizing in `splitExtraBlocks`. The SPEC neither adopts this nor
records why it is rejected.

A second unconsidered option: reporting the dropped answers as a non-skipping
advisory. The advisory-result mechanism already exists and is tested
(`tests/anki-sync-error-test.el:170`,
`imoogi-sync-error-test-advisory-on-ok-run-does-not-collapse-summary`), so the
user would at least learn that content was dropped.

Judgment: the behavior is **silent data loss mitigated only by documentation**
— the author's own framing. An author who interleaves a note between two
answers gets a card missing every answer after the note, with no diagnostic,
no warning, and no signal in the sync report. The mitigation is a README
paragraph the author must have read beforehand. AC-ML-009d pins the behavior
so it cannot change accidentally, which is good practice, but pinning a
silent loss is not the same as dispositioning it. Recorded as D10, blocking:
the SPEC must either adopt composition-local normalization, adopt an advisory,
or record why each is rejected.

---

## Defects Found (structured defect-list)

**D1. A checkbox answer item is corrupted by composition, and nothing in the
SPEC prevents it** — `spec.md:L131-134` (REQ-ML-001.2) /
`plan.md:L136-139` (DD-1) — Severity: **critical** — Class: **blocking**.

REQ-ML-001.2 admits "the unordered … list form", and an Org checkbox list is
an unordered list. The SPEC nowhere says what portion of a checkbox item is
wrapped, so the natural reading — wrap the item's own content, as for any
plain item — produces a corrupted marker. Measured against go-org v1.9.1:

```
IN : "- {{c1::[ ] A}}\n"
OUT: "<ul>\n<li class=\"unchecked\">::[ ] A}}</li>\n</ul>\n"
```

The opening `{{c1` is **destroyed**: go-org parses the checkbox `Status`
before the item content, consuming the marker's opening. The card ships with
a dangling `::[ ] A}}`. Wrapping the text after the checkbox is clean:

```
IN : "- [ ] {{c1::A}}\n"
OUT: "<ul>\n<li class=\"unchecked\">{{c1::A}}</li>\n</ul>\n"
```

This is the same hazard class DD-3 found and dispositioned for description
items, and it is undispositioned here. DD-1 (plan.md:L137) literally
enumerates `org.ListItem`'s fields as "`Bullet`, `Status`, `Value`,
`Children`" — the `Status` field is named and its consequence for wrapping is
never asked. The scanner/parse cross-check (plan.md:L149-155) cannot catch it:
item **counts** agree for a checkbox list, only the wrapped span differs.
Required fix: REQ-ML-001 gains a sub-clause stating that a status-bearing item
is wrapped on the content after its checkbox (mirroring REQ-ML-001.3's
definition-half rule), and `acceptance.md` gains an AC-ML-001 sub-criterion
and an Edge Cases row for `- [ ] A`.

**D2. AC-ML-006a is unsatisfiable: REQ-ML-007.1 must break the very marker
AC-ML-006a requires to survive** — `spec.md:L191-200` (REQ-ML-006.1) /
`spec.md:L202-213` (REQ-ML-007) / `acceptance.md:L156-162` (AC-ML-006a) —
Severity: **critical** — Class: **blocking**.

AC-ML-006a requires that for body `- Tokyo is {{c4::big}}` under direction
`->`, "the hand-written `{{c4::big}}` survives **byte-for-byte** inside the
generated marker's content". REQ-ML-007.1 requires that "a run of two or more
consecutive `}` characters inside wrapped content shall be separated by a
single space each". The hand-written marker's closing `}}` **is** a run of two
inside wrapped content. Running the repository's own rule confirms the
collision:

```
IN="Tokyo is {{c4::big}}"  OUT="Tokyo is {{c4::big} }"
```

(`imoogi-anki--cloze-safe-text`, `modules/org/24-anki.el:201-214`, executed
under `emacs -Q --batch`.) The marker no longer reads `}}` and is no longer a
valid Anki cloze. No implementation can satisfy both criteria.

Exempting the inner marker from the brace rule does not rescue it: Anki's
cloze pattern is non-greedy, so the outer `{{c5::` would close at the inner
`}}` — precisely the failure REQ-ML-007.2 exists to prevent, and REQ-ML-007.2
names "a hand-written marker — both of which REQ-ML-006 permits" as a
triggering case. The author saw the interaction and did not follow it through.

DD-5's premise (plan.md:L278-282) — "an author who clozed a word inside what
later becomes an answer item … keeps that card and gains the multiline ones" —
holds only when the hand-written marker sits **outside** the wrapped span: a
title marker under `->`, or the AC-ML-006c shape. When it sits **inside** a
wrapped answer under `->` or `<->`, the author's card is destroyed by the
chosen design, which is the outcome DD-5 rejected "silently renumbering" for.
Required fix: REQ-ML-006 gains an explicit rule for a hand-written marker
inside a span that would be wrapped — exclude that item from wrapping, or
reject the entry with a diagnostic — and AC-ML-006a is rewritten to assert
whichever rule is chosen.

**D3. A plain item inside a description-kind list has no specified answer
span** — `spec.md:L135-140` (REQ-ML-001.3) / `plan.md:L187-221` (DD-3) —
Severity: **major** — Class: **blocking**.

go-org's description-kind classification is per **list**, not per item: a
single `::` item makes every sibling descriptive, and a plain sibling is
emitted with a literal `?` as its term. Measured:

```
IN : "- term :: def\n- plain\n"
OUT: "<dl>\n<dt>\nterm\n</dt>\n<dd>def</dd>\n<dt>\n?\n</dt>\n<dd>plain</dd>\n</dl>\n"
```

REQ-ML-001.3 says "an item's answer shall be its **definition** half alone".
A plain sibling has no definition half, and the SPEC gives no rule. A
reasonable engineer could skip the item (losing an answer), or wrap it whole.
Wrapping it whole is measured clean —
`- term :: {{c1::def}}\n- {{c2::plain}}` → `<dd>{{c1::def}}</dd>` and
`<dd>{{c2::plain}}</dd>` — so the correct choice is available, but the SPEC
does not make it. Required fix: one sub-clause under REQ-ML-001.3 stating
that a description-kind item carrying no `::` is wrapped on its whole content,
plus an Edge Cases row.

**D4. DD-1's stated scanner grammar is not block-aware, and the cross-check
corpus is chosen by the same author** — `plan.md:L140-155` — Severity:
**major** — Class: **blocking**.

The mechanism is specified as `^(\s*)([+*-])(\s+(.*)|$)` plus the ordered
form, with no exclusion for block interiors. go-org does not read a bullet
line inside a block as a list item:

```
IN : "#+BEGIN_SRC text\n- not an item\n#+END_SRC\n- real\n"
OUT: "<div class=\"src src-text\">…<pre>\n- not an item\n</pre>…</div>\n<ul>\n<li>real</li>\n</ul>\n"

IN : "#+BEGIN_EXAMPLE\n- not an item\n#+END_EXAMPLE\n"
OUT: "<pre class=\"example\">\n- not an item\n</pre>\n"
```

The stated grammar would count `- not an item` as the **first** bullet and
take the answer list from inside a source block — a divergence the scanner
would not merely miscount but would silently wrap, injecting a cloze marker
into a code sample.

The guard is real but not sufficient. The count-equality property test is a
genuine falsifier for the shapes it covers: a scanner that counts a block
interior as an item disagrees with the parse, and the test fails. But the
corpus is chosen by the SPEC author, and the SPEC names no shapes it must
contain. **Shapes I would add, each mechanically confirmed to stress a
different axis:**

| Shape | Why it discriminates |
|---|---|
| `#+BEGIN_SRC text\n- x\n#+END_SRC\n- real\n` | Parser sees one item, the stated grammar sees two |
| `#+BEGIN_EXAMPLE\n- x\n#+END_EXAMPLE\n` | Parser sees **no** list at all; a naive scanner sees one and skips `multiline_answer_missing` |
| `#+BEGIN_QUOTE\n- inner\n#+END_QUOTE\n` | Parser sees a list, but nested inside a block — top-level-ness, not existence |
| `- A\n\n- B\n` vs `- A\n\n\n- B\n` | One blank line keeps one list, two split it. DD-4 depends on this and the scanner must implement it; the grammar as stated says nothing about blank lines |
| `- [ ] A\n- [X] B\n` | Counts agree, spans differ — the property test is blind by construction (D1) |
| `1) A\n2) B\n` | The `)` ordered form renders as `<ol>`; "the ordered form" is unspecified as to which terminators count |
| `  - A\n  - B\n` | Indented first bullet; base indentation is taken from the first bullet line, so this must not be read as nested |
| `- term :: def\n- plain\n` | Counts agree (2), kinds diverge per item (D3) |
| `- A\n+ B\n` | go-org merges `-` and `+` into one `<ul>`; real Org-mode treats a bullet change as a new list. The grammar happens to agree with go-org here — worth pinning so it stays deliberate |

Column-0 `* A` matches the stated grammar (`\s*` admits the empty indent) but
is parsed by go-org as a headline. It is unreachable through the Elisp body
extraction, which ends a heading's body at the next column-0 star, so it is
recorded for completeness rather than weighted. Required fix: DD-1's mechanism
paragraph states that the scanner skips block interiors and implements the
two-blank-line list break, and § F M1 names the corpus shapes above as
required members.

**D5. AC-ML-014's deciding command names a file that carries no pairing
assertion** — `acceptance.md:L372` — Severity: **major** — Class:
**blocking**.

AC-ML-014b asserts "the existing pairing assertion and its hard-coded back-end
code list" still hold, and names `tests/anki-sync-error-test.el` as the
deciding file. That file contains seven tests, none of them the pairing
assertion (`imoogi-sync-error-test-ac007-…` through
`imoogi-sync-error-test-keyed-error-…`, L40-196). The pairing assertion and
the hard-coded list both live in `tests/anki-error-test.el`:
`imoogi-error-test--go-emitted-codes` (L20-31),
`imoogi-error-test-go-codes-are-subset-of-table` (L51),
`imoogi-error-test-codes-match-source-file` (L57),
`imoogi-error-test-table-entries-are-all-paired-with-a-go-constant` (L101).

This is a regression against the predecessor, which got it right:
SPEC-ANKICARD-002's `acceptance.md:L510` names `tests/anki-error-test.el` for
the same criterion. Required fix: change `acceptance.md:L372` to
`tests/anki-error-test.el`. The aggregate `make test-elisp` would still run
the right test, so the defect is in the criterion's ability to decide, not in
the outcome.

**D6. REQ-ML-006.2 fixes the title at `c2` unconditionally, contradicting
REQ-ML-005.3 under incremental `<->`** — `spec.md:L198-200` /
`spec.md:L188-189` / `acceptance.md:L147-150` — Severity: **minor** —
Class: **blocking**.

REQ-ML-006.2: "Where no hand-written marker is present … the generated
numbers shall be exactly those the design record § 2.2 fixes: `c1` for the
answers and `c2` for the title." REQ-ML-005.3: "Where the direction is `<->`,
the title's number shall be distinct from every answer's." Under `<->` with
incremental **on** and two answers, REQ-ML-005.2 gives the answers `c1` and
`c2`, so REQ-ML-005.3 forces the title to `c3` — contradicting REQ-ML-006.2's
unconditional `c2`. AC-ML-005c avoids the collision by asserting only "a
number equal to neither", and AC-ML-006b pins `c1`/`c2` only for the
incremental-**off** case, so the contradiction is unpinned in either
direction.

Related, smaller: AC-ML-003b tests `<-` without pinning the title's number.
The design record § 2.2 fixes it at `c2` ("`<-`이면 제목을 `{{c2::…}}`") and
REQ-ML-006.2 agrees, so the SPEC is consistent here; only the criterion is
loose. Since numbering is a hash input and DD-5 notes the change cost is
"high after release", both deserve pinning. Required fix: REQ-ML-006.2 is
qualified to the incremental-off case, and AC-ML-003b pins the number.

**D7. AC-ML-013's named guard stops covering the production path the moment
REQ-ML-010 lands** — `acceptance.md:L330-355` / `spec.md:L243-254` —
Severity: **major** — Class: **blocking**.

REQ-ML-010 moves **both** production call sites (`planner.go:291`,
`migrate.go:208`) to the new option-taking entry point. After that, the only
callers of `orgdoc.Render` are `golden_baseline_test.go` and
`TestRenderSignatureIsUnchanged`. AC-ML-013's deciding command is
`go test ./internal/anki/orgdoc ./internal/anki/hashing` plus the `git diff`
— all of which exercise `Render`, a function no user code reaches. So
AC-ML-013 can pass in full while the new entry point's nil-option path
diverges from `Render` and every existing note re-hashes.

The regression would be caught, but by a criterion that does not claim to
catch it: `TestOptionFreeRequestLogGolden`
(`baseline_golden_test.go:194`) calls the real `Run(…)`, while its fixture
seeds the registry hash through `orgdoc.Render` (`:135`) — so a divergence
flips every no-op entry to `updated` and trips both
`len(noOp.WriteSequence) != 0` and the byte-comparison. That test is reached
only through AC-ML-015's `go test ./... -count=1`. REQ-ML-014 is the
constraint the SPEC says it is "shaped around" (spec.md:L424), and its own
criterion should carry the load-bearing guard. Required fix: AC-ML-013 gains
a sub-criterion asserting that the new entry point with no options set
produces output byte-identical to `Render` over every case in the existing
corpus, and names `TestOptionFreeRequestLogGolden` in its deciding command.

**D8. AC-ML-007c's shared fixture cannot be satisfied by the Elisp function
it names** — `acceptance.md:L189-198` / `spec.md:L208-213` — Severity:
**minor** — Class: **blocking**.

AC-ML-007c requires that "`imoogi-anki--cloze-safe-text` and the Go composer
produce the recorded form for every input". But that function performs
separation only; the trailing pad is applied by its **caller**,
`imoogi-anki-cloze-region` (`modules/org/24-anki.el:246-250`), as a separate
`pad` binding. Confirmed by execution: `imoogi-anki--cloze-safe-text` on
`"f(x}"` returns `"f(x}"` unchanged — no pad. REQ-ML-007.2 is precise about
this ("`imoogi-anki--cloze-safe-text` **and its pad**"); the criterion drops
the second half, so no single existing Elisp function produces the recorded
form and the ERT side of the shared fixture has nothing to call. Required fix:
AC-ML-007c names both the separator and the pad, or the SPEC requires
extracting a pad-applying Elisp helper the fixture can call.

**D9. REQ-ML-006 carries a `shall not` under an `[Ubiquitous]` label while
REQ-ML-014 labels the same form `[Ubiquitous — negated]`** —
`spec.md:L191` vs `spec.md:L305` — Severity: **minor** — Class:
**optional**.

"Generated numbers **shall not** collide with a hand-written marker" is the
negated form the SPEC itself distinguishes at L110-111. Both readings are
valid GEARS, so MP-2 is unaffected; the inconsistency is cosmetic. Required
fix: relabel REQ-ML-006 as `[Ubiquitous — negated]`, or drop the distinction.

**D10. The `#+BEGIN_EXTRA`-between-answers disposition rejects two fixes and
does not consider the one that survives both objections** —
`plan.md:L243-268` (DD-4) / `spec.md:L228-233` (REQ-ML-009.1) /
`acceptance.md:L229-242` (AC-ML-009d) — Severity: **major** — Class:
**blocking**.

Full reasoning under § 3b above. In brief: both rejected fixes are correctly
rejected, and DD-4's measurement reproduces exactly. But composition-local
gap normalization — collapsing the blank-line run inside the composition step,
which runs only for multiline entries — escapes the re-hash objection DD-4
used against normalizing in `splitExtraBlocks`, and is neither adopted nor
rejected. A non-skipping advisory diagnostic is likewise unconsidered, and
the mechanism exists (`tests/anki-sync-error-test.el:170`). As it stands, an
author who interleaves a note between answers silently loses every answer
after the note, with the only mitigation a README paragraph they must have
read in advance. Required fix: DD-4 evaluates composition-local normalization
and the advisory option explicitly, adopting one or recording why each is
rejected.

**D11. REQ-ML-002.1's stated rationale is false for the shape it most often
describes** — `spec.md:L150-153` — Severity: **minor** — Class:
**optional**.

"A leftward card with no answers hides the only content it has and leaves a
prompt with no visible cue." An entry with a lead paragraph and no list has
no answer item, so REQ-ML-002 fires — but the paragraph is visible, so the
card does have a cue. The requirement itself is unambiguous and binary; only
the justification overreaches. Required fix: narrow the rationale to the
empty-body case, or drop the sentence.

**D12. REQ-ML-012's container obligation is verified only under `->`** —
`spec.md:L266-277` / `acceptance.md:L277-284` — Severity: **minor** —
Class: **optional**.

REQ-ML-012.1 requires the `children-list` container unconditionally for a
multiline entry, but AC-ML-011a builds all three fixtures with direction
`->`. Under `<-` no answer is wrapped, and an implementation that attaches the
container as part of the answer-wrapping step would silently omit it. Required
fix: one AC-ML-011 sub-criterion under `<-`.

---

## Could Not Verify

- **Anki's own rendering.** Whether Anki accepts `<sup>`, `<br>`, or a marker
  spanning `</p><ul><li>` inside a cloze deletion. DD-2 is explicit that this
  is unverified and treats it as a reason to be conservative, which is the
  right handling. I ran no Anki instance and make no claim about it. D1 and
  D2 above are Org-level and go-org-level, and do not depend on Anki's
  behavior.
- **Whether the property test will be written to the corpus D4 names.** M1
  (plan.md:L414-422) describes the cross-check but the corpus is not
  enumerated. The guard's adequacy is unverifiable at plan time by
  construction; D4's required fix is to make it verifiable.
- **The go-org superscript interaction with AC-ML-007a.** `a^{2}` renders to
  `a<sup>2</sup>`, so some source braces vanish before the rendered assertion
  runs. I confirmed the padded and unpadded forms still differ in rendered
  output (`} } }}` vs `}}}`), so AC-ML-007a **can** fail and is not vacuous.
  Whether an implementer scanning for "its own closing `}}`" resolves the
  self-reference the same way I did is not something a plan-phase audit can
  settle; no defect is filed.
- **The elisp criteria AC-ML-012a-e.** Judged by reading only. The
  `imoogi-anki-cloze-region` note-type contract they mirror exists
  (`modules/org/24-anki.el:216-250`) and DD-7's description of it matches, but
  I did not execute the transient or the property-writing path.

---

## Recommendation

FAIL at iteration 1/3. This is a strong SPEC with an unusually honest
evidence base — every one of the twelve go-org claims reproduces byte-for-byte,
the pipeline-ordering claim is mechanically true, and § 8 correctly mirrors the
predecessor without reintroducing the gate its audit removed. The defects are
concentrated in shapes the author's own probe set did not reach, and in
criteria that name the wrong target.

Fix in this order. The first two are the ones that make the SPEC
unimplementable as written.

1. **D2 (critical)** — resolve the AC-ML-006a / REQ-ML-007.1 contradiction.
   Decide what happens when a hand-written marker sits inside a span that
   would be wrapped: exclude the item from wrapping, or reject the entry with
   a diagnostic. Rewrite AC-ML-006a to assert the chosen rule.
   `spec.md:L191-213`, `acceptance.md:L156-162`.
2. **D1 (critical)** — add a checkbox sub-clause to REQ-ML-001 (wrap the
   content after the status marker), plus an AC-ML-001 sub-criterion and an
   Edge Cases row. `spec.md:L131-134`, `acceptance.md:L39-71, L398`.
3. **D10 (major)** — evaluate composition-local gap normalization and the
   advisory option in DD-4; adopt one or record why each is rejected.
   `plan.md:L243-268`.
4. **D4 (major)** — make DD-1's scanner block-aware in the stated grammar,
   state the two-blank-line list break, and name the nine corpus shapes above
   as required members of the M1 property test. `plan.md:L140-155, L414-422`.
5. **D7 (major)** — give AC-ML-013 a sub-criterion asserting
   new-entry-point-with-no-options == `Render` over the existing corpus, and
   name `TestOptionFreeRequestLogGolden` in its deciding command.
   `acceptance.md:L330-355`.
6. **D3 (major)** — specify the answer span for a plain item inside a
   description-kind list. `spec.md:L135-140`.
7. **D5 (major)** — correct AC-ML-014's deciding command to
   `tests/anki-error-test.el`. `acceptance.md:L372`.
8. **D6 (minor)** — qualify REQ-ML-006.2 to the incremental-off case; pin the
   title's number in AC-ML-003b. `spec.md:L198-200`, `acceptance.md:L110-113`.
9. **D8 (minor)** — name the pad alongside `imoogi-anki--cloze-safe-text` in
   AC-ML-007c, or require a pad-applying helper. `acceptance.md:L189-198`.
10. **D12 (minor)** — add an AC-ML-011 sub-criterion under `<-`.
    `acceptance.md:L277-284`.

D9 and D11 are **optional** — surfaced for the orchestrator's discretion, not
routed into the revision. Do not let the length of this list manufacture a
scope expansion: ten blocking findings, each a bounded edit to an existing
requirement or criterion. No requirement needs to be added and no design
decision needs to be reversed. The spine decision — DD-1's pre-render,
source-level composition — is **correct and confirmed**, and nothing here
asks for it to be revisited.

Re-audit at iteration 2 will be scoped to this enumerated defect delta plus a
regression check over it, not a from-scratch full re-audit.
