# 구현 계획 — SPEC-ANKICARD-005

되돌리기 어려운 결정부터 적었다. § B는 설계 결정이고, 그중 DD-1–DD-3이 가장
바뀔 가능성이 높다. § F의 마일스톤은 기계적인 작업을 뒤에 둔다. 검토 시간이
짧다면 § B의 DD-2와 DD-3, 그리고 DD-6의 테스트 처분표를 먼저 보면 된다.

## § A 맥락

### A.1 지금 있는 것 (기준선 `c010009`)

| 표면 | 위치 | 이 SPEC에서의 쓰임 |
|---|---|---|
| 분리 패턴 `extraBlockPattern` | `internal/anki/orgdoc/extra.go:17-18` | 판정 문법의 원본. 짝짓기는 그대로 둔다. |
| `splitExtraBlocks` | `internal/anki/orgdoc/extra.go:31-46` | 시그니처·반환값 불변. 판정은 형제 함수로 붙는다. |
| Cloze 일반 렌더의 분리 → 표식 게이트 | `internal/anki/orgdoc/orgdoc.go:73-76` | 분리와 `ClozeMarkerMissingError` 사이에 판정을 넣는다. |
| multiline 렌더의 분리 → 구성 | `internal/anki/orgdoc/multiline.go:602-608` | 분리와 `composeMultiline` 사이에 판정을 넣는다. |
| `composeMultiline` 문서 주석의 "보고하지 않음" 결정 | `internal/anki/orgdoc/multiline.go:476-503` | 이 SPEC이 뒤집는다. 주석을 다시 쓴다. |
| `renderError` (skip/fail 매핑, 두 경로 공유) | `internal/anki/planner/multiline.go:43-63` | 오류 타입 하나를 skip으로 추가. |
| 일반 경로의 렌더 호출 | `internal/anki/planner/planner.go:295-306` | 변경 없음. 매핑으로 자동 처리. |
| 마이그레이션 경로의 렌더 호출 | `internal/anki/planner/migrate.go:214-218` | 변경 없음. 매핑으로 자동 처리. |
| 진단 코드 상수 블록 | `internal/anki/protocol/protocol.go:50-103` | 상수 하나 추가. |
| 코드 → 안내문 표 | `modules/org/anki/imoogi-error.el:20-73` (영어) | 항목 하나 추가. |
| keyed-error 줄 (`key (title): message`) | `modules/org/anki/imoogi.el:157-181` | 변경 없음. heading 이름이 공짜로 붙는다. |
| 양방향 짝짓기 테스트 | `tests/anki-error-test.el:9-31, 55-117` | 목록에 코드 하나 추가. |
| writeback은 `added`에만 반응 | `modules/org/anki/imoogi-writeback.el:75` | `skipped`는 heading을 건드리지 않는다. |
| 삭제 후보 = registry − census | `internal/anki/planner/planner.go:107-111` | 건너뛴 heading의 노트는 삭제 후보가 아니다. |

### A.2 기준선 측정 (`c010009`, 이번 세션에서 실행)

```
$ go test ./internal/anki/... -count=1 -cover
ok  .../internal/anki/ankiconnect  coverage: 86.2% of statements
ok  .../internal/anki/hashing      coverage: 100.0% of statements
ok  .../internal/anki/media        coverage: 91.3% of statements
ok  .../internal/anki/model        coverage: 98.4% of statements
ok  .../internal/anki/orgdoc       coverage: 100.0% of statements
ok  .../internal/anki/planner      coverage: 92.8% of statements
ok  .../internal/anki/protocol     coverage: [no statements]
ok  .../internal/anki/registry     coverage: 87.8% of statements
```

측정하지 않은 것: `make test-elisp`, `make lint`, `make fmt-check`,
`make ci-local`. run 단계가 M1 시작 전에 기준선을 잰다(§ C).

### A.3 PRESERVE 목록

run 단계의 어떤 마일스톤에서도 수정하지 않는다:

- `CLAUDE.md`, `go.mod`, `go.sum`, `go.work`(있다면), `vendor/**`,
  `packages.*`.
- 다른 SPEC 디렉터리 전부: `.moai/specs/SPEC-ANKICARD-001/`–`-004/`와 그 밖의
  모든 `.moai/specs/SPEC-*`. SPEC-ANKICARD-003의 `progress.md:319`와
  SPEC-ANKICARD-004의 `spec.md:972-980`도 포함 — 둘의 정리는 § sync 단계 후속
  작업에 적었다.
- `internal/anki/orgdoc/swift*`(존재한다면. 기준선에는 없다).
- `internal/anki/orgdoc/extra_test.go`, `golden_baseline_test.go`,
  `multiline_golden_test.go`, `internal/anki/orgdoc/testdata/**`,
  `internal/anki/hashing/**`. 이 파일들이 바뀌지 않는다는 것이 REQ-AKX-008의
  증거다.
- `internal/anki/planner/card_options.go` — 검증 게이트. 규칙을 더하지 않는다.
- `modules/org/anki/imoogi.el`, `imoogi-writeback.el`,
  `imoogi-{props,scan,process}.el`.
- `splitExtraBlocks`의 시그니처와 본문의 짝짓기 동작.

## § B 설계 결정

### DD-1 — 중첩은 구현하지 않고, 짝짓기는 Org와 같게 둔다

Org 9.7.11 측정(`spec.md` HISTORY v0.1.0): 같은 이름의 special block은
중첩되지 않고, 첫 BEGIN은 **첫** END에서 닫힌다. 짝 없는 BEGIN은 문단이다.
현재의 비탐욕 짝짓기는 이 의미와 정확히 같다. 중첩을 구현하면 Org 버퍼에서
사용자가 보는 구조(org-element가 읽는 구조)와 카드의 구조가 **달라진다**.
그래서 중첩 지원은 기능 추가가 아니라 Org와의 불일치를 새로 만드는 일이다.

남는 문제는 "Org와 같은 규칙으로 읽었더니 표식이 남는" 입력을 어떻게 다루느냐
이고, 답은 운영자 결정 2 — 보고 — 다.

### DD-2 — 추출된 보조 내용도 검사한다 (운영자 결정 2의 열린 질문)

**결정: 검사한다. 추출 내용에 여는 표식 줄이 있으면 같은 코드로 거부한다
(REQ-AKX-002).**

이번 세션의 측정(분리 패턴을 그대로 복사한 probe, `go run`)으로 모양별 잔여물을
확인했다:

| 분리 이전 본문 (요약) | 남은 본문 | 추출 내용 | 남은 본문에 표식 | 추출에 표식 |
|---|---|---|---|---|
| `- A / BEGIN note END / END / - B` | `- A\n#+END_EXTRA\n- B` | `note` | 예 | 아니오 |
| `- A / END / - B` (홀로 선 END) | 그대로 | 없음 | 예 | 아니오 |
| `- A / BEGIN note / - B` (안 닫힘) | 그대로 | 없음 | 예 | 아니오 |
| `- A / BEGIN outer BEGIN inner END tail END / - B` (균형 중첩) | `- A\ntail\n#+END_EXTRA\n- B` | `outer\n#+BEGIN_EXTRA\ninner` | 예 | **예** |
| `- A / BEGIN first BEGIN second END / - B` (여는 2, 닫는 1) | `- A\n- B` | `first\n#+BEGIN_EXTRA\nsecond` | **아니오** | **예** |
| `- A / BEGIN one END / BEGIN two END / - B` (차례로 둘) | `- A\n- B` | `one\n\ntwo` | 아니오 | 아니오 |
| `Q {{c1::x}} / BEGIN` (마지막 줄, 개행 없음) | 그대로 | 없음 | 예 | 아니오 |
| 블록 안의 `,#+BEGIN_EXTRA`와 `=#+END_EXTRA= inline` | `Q {{c1::x}}` | 두 줄 그대로 | 아니오 | 아니오 |
| 들여쓴 소문자 `  #+begin_extra … #+end_extra` | `Q {{c1::x}}` | `lower` | 아니오 | 아니오 |
| 줄 중간 `see #+END_EXTRA mid-line` | 그대로 | 없음 | 아니오 | 아니오 |

근거:

1. **정합성.** 추출 쪽을 보지 않으면 다섯째 행(`BEGIN BEGIN END`)은 통과하고
   넷째 행(`BEGIN BEGIN END END`, 더 균형 잡힌 입력)은 거부된다. 진단의 뜻이
   "표식이 짝지어지지 않았다"가 아니라 "잔여물이 어느 쪽에 떨어졌는가"가 된다.
2. **카드의 불만 그 자체.** t16의 불만은 "표식이 카드에 글자로 보인다"이다.
   다섯째 행은 지금 Back Extra에 `#+BEGIN_EXTRA`를 글자로 싣는다 —
   `internal/anki/planner/multiline_test.go:526-551`
   (`TestMultilineUnbalancedOpeningLeavesTheQuestionClean`)이 측정해 고정했다.
3. **오탐 탈출구가 있다.** 표식 글자를 정말 카드에 보여야 하는 사용자는 Org의
   블록 내부 이스케이프 관례인 `,#+BEGIN_EXTRA`나 줄 중간·verbatim 표기를 쓸 수
   있고, 측정상 이것들은 줄 머리 패턴에 걸리지 않는다(여덟째·열째 행).
4. **구성상 여는 표식만 보면 된다.** 추출 내부에 줄 머리 END가 있었다면 그 줄이
   매치를 끝냈을 것이다. 구현은 두 표식을 같은 판정 함수로 검사해도 결과가 같다
   — 그 편이 판정 함수를 하나로 유지한다.

### DD-3 — `failed`가 아니라 `skipped`로 보고한다

**결정: `skipped` + `extra_block_unbalanced` + 기존 식별자 + key (REQ-AKX-005).**

위임 지시문은 "그 노트는 실패한다"고 적었지만, 코드베이스의 계약은
`renderError`의 문서 주석(`internal/anki/planner/multiline.go:43-51`)이
정한다: 사용자가 Org 버퍼에서 고칠 수 있는 내용상의 결함은 **skip**이고
컬렉션·registry·heading을 건드리지 않는다. 분류되지 않은 렌더 오류만 `failed`
+ `org_parse_error`다. 렌더 중에 알 수 있는 기존 두 코드(`cloze_marker_missing`,
`multiline_answer_missing`)가 모두 skip이고, SPEC-ANKICARD-003 REQ-ML-002.2가
"다른 skip 진단과 같은 대우"를 명문화했다. 짝 없는 표식은 정확히 이 부류다.

운영자가 원한 관찰 결과는 모두 충족된다: 추가·갱신 없음, 다른 노트는 진행,
heading을 짚는 진단. Emacs 쪽 writeback은 `added`에만 반응한다
(`modules/org/anki/imoogi-writeback.el:75`). 마이그레이션 경로는 이미 모든 렌더
오류를 `skipped`로 돌려준다(`internal/anki/planner/migrate.go:217`,
`code, _ := renderError(err)`). 그래서 `skipped`를 고르면 두 경로가 같은 값을
보고한다.

**이 선택은 운영자 결정이다**(v0.1.1, plan audit iteration 1 이후 확정).

### DD-4 — 판정은 분리 바로 뒤, 한 곳에

- orgdoc에 이 조건 전용의 오류 타입을 둔다(가칭 `ExtraBlockUnbalancedError`,
  `ClozeMarkerMissingError`·`MultilineAnswerMissingError`와 같은 모양, 메시지는
  영어 한 문장).
- 판정 함수 하나(가칭 `hasExtraMarkerLine`)를 `extra.go`에 `extraBlockPattern`
  옆에 둔다. 두 호출 지점이 어긋나지 않도록, "분리 + 판정"을 한 번에 하는 작은
  도우미를 두고 `orgdoc.go:73`과 `multiline.go:602` 둘 다 그것을 부르게 하는
  쪽을 권한다. `splitExtraBlocks` 자체는 손대지 않는다 — 그래야
  `extra_test.go`가 바이트 그대로 통과한다는 것이 구조적으로 보장된다.
- 순서: Cloze 일반 렌더는 분리 → 판정 → 표식 게이트. multiline 렌더는 분리 →
  판정 → 구성 → 표식 게이트. 카드 옵션 검증 게이트는 여전히 그 앞
  (`planner.go:286`).

### DD-5 — 판정 문법은 분리 문법과 같다

판정 패턴은 분리 패턴의 여는/닫는 줄 조각과 같은 모양이어야 한다: `(?im)`,
`^[ \t]*#\+(begin|end)_extra`. 가능하면 조각을 상수 하나로 뽑아 두 정규식이
공유하게 한다. 분리 패턴의 `[^\n]*` 관대함(`#+BEGIN_EXTRAS`도 여는 표식으로
셈)은 넓히지도 좁히지도 않는다 — 판정도 같은 관대함을 따른다. 줄 끝 개행은
요구하지 않는다(개행 없는 마지막 줄의 BEGIN을 잡기 위해).

### DD-6 — 테스트 처분표

운영자 결정 4: 잔여 표식을 "평범한 내용"으로 고정한 테스트를 진단을 기대하도록
바꾼다. 그런데 `internal/anki/orgdoc/multiline_test.go:150-223`의 다섯 행은
**분리 이후** 잔여물을 스캐너에 직접 넣는 스캐너 수준 코퍼스라서 진단을 기대할
수 없다(스캐너는 진단을 내지 않는다). 그래서 "교체"를 이렇게 해석한다:

| 테스트 | 위치 | 처분 |
|---|---|---|
| 코퍼스 행 `dangling_end_marker_ends_the_answer_list`, `dangling_end_marker_survives_the_collapse`, `lone_end_marker_with_no_opening`, `dangling_begin_marker_swallows_the_later_answer`, `balanced_nested_blocks_leave_a_stray_closing_marker` | `orgdoc/multiline_test.go:163-211` | 스캐너 코퍼스에서 **제거**하고, 분리 이전 본문으로 바꿔 새 `TestRenderRejectsUnpairedExtraMarker`(AC-AKX-001)에 옮긴다. 이 입력은 이제 구성 단계에 도달하지 못한다. |
| 코퍼스 행 `sequential_blocks_leave_only_a_gap_the_collapse_closes` | `orgdoc/multiline_test.go:212-223` | 유지(양성 대조군). |
| 코퍼스 구역 머리 주석 | `orgdoc/multiline_test.go:150-161` | 다시 쓴다: 잔여 표식은 분리 직후 거부되며 여기 남은 행은 대조군이다. |
| collapse 행 `an_orphan_end_marker_opens_nothing` | `orgdoc/multiline_test.go:324-332` | 행은 유지(collapse 함수는 `#+END_SRC` 등 모든 `#+END_`에 대한 일반 함수이고 사실은 여전히 참). "잘 쓴 입력에서 도달 가능" 주석만 고친다. |
| `TestMultilineDanglingExtraMarker` | `planner/multiline_test.go:425-524` | `TestMultilineDanglingExtraMarkerIsReported`로 이름을 바꾸고, 네 모양 모두 `expectSkippedWithCode(…, protocol.CodeExtraBlockUnbalanced)`를 기대하게 한다. 문서 주석을 다시 쓴다. |
| `TestMultilineUnbalancedOpeningLeavesTheQuestionClean` | `planner/multiline_test.go:526-551` | `TestMultilineUnbalancedOpeningIsReported`로 바꾸고 skip + 코드를 기대(DD-2). |
| `TestMultilineSequentialExtraBlocksKeepEveryAnswer` | `planner/multiline_test.go:553-590` | 유지(양성 대조군). 문서 주석의 "dangling-marker table" 참조만 새 이름으로. |
| `TestRenderErrorMapping` | `planner/multiline_test.go:391-423` | 행 하나 추가. |
| `internal/anki/orgdoc/extra_test.go`, `internal/anki/orgdoc/golden_baseline_test.go`, `internal/anki/orgdoc/multiline_golden_test.go`, `internal/anki/planner/extra_alias_test.go` | — | 수정 없음. |

**코퍼스 행을 옮겨도 orgdoc 커버리지 100.0%는 유지된다(측정).** 이번 세션에서
orgdoc 패키지와 `go.mod`/`go.sum`을 scratchpad에 복사하고, 복사본의
`multiline_test.go`에서 다섯 행(163-211행)을 지운 뒤 기준선 코드에 대해
`go test ./internal/anki/orgdoc -count=1 -cover`를 돌렸다. 결과는
`ok … coverage: 100.0% of statements`였다. 다섯 행이 밟는 스캐너 분기는 남은
코퍼스 행들이 이미 덮는다. run 단계는 새 판정 분기(표식 있음/없음, 남은 본문
쪽/추출 쪽)를 AC-AKX-001–003의 양·음 행으로 덮어서 100.0%를 지킨다. 새 분기가
덮이지 않아 100.0% 아래로 떨어지면, 행을 지우지 말고 주석만 고친 스캐너 수준의
사실 행으로 되돌리는 것이 대안이다.

### DD-7 — Emacs 쪽은 표 항목 하나

`imoogi--keyed-error-lines`(`modules/org/anki/imoogi.el:157-181`)가 이미 모든
keyed 오류를 `key (title): message`로 보여 준다. 그래서 heading 이름은 추가
작업 없이 붙는다. 표는 영어이므로 영어 안내문을 쓴다. 초안:

> This heading's body has a `#+BEGIN_EXTRA' or `#+END_EXTRA' line with no
> partner, so the heading was skipped and nothing was written for it; a note
> already in Anki from an earlier sync keeps its previous content.
> Supplementary blocks do not nest -- as in Org itself, the first
> `#+END_EXTRA' closes the first `#+BEGIN_EXTRA' -- so a block written inside
> another, a second opening before the closing, or a leftover marker each leave
> one line unpaired. Write each supplementary block as its own
> `#+BEGIN_EXTRA' ... `#+END_EXTRA' pair, one after another, and delete any
> marker left over. To show the marker text itself on a card, write it as
> verbatim text (`=#+BEGIN_EXTRA=') or place it mid-line; inside a
> `#+BEGIN_SRC' or `#+BEGIN_EXAMPLE' block, escape it with a leading comma
> (`,#+BEGIN_EXTRA') instead.

산문에서 쉼표 이스케이프를 권하지 않는 이유(plan audit iteration 1의 probe, 감사
D5): 올바른 블록 안의 `,#+BEGIN_EXTRA`는 `<p>,#+BEGIN_EXTRA</p>`로 렌더되어
쉼표가 카드에 남는다. Org는 src/example 블록 안에서만 쉼표를 걷어 낸다.
verbatim `=#+END_EXTRA=`는 `<code class="verbatim">#+END_EXTRA</code>`로 깨끗하게
렌더된다.

금지 문자열(`goroutine`, `panic:`, `connection refused`, `dial tcp`,
`exit status`, `wrong-type-argument`, `file-missing`, `Debugger entered`)은
없다. `tests/anki-error-test.el`의 `imoogi-error-test--go-emitted-codes`에
코드를 추가하고, docstring의 "The last four are …" 문장이 낡았으므로 개수를
언급하지 않도록 고친다. ERT의 교차 검사 정규식
(`\_<Code[A-Za-z]+[ \t]*=[ \t]*"\([a-z_]+\)"`)에 상수
`CodeExtraBlockUnbalanced = "extra_block_unbalanced"`가 맞는다.

### DD-8 — SPEC-ANKICARD-004(swift)와의 관계

SPEC-ANKICARD-004는 draft이고, 잔여 표식을 "화살표가 아닌 평범한 줄"로 다룬다고
적었다(REQ-SW-001.7, Out of Scope `spec.md:972-980`). 004의 인수 기준도 같은
결과를 고정한다. `.moai/specs/SPEC-ANKICARD-004/acceptance.md:497` 행 "Dangling
`#+END_EXTRA` between two arrow lines | Both lines are still arrow lines; the
marker renders as visible literal text on its own line"이 그것이고,
`acceptance.md:118`은 이 고정이 "the precedent SPEC-ANKICARD-003 set"을 따른다고
적는다. 이 SPEC 이후 그 선례는 뒤집힌다. 이 SPEC이 먼저 착지하면,
swift 경로가 `RenderWithOptions`의 같은 분리를 거치는 한 판정도 거치고 그
문구는 사실상 공허해진다. 반대로 004가 먼저 착지해도 DD-4의 "분리 + 판정"
도우미를 swift 분기가 같이 쓰면 된다. 어느 쪽이든 004의 문구 정리는 그 SPEC의
재개 시점에 manager-spec이 할 일이며, 이 SPEC은 004 파일을 건드리지 않는다.

## § C 사전 점검

run 단계 시작 전에 한 번에(병렬로) 실행한다:

```bash
git rev-parse --short HEAD                     # c010009 또는 그 후손인지
git status --short -- internal/anki modules/org/anki tests
go test ./internal/anki/... -count=1 -cover    # § A.2와 같은지
make test-elisp                                # Elisp 기준선 (미측정)
grep -rn "splitExtraBlocks(" internal/anki     # 호출 지점이 두 곳인지 (orgdoc.go, multiline.go)
ls internal/anki/orgdoc | grep -i swift        # swift 파일 유무 (DD-8)
```

## § D 제약

- 새 응답 필드, 프로토콜 버전 변경, 새 노트 타입 없음.
- `splitExtraBlocks` 시그니처·동작 불변(DD-4).
- 골든 코퍼스 재녹화 금지.
- 판정 문법은 분리 문법과 같다(DD-5).

## § E 자기 검증 (run 단계 산출물)

manager-develop은 `progress.md` § E.2에 다음을 실제 실행 출력 그대로 남긴다:

1. `go test ./internal/anki/... -count=1 -cover` — 종료 코드와 패키지별 커버리지.
2. AC-AKX-001–006의 개별 `-run` 명령 출력.
3. `make test-elisp` 결과 줄(`Ran N tests, …, 0 unexpected`).
4. AC-AKX-008의 `git diff --stat` 출력(비어 있어야 함).
5. `make lint`, `make fmt-check` 종료 코드.

## § F 마일스톤

### M1 — 백엔드 판정과 보고 (우선순위 높음, 결정이 몰린 곳)

REQ-AKX-001, 002, 003, 004, 005, 006, 008.

1. `protocol.go`에 `CodeExtraBlockUnbalanced` 추가(주석: 렌더 중 조건, 게이트의
   고정 순서 밖 — `CodeMultilineAnswerMissing`과 같은 위치 논리).
2. `extra.go`에 판정 함수와 "분리 + 판정" 도우미, `orgdoc`에 오류 타입.
3. `orgdoc.go:73`, `multiline.go:602`의 호출을 도우미로 교체.
4. `planner/multiline.go`의 `renderError`에 case 하나(skip=true).
5. DD-6 처분표대로 테스트 교체·추가: orgdoc 새 테스트 두 개, planner 새 테스트
   두 개, 마이그레이션 테스트 하나, 프로토콜 테스트 하나, 기존 두 테스트 교체,
   매핑 행 추가.
6. 문서 주석 정리: `composeMultiline`(`multiline.go:476-503`)의 "보고하지 않음"
   단락을 이 SPEC의 결정으로 바꾸고, `splitExtraBlocks` 주석에 판정이 뒤따른다는
   한 줄을 더한다.

### M2 — 프런트엔드 안내문과 ERT (우선순위 중간, 기계적)

REQ-AKX-007.

1. `imoogi-error.el`에 DD-7 안내문 추가(`multiline_answer_missing` 항목 근처).
2. `tests/anki-error-test.el` 코드 목록과 docstring 정리.
3. `tests/anki-sync-error-test.el`에
   `imoogi-sync-error-test-extra-block-unbalanced-names-the-heading` 추가.
4. `make test-elisp`.

## § G 위험

- **R-1 (의도된 행위 변경).** 지금 잔여 표식을 단 채 동기화되던 heading은 다음
  동기화부터 건너뛰어진다. Anki의 기존 노트는 이전 내용 그대로 남고(갱신 안 됨,
  삭제 후보 아님 — `planner.go:107-111`), 사용자는 동기화 보고의 keyed 오류 줄
  `a.org::N (heading 제목): <안내문>`(`imoogi.el:157-181`)으로 **어느 heading**
  인지 안다. 고치고 다시 동기화하면 정상 갱신된다. 이 변경은 sync 단계의
  README 문구에도 적는다.
- **R-2 (오탐).** 표식 글자 자체를 줄 머리에 두고 싶은 사용자는 거부된다.
  산문이라면 탈출구는 verbatim `=#+BEGIN_EXTRA=`나 줄 중간 표기다(DD-2 측정,
  DD-7). **새로 생기는 거부 사례가 하나 있다.** `#+BEGIN_SRC`(또는 다른 블록) 안에
  홀로 선 줄 머리 EXTRA 표식이다. 예를 들어 Org 문법을 가르치는 카드
  `Q {{c1::x}}\n#+BEGIN_SRC org\n#+BEGIN_EXTRA\n#+END_SRC\n`에서 분리는 src 블록을
  모른다. 그래서 그 BEGIN이 남은 본문에 남고, 이 SPEC 이후 이 항목은
  `extra_block_unbalanced`로 거부된다(plan audit iteration 1 probe로 확인). 이전에는
  글자로 렌더됐다. 탈출구는 블록 안의 Org 쉼표 이스케이프 `,#+BEGIN_EXTRA`다.
  src 블록 안에서는 쉼표가 걷히므로 카드에 표식이 깨끗하게 보인다(이번 세션 probe: go-org는
  `#+BEGIN_SRC org\n,#+BEGIN_EXTRA\n#+END_SRC`를 `<pre>\n#+BEGIN_EXTRA\n</pre>`로,
  `#+BEGIN_EXAMPLE` 경우도 같게 렌더함). 안내문도 이
  방법을 알려 준다. 분리가 src 블록을 인식하게 만드는 것은 범위 밖이다(`spec.md`
  § 5 Changing the pairing rule).
- **R-3 (문법 어긋남).** 판정과 분리가 다른 문법을 쓰면 어긋난다. DD-5의 공유
  조각과 AC-AKX-001의 들여쓴 소문자 행이 막는다.
- **R-4 (swift 순서).** DD-8.
- **R-5 (커버리지).** orgdoc 100.0% 유지가 AC-AKX-008의 조건이다. 새 분기마다
  양·음 행이 있어야 한다.

## § H 테스트 방법

- 표 기반 Go 테스트, 입력은 **분리 이전** 본문(파이프라인 전체를 거치게).
- 두 진입점(Cloze 일반, multiline `->`)에 같은 행을 넣어 REQ-AKX-001.4를 확인.
- planner 테스트는 기존 도우미(`runOne`, `multilineEntry`,
  `expectSkippedWithCode`, 가짜 클라이언트의 `addCalls`)를 재사용.
- ERT는 기존 keyed-error 테스트를 본떠 `imoogi-process-runner`를 가짜로 바꾼다.

## sync 단계 후속 작업

manager-docs가 sync 단계에서 처리한다(이 SPEC의 run 단계는 건드리지 않는다):

- `README.md:1136` 근처 `#+BEGIN_EXTRA` 설명에 한두 문장: 보조 블록은 중첩되지
  않으며, 짝 없는 표식이 있는 heading은 `extra_block_unbalanced`로 건너뛴다.
- `.moai/project/domains.md:190`의 Known gaps `t16` 줄을 해소 표기로 바꾸거나
  지운다.
- `.moai/specs/SPEC-ANKICARD-003/progress.md:319`의
  `dangling_extra_marker: pinned-not-reported`는 이 SPEC이 대체한다. 003의
  파일은 수정하지 않고, 이 SPEC의 § E.4에 대체 사실을 기록한다.
- `.moai/reports/anki-card-types-plan-20260920.md:215`의 "중첩된
  `#+BEGIN_EXTRA`(정상 Org)"는 측정으로 틀린 전제다. 필요하면 정정 주석을 단다.
- SPEC-ANKICARD-004를 재개할 때 manager-spec이 다음을 맞춘다(DD-8): REQ-SW-001.7,
  Out of Scope 문구(`spec.md:972-980`), **`acceptance.md:497` 행**(떠도는
  `#+END_EXTRA`가 글자로 보인다고 고정한 행 — 이 SPEC 이후에는 거짓), 그리고
  `acceptance.md:118`의 선례 문장. 여기서는 후속 항목으로만 기록한다.
- 백로그 카드 t16의 종료는 kanban lead의 몫이다.
