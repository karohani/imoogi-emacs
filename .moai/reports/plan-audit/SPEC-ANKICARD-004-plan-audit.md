# SPEC Review Report: SPEC-ANKICARD-004

Iteration: 1/3
Verdict: **FAIL**
Overall Score: **0.75** (harmonic mean; Tier M PASS threshold is 0.80)

Reasoning context ignored per M1 Context Isolation. The audit read
`spec.md`, `plan.md`, and `acceptance.md` (the Tier M artifact set) plus the
code surfaces they name. MP-2 was judged against the `REQ-XXX` requirement
layer in `spec.md` only; the Given-When-Then entries in `acceptance.md` are the
verification layer and were graded under Group 4.

All measurements below were taken in a scratch Go module outside the checkout
(`/private/tmp/.../scratchpad/sw4`) against `go-org v1.9.1` from the module
cache, at repository HEAD `08f7504`. No git command that writes was run.

---

## Must-Pass Results

- **[PASS] MP-1 REQ number consistency.** `grep -o 'REQ-SW-[0-9]\{3\}' spec.md | sort -u`
  yields `REQ-SW-001 … REQ-SW-014`, contiguous, consistently zero-padded.
  `grep -c '^#### REQ-SW-'` = 14; `sort | uniq -d` over the headings prints
  nothing, so there is no duplicate.
- **[PASS] MP-2 GEARS format compliance** (requirement layer only). All 14
  `REQ-SW-NNN` headings carry exactly one pattern tag: `[Where]` ×2
  (REQ-SW-001, REQ-SW-002), `[Ubiquitous]` ×10, `[Ubiquitous — negated]` ×2
  (REQ-SW-005 "Composition shall not disturb…", REQ-SW-012 "The back end shall
  not change…"). Each body sentence matches its tag, e.g. spec.md:132
  "Where an entry is a swift entry, the back end shall take its cards from the
  **arrow lines** of the remaining body". No Given-When-Then entry appears in
  the requirement layer.
- **[PASS] MP-3 YAML frontmatter validity.** All 12 canonical fields present
  with correct types at spec.md:2-15 — `id`, `title`, `version: "0.1.0"`
  (quoted), `status: draft` (valid enum), `created`/`updated: 2026-09-21`
  (ISO), `author`, `priority: P2`, `phase`, `module`, `lifecycle:
  spec-anchored`, `tags` (comma-separated string). No rejected snake_case alias
  (`created_at` / `updated_at` / `labels` / `spec_id`) is present. Optional
  `tier: M` and `depends_on` also present.
- **[N/A] MP-4 language neutrality.** Single-project SPEC scoped to this
  repository's Go packages and its Emacs Lisp modules. It is not
  template-bound or universal content, so the 16-language enumeration rule does
  not apply.
- **[PASS] MP-5 D7 cross-SPEC reconciliation.** The four `SPEC-…` references
  extracted from the body all resolve: SPEC-ANKICARD-001 / -002 / -003 all read
  `status: in-progress`; none is `retired`, `superseded`, or `archived`, so no
  reconciliation obligation fires. The SPEC-ANKICARD-003 `in-progress`
  dependency is disclosed with its authorization at spec.md:594-604 rather than
  left implicit. No BLOCKING finding.
- **[PASS] MP-6 D8 cross-platform discipline.** `grep -c 'syscall' spec.md` = 0.
  Auto-PASS per D8-4.
- **[N/A] MP-7 clarification gate.** `grep -rn '\[NEEDS CLARIFICATION'` over the
  SPEC directory returns exit 1 (no match). `research.md` does not exist —
  correct for Tier M — so the criterion is N/A on that artifact and PASS on
  `plan.md`.

No must-pass criterion fails. **The FAIL verdict rests on D1, D2, and D12 as
blocking findings and on the aggregate score falling below the Tier M
threshold, not on the firewall.**

---

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.50 | 0.50 — "Multiple requirements require interpretation. A reasonable engineer might implement them differently than intended." | Three independent divergences where two conformant implementations differ: D2 (spec.md:141 regex vs spec.md:155 "only the first token"), D1 (spec.md:222 head clause vs :225 per-side rule), D12 (spec.md:172 line-local predicate vs go-org list-continuation semantics). |
| Completeness | 1.00 | 1.0 | HISTORY (:18), Purpose/WHY (:40), Scope/WHAT (:52), Glossary (:70), Requirements (:104), Out of Scope (:462, six `### Out of Scope — <topic>` H3 headings each carrying specific `-` bullets), Constraints (:531), Brownfield Delta (:551), Dependencies (:592), Traceability (:610). Acceptance criteria in `acceptance.md` per the Tier M artifact set. Frontmatter complete. |
| Testability | 0.75 | 0.75 — "One AC is not precisely binary-testable but is measurable with minor interpretation." | No weasel word appears (`grep -nio 'appropriate\|adequate\|reasonable\|properly\|as needed\|if necessary\|sufficiently' acceptance.md` → none). Every criterion names a deciding command. AC-SW-007 (acceptance.md:183-184) is the exception: it claims the snippet bound is "mechanically checked rather than reviewed", but the deciding step — "Any hit whose snippet content is a variable rather than a constant fails this criterion" — requires reading the grep output. Three measured shapes (D1, D3, D12) have no corpus row. |
| Traceability | 1.00 | 1.0 | All 14 REQs are covered: AC-SW-001→REQ-SW-001, 002→002+010, 003→003, 004→004, 005→005, 006→006, 007→007, 008→008, 009→009, 010→010, 011→011, 012→012.1-.3, 013→012.4-.5, 014→013, 015→014. Every AC names its requirements; every named requirement exists. No orphan, no uncovered REQ. |

Harmonic mean: `4 / (1/0.50 + 1/1.00 + 1/0.75 + 1/1.00)` = `4 / 5.333` = **0.75**.
Tier M threshold (`spec-workflow.md` § SPEC Complexity Tier, line 330) is
**0.80**. 0.75 < 0.80.

Tier budget: 14 requirements and 15 criteria, both within the Tier M ceiling of
16 each.

---

## Defects Found (structured defect-list)

### D1 — A hand-written marker spanning the arrow token defeats REQ-SW-005 and corrupts the card

`spec.md`:L222-232 — Severity: **critical** — Class: **blocking**

REQ-SW-005's head clause is absolute: "Composition shall not disturb a
hand-written marker already present in the title or the remaining body."
Sub-clause 1 delivers a *per-side* rule instead: "A side that already carries a
hand-written marker shall not be wrapped." A marker that **spans** the arrow
token satisfies neither side's guard, because each half carries only a
fragment of it.

Measured, composing REQ-SW-001/003/004/005/006 exactly as written over the line
`{{c1::도쿄 :-> 일본}}`:

```
LINE      "{{c1::도쿄 :-> 일본}}"
  base    = 1                       (highestClozeNumber sees c1)
  left    = "{{c1::도쿄"             hasClozeMarker → true  → not wrapped
  right   = "일본}}"                 hasClozeMarker → FALSE → wrapped at c2
  COMPOSED "{{c1::도쿄 @@html:<b>:-&gt;</b>@@ {{c2::일본} } }}"
  RENDER   "<p class=\"swift-list\">{{c1::도쿄 <b>:-&gt;</b> {{c2::일본} } }}</p>"
```

Applying Anki's own non-greedy cloze close to that rendered field yields **one**
card, not two:

```
  anki card c1 := "도쿄 <b>:-&gt;</b> {{c2::일본} } "
```

The author's single `c1` deletion is destroyed and the literal text
`{{c2::일본} }` shows on the card. The `:<->` form behaves identically
(`{{c3::도쿄 :<-> 일본}}` → one card `c3` carrying `{{c4::…}}` as visible text).

Reachability is high: `imoogi-anki-cloze-region` writes exactly this shape when
the user selects a whole line, and `ANKI_SWIFT` inherits, so a file-level
property reaches headings marked before the option existed.

This is also a **regression**, and `acceptance.md`'s Residual Risk 4 misstates
it: that entry says a swift heading with an arrow line "has its `Text` field
rewritten with markers, emphasis, and the container, so its hash changes and it
is reported **updated** once — correctly, since its card genuinely changed."
For this shape the card is corrupted, not changed correctly.

**Required fix:** add a sub-clause to REQ-SW-005 making a line whose arrow token
sits *inside* a hand-written marker's span not an arrow line (or, equivalently,
forbidding composition on such a line), and pin it with a corpus row
`{{c1::도쿄 :-> 일본}}` → 0 arrow lines plus an AC-SW-005 clause. Detecting the
span needs a marker-extent scan rather than the existing `hasClozeMarker`
prefix predicate; say which, since `hasClozeMarker` alone cannot decide it.

### D2 — REQ-SW-001.1's regex contradicts REQ-SW-001.3's "first token" rule

`spec.md`:L141 and L155-160 — Severity: **major** — Class: **blocking**

REQ-SW-001.3 states as an obligation: "Only the **first** arrow token on a line
shall be read. A second token is part of the right side and shall render as
literal escaped text, carrying no emphasis and earning no second pair of
numbers", and attributes the outcome to the expression: "The expression's
non-greedy left group is what produces this."

Non-greediness minimises the *left* side; it does not pin the *first* token.
When the text before the first token is empty, `(.+?)` must grow across that
token to find a later one. Measured against
`(.+?)\s*(:<->|:->|:<-)\s*(.+)` in Go's `regexp`:

```
  ":-> B :-> C"   -> tok=":->"  L=":-> B"   R="C"
  ":<- B :-> C"   -> tok=":->"  L=":<- B"   R="C"
  "A :-> B :-> C" -> tok=":->"  L="A"       R="B :-> C"   (as REQ-SW-001.3 says)
```

On the first two lines the first token is **not** read as the arrow, is **not**
part of the right side, and lands in the left side; the **second** token becomes
the arrow and is the one emphasized. Both sentences of REQ-SW-001.3 are false
for that input class.

An implementer following REQ-SW-001.1 literally and an implementer following
REQ-SW-001.3 literally produce different code and different cards. Corpus row 8
(`:-> B` → 0 arrow lines) pins the adjacent case but not this one.

**Required fix:** pick one rule and make the other follow it. The cheapest
consistent form is a sub-clause rejecting a line whose left side still contains
an arrow token — which also keeps the shape consistent with row 8's rejection —
plus a corpus row `:-> B :-> C`.

### D12 — The container attribute line restructures a list when the arrow sits on an indented continuation

`spec.md`:L172-180 (REQ-SW-001.6) with L399-410 (REQ-SW-011.1) — Severity: **major** — Class: **blocking**

REQ-SW-001.6 excludes "a line the **bullet grammar** reads as a list item".
`readBullet` (`internal/anki/orgdoc/multiline.go:283`) is **line-local**: it
matches a bullet character at the head of the line. go-org's list membership is
not line-local — an indented continuation line is list content while carrying no
bullet. So a continuation line carrying an arrow passes REQ-SW-001.6's exclusion
and becomes an arrow line; REQ-SW-011.1 then prepends `#+ATTR_HTML:` at column
zero immediately above it, which **terminates the list**.

Measured:

```
body      "- 수도\n  도쿄 :-> 일본\n"
  line "- 수도"          bullet=true   arrow=false
  line "  도쿄 :-> 일본"  bullet=false  arrow=true      <- not excluded
  BEFORE  "<ul>\n<li>수도\n도쿄 :-&gt; 일본</li>\n</ul>\n"
  AFTER   "<ul>\n<li>수도</li>\n</ul>\n<p class=\"swift-list\">도쿄 <b>:-&gt;</b> {{c1::일본}}</p>\n"
```

The continuation is pulled out of its `<li>` and becomes a sibling paragraph.
The same holds under an ordered item (`1. 수도` / `   도쿄 :-> 일본`) and at
deeper indents. A control run without the attribute line shows the damage is
caused specifically by REQ-SW-011.1's placement, not by the markers:

```
  control "- 수도\n  도쿄 @@html:…@@ {{c1::일본}}\n"
       -> "<ul>\n<li>수도\n도쿄 <b>:-&gt;</b> {{c1::일본}}</li>\n</ul>\n"   (list intact)
```

This violates REQ-SW-001.7 — "Content that is not an arrow line — a lead
paragraph, a **list**, … — shall render as it does today, in document order" —
and it is a *structural* change to authored content, not a styling difference.
It is materially worse than the list-item silence § 5 discloses, and it is
undisclosed.

**Required fix:** state what the container attribute line may be inserted above.
Two options, either sufficient: (a) extend REQ-SW-001.6's exclusion from
"a line the bullet grammar reads as a list item" to "a line that is list
content", deriving membership from the continuation rule `scanAnswerList`
already implements; or (b) constrain REQ-SW-011.1 so no attribute line is
emitted where it would interrupt a list or any other multi-line construct, and
say what happens to an arrow line that therefore gets no container. Pin the
chosen behaviour with a corpus row `- 수도` ⏎ `  도쿄 :-> 일본`.

### D3 — A run's container has no stated end, and swallows trailing non-arrow lines

`spec.md`:L399-410 — Severity: **minor** — Class: **blocking**

REQ-SW-011.1 fixes where a container *opens* ("immediately above the first arrow
line of each run") and never says where it closes. Measured, the paragraph runs
to the next blank line or the next attribute line, so a non-arrow, non-blank
line following an arrow line is drawn inside the styled container:

```
IN   "#+ATTR_HTML: :class swift-list\nA :-> B\ncontext line\n#+ATTR_HTML: :class swift-list\nC :-> D\n"
OUT  "<p class=\"swift-list\">A :-&gt; B\ncontext line</p>\n<p class=\"swift-list\">C :-&gt; D</p>\n"
```

`context line` is not an arrow line, and REQ-SW-001.7 promises it renders "as it
does today" — today it sits in an unclassed paragraph; after this SPEC it
inherits whatever REQ-SW-011.2's rule does. Corpus row 14 covers prose *above*
an arrow line (the attribute line correctly ends it); no row covers prose
immediately *below* one, and it is the symmetric case.

**Required fix:** one sub-clause naming what closes a run, plus a corpus row
`도쿄 :-> 일본` ⏎ `참고: …` with the expected rendering stated.

### D5 — The prescribed reuse of `imoogi-anki--write-card-option` breaks REQ-SW-013.2's OFF path

`spec.md`:L440-444 (REQ-SW-013.3) with `plan.md`:L26 — Severity: **major** — Class: **blocking**

REQ-SW-013.3 requires the new command to reach the note-type contract "through
their shared implementation rather than a second copy", and `plan.md` § A.1
names `imoogi-anki--write-card-option` and `imoogi-anki--ensure-cloze-type` as
"Reused by the new command."

`imoogi-anki--write-card-option` (`modules/org/24-anki.el:352-359`)
unconditionally runs `imoogi-anki--resolve-swift-conflict` **before** the
note-type check:

```elisp
(when (and (imoogi-anki--resolve-swift-conflict)
           (imoogi-anki--ensure-cloze-type))
  (org-set-property property value) t)
```

`imoogi-anki--resolve-swift-conflict` (:328-349) resolves `ANKI_SWIFT` with
inheritance and, when it is on, prompts "swift 가 켜져 있습니다 … swift 를 끄고
계속할까요?" and returns nil on decline. For the swift toggle itself this is
backwards on exactly the path REQ-SW-013.2 specifies: turning the option **off**
happens precisely when the resolved value is on, so every OFF fires a prompt
asking whether to turn swift off before turning swift off, and a declining user
cannot turn it off at all. `imoogi-anki--ensure-cloze-type`'s own failure
message also reads "멀티라인 옵션을 달 수 없습니다", which is wrong for a swift
heading.

REQ-SW-013.4's mirror requirement compounds it: the swift command needs the
*multiline*-conflict check, which this helper does not provide.

**Required fix:** say which helper is reused. Either factor the note-type
contract out of `imoogi-anki--write-card-option` so the swift command reuses
`imoogi-anki--ensure-cloze-type` plus its own mirror check, or state that
`imoogi-anki--write-card-option` gains a conflict-check parameter. Also correct
the note-type failure message so it does not name multiline options, and add an
ERT case for the OFF path that does **not** stub `yes-or-no-p` to always-yes —
otherwise AC-SW-014's inheritance clause passes while the prompt still fires.

### D9 — REQ-SW-013.4's mirror check is singular where two options can be on

`spec.md`:L445-450 — Severity: **minor** — Class: **blocking**

"Where the heading resolves a multiline option to an on value, the command shall
report the conflict and shall write nothing unless the user confirms clearing
**it**." Both `ANKI_DIRECTION` and `ANKI_INCREMENTAL` can be on at once — row
`swift with both` in `TestSwiftWithMultilineOptionIsRejected`
(`internal/anki/planner/card_options_test.go:197`) is exactly that entry. The
existing `imoogi-anki--resolve-swift-conflict` clears one property because it
has one to clear; the mirror has two.

**Required fix:** state whether confirmation clears every on multiline option or
only the one reported, and which value is written to each.

### D4 — REQ-SW-012.4's sufficiency claim for the hash-probe pair is not established

`spec.md`:L423-433 and `plan.md`:L300-320 (DD-5) — Severity: **minor** — Class: **blocking**

REQ-SW-012.4: "Neither half states the contract alone: the first would permit an
implementation that hashed an on option directly, and the second would permit
one that hashed the option field itself." That is true, and it implies the pair
together *does* state the contract. It does not.

Trace an implementation that adds the **resolved boolean** to the hash inputs
while ignoring the raw wire field:

- Half 1 (all three fields falsy vs all absent → both resolve off) → same hash.
  **Passes.**
- Half 2 (swift absent vs swift on over an arrow body) → different hash.
  **Passes.**

So the pair admits the very implementation REQ-SW-012.2 forbids ("The content
hash's input set shall gain no member"). What actually excludes it is
`TestHashCallSiteTakesFourArguments`
(`internal/anki/planner/baseline_golden_test.go:304`), which `acceptance.md`
AC-SW-012 cites and `plan.md` § A.3 preserves — but which REQ-SW-012.4 and DD-5
never name at the point where they make the sufficiency claim.

**Judgment on the hash-probe pair (requested deliverable):** the reshaping is
the right move and the two halves are individually correct and individually
necessary. They are not jointly sufficient. The contract is stated by a
**trio** — falsy-vs-absent identity, on-vs-absent difference, and the
four-argument compile-time assertion — and the SPEC should say so.

**Required fix:** name the four-argument compile assertion as the third leg in
REQ-SW-012.4 and in DD-5, and replace "Neither half states the contract alone"
with a statement of what the three together exclude. Also carry the third leg
into the helper docstring REQ-SW-012.4 already requires.

### D6 — The composition corpus is miscounted

`acceptance.md`:L16 — Severity: **minor** — Class: optional

"Sixteen rows" precedes a table of seventeen
(`awk '/^\| # \| Body/,/^## AC Matrix/' acceptance.md | grep -c '^| [0-9]'` → 17;
rows are numbered 1 through 17). Row 17 (`도쿄 :-> 일본` ⏎⏎ `- 참고 사항`) is the
one the count omits, and it is a load-bearing row — AC-SW-001f decides on it.

**Required fix:** change "Sixteen rows" to "Seventeen rows".

### D7 — `plan.md` § A.2 mis-cites the non-regression criterion

`plan.md`:L35 — Severity: **minor** — Class: optional

"`acceptance.md` AC-SW-015 is the non-regression criterion over this baseline."
AC-SW-015 (`acceptance.md`:360) is "The new code is paired with a message, both
ways (REQ-SW-014)". The non-regression content over the `08f7504` baseline is
the **Quality Gate Criteria** table (`acceptance.md`:434-444).

**Required fix:** point § A.2 at the Quality Gate Criteria table.

### D8 — `spec.md` Constraint 6 overstates the raw-HTML surface

`spec.md`:L547-549 — Severity: **minor** — Class: optional

"**Raw HTML reaches the card from exactly one construct.**" Measured, an export
snippet an *author* writes in a body already reaches the card as raw HTML, and
has since before this SPEC:

```
IN   "Tokyo @@html:<b>:-></b>@@ Japan\n"   OUT "<p>Tokyo <b>:-></b> Japan</p>"
```

REQ-SW-007.3 is correctly scoped — it binds what *composition* places inside a
snippet, and AC-SW-007's grep checks that bound. Constraint 6 as worded claims a
property of the whole card surface that does not hold.

**Note on the injection question:** the bound is specified rather than assumed,
and it is specified at the right strength. REQ-SW-007.3 makes it narrow and
absolute; DD-2 names both failure modes (verbatim pass-through, and an
unrecognized backend deleting content silently — measured:
`@@latex:<b>x</b>@@` → `<p>Tokyo  Japan</p>`); § G anti-pattern 3 repeats it;
AC-SW-007 greps for it. The emitted snippet is a constant from a three-element
table, never assembled from line text. No new injection surface is introduced.

**Required fix:** restate Constraint 6 as a bound on composition
("Composition places raw HTML in exactly one construct"), not on the card.

### D10 — REQ-SW-001.4 does not fix which whitespace "trimmed" means

`spec.md`:L162-168 — Severity: **minor** — Class: optional

Go's `\s` inside the regex is ASCII-only; `strings.TrimSpace` is Unicode-aware.
They disagree on U+00A0, which a Korean IME can produce. Measured with
`TrimSpace`, `"A :-> "` yields an empty right side and is correctly
rejected; an ASCII-only trim would admit it and compose a marker around a
non-breaking space.

**Required fix:** name the trim as Unicode whitespace, since the project's notes
are Korean and the SPEC already argues from multi-byte content elsewhere.

### D11 — AC-SW-007's bound check is not fully mechanical

`acceptance.md`:L183-184 — Severity: **minor** — Class: optional

"so REQ-SW-007.3's bound is mechanically checked rather than reviewed" followed
by "Any hit whose snippet content is a variable rather than a constant fails
this criterion". The grep is mechanical; deciding variable-vs-constant over its
output is a read.

**Required fix:** either drop the "rather than reviewed" claim, or make the
check decide itself — e.g. a test asserting the arrow-token table is the only
producer of an `@@html:` string and that its three values are the three tokens.

---

## Explicit judgment on the list-item exclusion and its residual silence (requested)

**Disposition: ACCEPTED as a trade, with one wording correction.**

The residual behaviour § 5 discloses — a body mixing plain arrow lines with
list-item arrow lines takes only the plain ones — is materially **not** the
failure class the predecessor's overturned disposition belonged to. There, the
answers after a dangling marker left the card entirely, invisible to the
reviewer. Here REQ-SW-001.7 keeps the list-item arrow line **on the card**, in
document order, rendered as an ordinary list item; AC-SW-001f and corpus row 17
pin exactly that. The content is not dropped. What is withheld is the cloze
pair, and the line itself is visible on the card for the user to notice.

The author's justification for not reporting it is also sound and measured. The
sharpest composition shape is real:

```
IN   "- Tokyo :: Japan :-> x\n"
OUT  "<dl>\n<dt>\nTokyo\n</dt>\n<dd>Japan :-&gt; x</dd>\n</dl>\n"
```

The description term's `::` and the arrow claim the same line, and a new
diagnostic code for "some arrows were not read" would be describing a shape
whose correct rendering is genuinely undecided.

**A third option exists and is worth naming, but is not a required fix.** The
ambiguity the author cites comes entirely from the three consumptions
`answerContentOffset` (`multiline.go:319`) performs — the counter cookie, the
checkbox status, and the description term. For a plain item such as corpus row
10 (`- 도쿄 :-> 일본`) that function returns 0 and `readBullet`'s `contentStart`
gives an unambiguous content offset. Swift could therefore admit list items
where `answerContentOffset` is 0 and exclude only those where it is not,
reusing the two functions the plan already reuses and adding no second grammar.
That would remove the silence for the common shape while keeping the
undecidable one excluded. It also carries costs the author would have to weigh
— an arrow line inside an `<li>` breaks the "run of consecutive arrow lines"
container model of REQ-SW-011.1, which is precisely the interaction D12 shows is
already fragile. I record it as an option the orchestrator may put to the
author; I do not require it.

**What I do require here is a wording correction.** `spec.md`:L472-474 says a
mixed body "takes only the plain ones and drops the rest with no signal, which
is not [visible]". Nothing is dropped — the list-item arrow renders on the card
untouched. Describing it as a drop overstates the harm and invites a future
reader to repair a loss that is not occurring. Restate it as "carries the
list-item arrow as ordinary content rather than as a card, with no diagnostic".
Severity minor, class optional.

---

## Could not verify

Recorded as unverified rather than assumed either way.

1. **Anki's actual card generation over non-contiguous cloze numbers.** The
   positional rule leaves gaps by design (`:->` lines yield `c1`, `c3`, `c5`; a
   lone `:<-` line yields `c2` with no `c1` at all). `acceptance.md` Residual
   Risk 1 names the odd-gap case but not the no-`c1` case. Deciding either needs
   a running collection; AnkiConnect is not reachable here. The SPEC's
   disposition — record, do not check — is the right one.
2. **`white-space`-style rendering of the `swift-list` rule inside Anki's
   webview.** REQ-SW-011.2 fixes the obligation but not the declaration, and the
   rule does not exist yet. I verified the precedent it mirrors
   (`.card .children-list` in `internal/anki/model/assets/base.css`, section 9,
   naming no network resource and deriving nothing from a deck name) but not the
   new rule's rendered effect.
3. **`<b>` inside or adjacent to a cloze deletion on a real card.** The emphasis
   sits between two markers, which is the safer arrangement, but multi-element
   HTML on an Anki cloze card is unverified here — as `acceptance.md` Residual
   Risk 3 already states.
4. **`make lint`, `make fmt-check`, `make ci-local`, `make test-elisp`.** Not
   run. Running the Elisp suite would load this checkout's modified modules,
   which belong to another actor. I verified only `go test ./internal/anki/...
   -count=1`, which passed on all eight packages at HEAD `08f7504`. The rest of
   `plan.md` § A.2's baseline is the delegating session's measurement, carried
   forward unverified.
5. **Whether `plan.md` § A.4's fix list is complete beyond the planner
   package.** I enumerated every Go reference to the `Swift` field
   (`grep -rn "Swift\|swift" --include="*.go" internal/`) and confirmed the
   three named breaking sites are the only ones that set swift truthy over an
   arrow-free body **and expect acceptance** — every other truthy-swift site
   (`card_options_test.go:224, 227, 274, 301, 309, 330, 398, 424`) expects a
   gate rejection and is unaffected. The fourth site A.4 names as
   under-covering, `TestMultilineFalsyOptionsAreByteIdenticalToNoOptions`, is
   correctly characterised. I did **not** enumerate the Elisp test suite for
   equivalent sites.

**Mechanical confirmations vs reasoning.** D1, D2, D3, D12, D6, D8, D10, and the
D5 control flow were established by running code or reading source, and each
carries its verbatim output above. D4, D9, D11, and the item-3 judgment are
reasoning over the documents and the measured facts; they are labelled as such.
Every `plan.md` measurement I checked reproduced exactly — DD-1's three escape
forms, DD-2's five snippet forms including the `@@latex:` deletion and the
case-insensitive backend name, DD-3's three attribute-line forms, DD-4's
description-item and three block forms, and DD-7's blank-run form. The author's
probe work is sound; the defects are in the rules built on top of it.

---

## Recommendation

Verdict is FAIL. Six findings are blocking; five are optional and left to the
orchestrator's discretion. Fix in this order — the first three are the ones that
change what gets built.

1. **D1** (`spec.md` REQ-SW-005) — add the spanning-marker exclusion and its
   corpus row. This is the only finding that corrupts a card, and it is a
   regression for existing `ANKI_SWIFT` headings. Correct Residual Risk 4 in the
   same pass.
2. **D12** (`spec.md` REQ-SW-001.6 with REQ-SW-011.1) — decide between widening
   the exclusion to list *content* and constraining where the container
   attribute line may be inserted. Pin the choice with a corpus row.
3. **D2** (`spec.md` REQ-SW-001.1 / .3) — make the regex and the "first token"
   rule agree, and add the `:-> B :-> C` corpus row.
4. **D5** (`spec.md` REQ-SW-013.3, `plan.md` § A.1) — name the helper actually
   reused, and require an OFF-path ERT case that does not stub the prompt.
5. **D3** (`spec.md` REQ-SW-011.1) — state what ends a run's container; add the
   prose-below corpus row.
6. **D9** (`spec.md` REQ-SW-013.4) — say what the mirror check clears when both
   multiline options are on.
7. **D4** (`spec.md` REQ-SW-012.4, `plan.md` DD-5) — name the four-argument
   compile assertion as the third leg of the hash contract.

Optional, at the orchestrator's discretion: D6 (corpus count), D7 (`plan.md`
§ A.2 cross-reference), D8 (Constraint 6 wording), D10 (trim definition), D11
(AC-SW-007's mechanicity claim), and the § 5 "drops … with no signal" wording
correction recorded above.

The document is strong where it is strong: every go-org measurement reproduces,
the traceability is complete, the reuse inventory in `plan.md` § A.1 is accurate
against the tree, the PRESERVE list correctly names the other actor's files, and
§ A.4's amendable-test analysis is right including the fourth site the brief did
not have. The defects cluster in one place — rules layered on top of correct
measurements, where a predicate's scope (a side, a line, a run) is narrower than
the obligation stated above it.
