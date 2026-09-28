---
id: SPEC-ANKICARD-005
title: "Report unpaired #+BEGIN_EXTRA / #+END_EXTRA markers"
version: "0.1.1"
status: completed
created: 2026-09-28
updated: 2026-09-28
author: jay
priority: P2
phase: "v0.2.0 target"
module: "internal/anki/orgdoc, internal/anki/planner, internal/anki/protocol, modules/org/anki"
lifecycle: spec-anchored
tags: "anki, org-mode, cloze, extra-block, diagnostics"
tier: S
related_specs: [SPEC-ANKICARD-003, SPEC-ANKICARD-004]
---

## HISTORY

### v0.1.1 (2026-09-28)

plan audit iteration 1(`.moai/reports/plan-audit/SPEC-ANKICARD-005-review-1.md`,
**FAIL 0.81**, 결함 기인)에 대한 응답. 운영자가 남은 두 질문에 답했다. REQ 8개,
AC 8개로 개수는 그대로다.

- CHANGED(운영자 결정, 감사 D2): frontmatter에서 `depends_on: [SPEC-ANKICARD-003]`을
  **삭제**하고 `related_specs`로 옮겼다. 이 SPEC에 필요한 것은 SPEC-ANKICARD-003의
  multiline 코드뿐이고, 그 코드는 이미 main(`08f7504`)에 있다. 003의 상태에 대한
  의존은 없다. 그래서 run 단계의 depends_on 사전 점검은 걸리지 않는다.
- CHANGED(운영자 결정): 거부된 항목을 `skipped`로 보고하는 것은 이제 가정이 아니라
  **운영자 결정**이다(REQ-AKX-005.2, `plan.md` DD-3). DD-3에 있던 `failed` 대안의
  비용 문단은 틀렸기 때문에 지웠다(감사 D3: `migrate.go:217`은 skip 플래그와 상관없이
  `skipped`를 돌려준다).
- CHANGED(감사 D1, 필수): AC-AKX-004 (b)의 본문을 `#+BEGIN_EXTRA\nnote\n`로
  바꿨다. 이 본문은 기준선에서 `MultilineAnswerMissingError`를 돌려주므로, 판정이
  구성 단계보다 먼저 돈다는 것을 실제로 가려낸다. 이전 본문(`… - A`)은 기준선에서도
  오류가 없어 순서를 검증하지 못했다. (a)·(b)의 기준선 결과를 AC에 적었다.
- CHANGED(감사 D5): 표식 글자를 보이게 하는 방법으로 verbatim `=#+BEGIN_EXTRA=`나
  줄 중간 표기를 권한다. 산문에서 쉼표 이스케이프 `,#+BEGIN_EXTRA`는 쉼표가 그대로
  보인다. 쉼표 이스케이프는 src/example 블록 안에서만 권한다(REQ-AKX-007.2, DD-7).
- CHANGED(감사 D6): `plan.md` R-2에 새로 생기는 거부 사례를 적었다. `#+BEGIN_SRC`
  블록 안의 줄 머리 EXTRA 표식은 이제 거부된다.
- CHANGED(감사 D4): SPEC-ANKICARD-004의 `acceptance.md:497` 행(떠도는
  `#+END_EXTRA`가 글자로 보인다고 고정한 행)과 `acceptance.md:118`의 선례 문장을
  DD-8과 sync 후속 작업에 적었다.
- CHANGED(감사 D7, D8): AC-AKX-007은 표의 안내문 전체를 그대로 확인한다.
  AC-AKX-005는 `writeSequence`에 `deleteNotes`가 없음도 확인한다.
- CHANGED(감사 D10): `extra_alias_test.go`에 전체 경로를 붙였다. 코퍼스 행을 옮길 때
  orgdoc 커버리지 100.0%를 지키는 방법을 DD-6에 적었다.
- 참고(감사 D9): 손대는 파일 수(운영 코드 6개와 테스트 여러 개)가 Tier S의 "5개
  미만" 기준을 넘는다. LOC는 Tier S 범위 안으로 본다. 이 기준은 안내일 뿐이므로
  Tier S를 유지한다.

### v0.1.0 (2026-09-28)

초안. 백로그 카드 **t16**("Anki 보조 블록 중첩/비대칭 처리")을 SPEC으로 옮긴다.
Tier S — `spec.md`(인수 기준 인라인 § 4) + `plan.md` + `progress.md`.

- **출발점.** 보조 블록 분리(`internal/anki/orgdoc/extra.go:17-18`의
  `extraBlockPattern`, 비탐욕 매칭)는 첫 `#+BEGIN_EXTRA`부터 첫 `#+END_EXTRA`
  까지만 떼어낸다. 남은 표식 줄은 본문에 그대로 남아 답 목록을 끊고, 카드에
  `#+END_EXTRA`가 글자로 보인다. 이 동작은 t12(`5d3868f`)에서 왔고,
  t14(`08f7504`, SPEC-ANKICARD-003)가 테스트로 고정만 했다
  (`.moai/specs/SPEC-ANKICARD-003/progress.md:319`
  `dangling_extra_marker: pinned-not-reported`).
- **측정 — 카드 전제의 정정.** 이번 세션에서 오케스트레이터가 Emacs 30 번들
  Org 9.7.11의 `org-element-parse-buffer`로 측정했다.
  `#+BEGIN_EXTRA\nA\n#+BEGIN_EXTRA\nB\n#+END_EXTRA\nC\n#+END_EXTRA\n`은
  첫 BEGIN부터 **첫** END까지를 special-block **하나**로 읽고, 그 뒤를 문단
  `"C\n#+END_EXTRA"`로 읽는다. 짝 없는 `#+BEGIN_EXTRA` 하나는 평범한 문단이다.
  즉 Org 자신도 같은 이름의 special block을 중첩하지 않는다. 카드가 전제한
  "중첩은 정상 Org 문법"은 **거짓**이고, 현재의 첫 BEGIN–첫 END 짝짓기는 Org와
  일치한다. 그래서 중첩을 구현하지 않는 것(운영자 결정 1)은 후퇴가 아니라 Org
  의미를 지키는 선택이다.
- **운영자 결정(확정).** (1) 중첩은 구현하지 않고 기존 짝짓기를 유지한다.
  (2) 분리 뒤에 짝 없는 표식이 남으면 그 노트만 새 진단 코드로 거부하고, 같은
  동기화의 다른 노트는 정상 진행한다. (3) Emacs 쪽 코드-안내문 표에 새 코드를
  추가하고 양방향 짝짓기 테스트를 통과시킨다. (4) 잔여 표식을 "평범한 내용"으로
  고정한 t14 테스트를 진단을 기대하도록 바꾼다.
- **이 초안이 정한 것.** 추출된 보조 내용 쪽도 검사한다(REQ-AKX-002, 근거
  `plan.md` DD-2). 거부된 노트는 `failed`가 아니라 `skipped`로 보고한다
  (REQ-AKX-005.2, 근거 `plan.md` DD-3. v0.1.1에서 운영자 결정으로 확정).
- **행위 변경 공지.** 지금 표식 잔여물을 단 채 동기화되던 heading은 이 SPEC
  이후 건너뛰어진다. 이미 Anki에 있던 노트는 이전 내용 그대로 남는다. 의도된
  변경이다(`plan.md` § G R-1).

## 1. 개요

### 1.1 목적

보조 블록 분리가 남긴 짝 없는 `#+BEGIN_EXTRA` / `#+END_EXTRA` 표식을 **조용히
카드에 싣지 않고 진단으로 보고**한다. 지금은 표식이 카드 앞면이나 Back Extra에
글자로 찍히고, 그 뒤의 답이 답 목록에서 빠진다. 사용자는 어떤 heading이
망가졌는지 알 길이 없다.

### 1.2 범위

범위 안: 분리 직후의 짝 없는 표식 판정, 새 진단 코드 `extra_block_unbalanced`
(protocol 상수, 플래너 매핑, Emacs 안내문), 진단 순서, 해당 테스트의 교체.

범위 밖: 중첩 지원, 짝짓기 규칙 변경, 자동 복구, Basic 노트 타입, swift 화살표
카드(SPEC-ANKICARD-004). § 5가 경계를 정한다.

### 1.3 읽는 순서

`spec.md`(이 파일, 인수 기준은 § 4에 인라인) → `plan.md`(설계 결정, 파일별
접근, 테스트 처분표). 배경 기록은
`.moai/reports/anki-card-types-plan-20260920.md:215`(t16 등록 문단)이다.

## 2. 용어

- **보조 블록 분리(supplementary split)** — Cloze 계열 렌더가 본문에서
  `#+BEGIN_EXTRA … #+END_EXTRA` 블록을 떼어 Back Extra로 보내는 단계
  (`internal/anki/orgdoc/extra.go:31`). Basic 렌더는 분리를 하지 않는다
  (`internal/anki/orgdoc/orgdoc.go:55-59`).
- **EXTRA 표식 줄(EXTRA marker line)** — 분리가 블록 경계로 인식하는 것과
  같은 모양의 줄: 줄 머리에서(앞의 공백·탭 허용) 대소문자 구분 없이
  `#+BEGIN_EXTRA` 또는 `#+END_EXTRA`로 시작하는 줄.
- **남은 본문(remaining body)** — 분리 뒤 질문 쪽에 남는 본문.
- **추출된 보조 내용(extracted supplementary content)** — 분리가 떼어낸 블록
  내부들을 이어 붙인 것. Back Extra 필드가 된다.
- **짝 없는 표식(unpaired marker)** — 분리가 끝난 뒤 남은 본문이나 추출된 보조
  내용에 남아 있는 EXTRA 표식 줄. 중첩 잔여물, 홀로 선 BEGIN/END, 여는 표식이
  닫는 표식보다 많은 경우를 모두 포함한다.

## 3. Requirements (GEARS)

규범 문장은 GEARS 형식의 영어로 쓰고, 하위 조항과 근거는 한국어로 쓴다.

### 3.1 판정

#### REQ-AKX-001 [When]
When the supplementary split leaves an EXTRA marker line in the remaining body
of an entry, the back end shall reject that entry with the diagnostic code
`extra_block_unbalanced`.

1. EXTRA 표식 줄의 판정 문법은 분리가 쓰는 문법과 **같아야 한다**: 줄 머리,
   앞의 공백·탭 허용, 대소문자 무시, `#+BEGIN_EXTRA` 또는 `#+END_EXTRA`로 시작.
   두 문법이 다르면 분리와 판정이 서로 다른 줄을 표식으로 세게 된다.
2. 줄 끝 개행이 없어도 표식이다. 본문 마지막 줄의 `#+BEGIN_EXTRA`(개행 없음)는
   분리 패턴에 걸리지 않고 남은 본문에 그대로 남으며, 이 요구사항이 잡아야 할
   대상이다.
3. 줄 중간에 나오는 표식 문자열, 쉼표로 이스케이프한 `,#+BEGIN_EXTRA`, 줄 머리가
   `=`인 verbatim `=#+END_EXTRA=`는 표식이 아니다. 측정: `plan.md` DD-2 표.
   (쉼표 이스케이프는 산문에서 쉼표가 그대로 렌더되므로 src/example 블록 안에서만
   쓸모가 있다. REQ-AKX-007.2 참고.)
4. 이 요구사항은 분리를 수행하는 **모든** 렌더 경로에 적용된다 — 오늘 기준
   Cloze 계열의 일반 렌더와 multiline 렌더 두 곳.

#### REQ-AKX-002 [When]
When the supplementary content the split extracted contains an opening EXTRA
marker line, the back end shall reject that entry with the diagnostic code
`extra_block_unbalanced`.

1. 추출된 내부에는 구성상 닫는 표식 줄이 있을 수 없다(있었다면 그 줄이 매치를
   끝냈다). 그래서 여는 표식만 검사하면 충분하다.
2. 이 조항이 잡는 모양: 중첩 블록의 안쪽 여는 표식, 그리고 "여는 표식 둘 +
   닫는 표식 하나". 후자는 남은 본문이 깨끗하므로 REQ-AKX-001만으로는 통과하고,
   지금은 Back Extra에 `#+BEGIN_EXTRA`가 글자로 찍힌다
   (`internal/anki/planner/multiline_test.go:526-551`이 측정해 고정).
3. 이 조항이 없으면 `BEGIN BEGIN END`는 통과하고 더 균형 잡힌
   `BEGIN BEGIN END END`는 거부되는 모순이 생긴다. 근거 전문은 `plan.md` DD-2.

#### REQ-AKX-003 [Ubiquitous — negated]
The supplementary split shall not change its pairing rule: each opening marker
pairs with the next closing marker, blocks shall not nest, and sequential blocks
shall keep concatenating in document order.

1. Org 9.7.11 측정(HISTORY v0.1.0)과 같은 규칙이다.
2. 중첩을 해석해 바깥 블록 전체를 떼어내는 것, 짝을 추측해 표식을 지우거나
   옮기는 것은 금지된다. 짝 없는 표식은 보고할 뿐 고치지 않는다.

### 3.2 파이프라인 위치와 보고

#### REQ-AKX-004 [Ubiquitous]
The unpaired-marker check shall run immediately after the split and before every
other render-time diagnostic, so that an entry carrying an unpaired marker
receives `extra_block_unbalanced` and no other diagnostic.

1. `cloze_marker_missing`(`internal/anki/orgdoc/orgdoc.go:73-76`)과
   `multiline_answer_missing`(`internal/anki/orgdoc/multiline.go:602-608`)보다
   먼저다. 짝 없는 표식은 질문과 보조 내용의 경계를 옮기고 답 항목을 문단에
   삼키므로, 뒤의 두 진단을 **일으키는 원인**이 될 수 있다. 원인을 먼저 보고해야
   사용자가 엉뚱한 줄을 고치지 않는다.
2. 카드 옵션 검증 게이트(`internal/anki/planner/planner.go:286-289`의
   `card_option_invalid` / `card_option_conflict` / `card_option_needs_cloze`)는
   여전히 렌더보다 먼저 돌고, 그 코드가 이긴다. 새 코드는
   `multiline_answer_missing`처럼 렌더 중에만 알 수 있는 조건이므로 게이트의
   고정 순서에 끼어들지 않는다(SPEC-ANKICARD-003 REQ-ML-011과 같은 논리).
3. 한 항목에는 진단이 하나만 붙는다.

#### REQ-AKX-005 [When]
When an entry is rejected with `extra_block_unbalanced`, the back end shall
report it as skipped carrying its entry key and its existing note identifier,
shall leave the collection, the registry, and the Org heading untouched for that
entry, and shall continue processing every other entry in the same run.

1. 일반 동기화 경로와 마이그레이션 경로 모두에 적용된다. 두 경로는 같은 오류
   매핑을 공유한다(`internal/anki/planner/multiline.go:43-63`,
   `internal/anki/planner/planner.go:295-306`,
   `internal/anki/planner/migrate.go:214-218`).
2. `skipped`로 보고한다 — **운영자 결정**(v0.1.1). 사용자가 Org 버퍼에서 고칠 수
   있는 내용상의 결함은 `skipped` + 코드라는 기존 계약(`renderError`의 문서 주석,
   `internal/anki/planner/multiline.go:43-51`)과 일치하며, 마이그레이션 경로는
   이미 모든 렌더 오류를 `skipped`로 돌려준다(`migrate.go:217`). 운영자가 요구한
   관찰 결과 — 추가·갱신 없음, 다른 노트는 진행, heading을 짚는 진단 — 는 모두
   충족된다.
3. 이전 동기화로 이미 Anki에 있는 노트는 갱신되지 않은 채 남는다. 삭제 후보도
   아니다: 삭제 후보는 registry에서 census가 보고한 식별자를 뺀 것이고
   (`internal/anki/planner/planner.go:107-111`), 건너뛴 heading의 식별자는
   census에 있다.

#### REQ-AKX-006 [Ubiquitous]
The protocol package shall declare `extra_block_unbalanced` as a diagnostic code
constant beside the existing codes, and the wire protocol version shall remain 2.

1. 응답 문서에 필드가 늘지 않는다. 기존 `errors[]`의 `code` 값이 하나 늘 뿐이다.
2. 상수 이름은 기존 `Code…` 관례를 따른다(`internal/anki/protocol/protocol.go:50-103`).

#### REQ-AKX-007 [Ubiquitous]
The front end's code-to-message table shall carry an entry for
`extra_block_unbalanced` that names the problem and states the corrective
action, and the per-entry error report shall name the heading that carried the
unpaired marker.

1. 안내문은 표의 기존 언어인 영어로 쓴다(`modules/org/anki/imoogi-error.el:20-73`).
2. 안내문이 담을 것: 짝 없는 표식이 있다는 사실, 이 heading은 건너뛰었고
   아무것도 쓰지 않았다는 사실, 이미 Anki에 있던 노트는 이전 내용을 유지한다는
   사실, 보조 블록은 중첩되지 않는다는 사실(Org도 같다), 고치는 법(블록을 차례로
   늘어놓기, 남은 표식 지우기). 표식 글자 자체를 카드에 보여야 하면 verbatim
   `=#+BEGIN_EXTRA=`으로 쓰거나 줄 중간에 쓰라고 권한다. 산문에서 쉼표 이스케이프
   `,#+BEGIN_EXTRA`는 쉼표가 카드에 그대로 보이므로 권하지 않는다. 쉼표
   이스케이프는 `#+BEGIN_SRC` / `#+BEGIN_EXAMPLE` 블록 안에서만 권한다.
3. 안내문에 Go 스택, HTTP 전송 오류, Emacs 백트레이스, 프로세스 종료 코드 같은
   원시 진단이 들어가지 않는다(`tests/anki-error-test.el`의 금지 문자열 검사).
4. heading 이름은 기존 keyed-error 줄이 이미 붙인다
   (`modules/org/anki/imoogi.el:157-181`, `key (title): message`). Emacs 쪽
   코드 변경은 표 항목 하나다.
5. 양방향 짝짓기 테스트(`tests/anki-error-test.el`)가 새 코드를 짝지어 통과한다.

### 3.3 비간섭

#### REQ-AKX-008 [Ubiquitous — negated]
The back end shall not change the rendered fields or the content hash of any
entry whose remaining body and extracted supplementary content carry no EXTRA
marker line after the split.

1. 표식이 없는 본문, 블록 하나, 차례로 늘어선 블록 여럿은 필드 값이 바이트
   단위로 같다. 해시 입력이 같으므로 불필요한 `updated`가 생기지 않는다.
2. Basic 노트 타입은 분리를 하지 않으므로 이 SPEC의 판정 대상이 아니다.
   `Render(Basic)`의 출력은 바뀌지 않는다.
3. 골든 코퍼스(`internal/anki/orgdoc/testdata/render-golden.json`,
   `multiline-golden.json`, `cloze-brace-fixture.json`)는 다시 녹화하지 않는다.
   측정: 세 파일 모두 EXTRA 표식 0개.

## 4. Acceptance Criteria (inline, Tier S)

각 기준은 Given–When–Then이고, 판정 명령이 붙는다. 테스트 이름은 run 단계의
목표 이름이며, `plan.md` DD-6이 기존 테스트의 처분을 정한다.

#### AC-AKX-001 — 남은 본문의 짝 없는 표식 (REQ-AKX-001, REQ-AKX-004)
- **Given** 새 테스트 `TestRenderRejectsUnpairedExtraMarker`
  (`internal/anki/orgdoc/extra_unbalanced_test.go`)의 분리 **이전** 본문 행들:
  쌍 뒤에 남은 `#+END_EXTRA`, 홀로 선 `#+END_EXTRA`, 닫히지 않은
  `#+BEGIN_EXTRA`(본문 중간), 개행 없이 마지막 줄에 놓인 `#+BEGIN_EXTRA`,
  균형 잡힌 중첩 블록, 들여쓴 소문자 `  #+end_extra`.
- **When** 각 행을 Cloze 일반 렌더와 multiline(`->`) 렌더 두 진입점에 넣으면,
- **Then** 두 진입점 모두 이 조건 전용의 타입 있는 오류를 돌려주고 필드 맵은
  돌려주지 않는다.
- 판정: `go test ./internal/anki/orgdoc -run TestRenderRejectsUnpairedExtraMarker -count=1` 종료 코드 0.

#### AC-AKX-002 — 추출된 보조 내용의 여는 표식 (REQ-AKX-002)
- **Given** 같은 테스트의 행 `- A / #+BEGIN_EXTRA / first / #+BEGIN_EXTRA /
  second / #+END_EXTRA / - B` — 분리 뒤 남은 본문은 깨끗하고, 추출 내용에
  `#+BEGIN_EXTRA`가 남는 모양.
- **When** 두 진입점에 넣으면,
- **Then** 둘 다 AC-AKX-001과 같은 오류를 돌려준다.
- 판정: AC-AKX-001과 같은 명령.

#### AC-AKX-003 — 표식이 아닌 것은 통과 (REQ-AKX-001.3, REQ-AKX-003, REQ-AKX-008)
- **Given** 새 테스트 `TestRenderAcceptsExtraMarkerLookalikes`의 행들: 올바른
  블록 안의 `,#+BEGIN_EXTRA`, 줄 중간의 `see #+END_EXTRA here`, 올바른 블록 안의
  `=#+END_EXTRA=`, 블록 하나, 차례로 늘어선 블록 둘.
- **When** Cloze 일반 렌더에 넣으면,
- **Then** 오류가 없고, 이스케이프·줄 중간·verbatim 표식 글자는 해당 필드에
  그대로 실리며, 차례로 늘어선 두 블록의 내부는 모두 Back Extra에 실린다.
- 판정: `go test ./internal/anki/orgdoc -run TestRenderAcceptsExtraMarkerLookalikes -count=1` 종료 코드 0.

#### AC-AKX-004 — 진단 순서 (REQ-AKX-004)
- **Given** 새 테스트 `TestExtraBlockUnbalancedPrecedence`
  (`internal/anki/planner/extra_unbalanced_test.go`)의 세 항목: (a) cloze 표식이
  어디에도 없고 `#+END_EXTRA`가 남는 Cloze 항목(기준선 결과:
  `cloze_marker_missing`), (b) 본문이 `#+BEGIN_EXTRA\nnote\n`인 multiline(`->`)
  항목(기준선 결과: `multiline_answer_missing` — plan audit iteration 1의 probe와
  이번 세션의 재측정 모두에서 `RenderWithOptions`가
  `*orgdoc.MultilineAnswerMissingError`를 돌려줌),
  (c) `ANKI_DIRECTION` 값이 인식 불가이고 본문에 짝 없는 표식이 있는 항목. (a)와
  (b)는 기준선에서 경쟁 코드를 내는 본문이므로, 판정이 해당 진단보다 **먼저**
  돌 때만 통과한다.
- **When** 각 항목을 한 번씩 동기화하면,
- **Then** (a)와 (b)는 오류가 정확히 하나이고 코드가 `extra_block_unbalanced`
  이며(`cloze_marker_missing`도 `multiline_answer_missing`도 아님), (c)는
  오류가 정확히 하나이고 코드가 `card_option_invalid`이다.
- 판정: `go test ./internal/anki/planner -run TestExtraBlockUnbalancedPrecedence -count=1` 종료 코드 0.

#### AC-AKX-005 — 그 노트만 건너뛰고 나머지는 진행 (REQ-AKX-005)
- **Given** 한 요청에 두 항목: 첫째는 기존 노트 식별자를 가진 Cloze 항목이고
  본문에 짝 없는 표식이 있다. 둘째는 올바른 새 Cloze 항목이다.
- **When** `TestExtraBlockUnbalancedSkipsOnlyThatEntry`가 동기화를 돌리면,
- **Then** 첫째 결과는 `skipped`이고 기존 식별자를 싣는다. 오류는 하나이고
  코드가 `extra_block_unbalanced`, key가 첫째 항목이다. 가짜 클라이언트는 첫째
  노트에 대한 `updateNoteFields` 요청을 받지 않고(`writeSequence` 기록,
  `internal/anki/planner/fake_client_test.go:238`), `writeSequence`에
  `deleteNotes`도 없으며, 둘째에 대한 추가 요청을 하나 받는다. 첫째의 registry
  기록(해시)은 실행 전과 같다. 아울러 교체된
  `TestMultilineDanglingExtraMarkerIsReported`(네 모양)와
  `TestMultilineUnbalancedOpeningIsReported`가 각 모양을 `skipped` +
  `extra_block_unbalanced`로 판정한다.
- 판정: `go test ./internal/anki/planner -run 'TestExtraBlockUnbalancedSkipsOnlyThatEntry|TestMultilineDanglingExtraMarkerIsReported|TestMultilineUnbalancedOpeningIsReported' -count=1` 종료 코드 0.

#### AC-AKX-006 — 마이그레이션 경로, 매핑, 프로토콜 상수 (REQ-AKX-005.1, REQ-AKX-006)
- **Given** (a) 짝 없는 표식이 있는 마이그레이션 후보, (b) `TestRenderErrorMapping`
  에 추가된 행 "an unpaired supplementary marker is a skip", (c) 새 테스트
  `TestExtraBlockUnbalancedCodeMatchesWireString`.
- **When** 각 테스트를 돌리면,
- **Then** (a) `TestMigrate_UnpairedExtraMarker_IsSkippedAndLeavesTheOriginal`은
  그 후보가 `skipped` + `extra_block_unbalanced`이고 원래 노트와 registry
  기록이 그대로임을 확인한다. (b) 매핑은 (`extra_block_unbalanced`, skip=true)
  이다. (c) 상수 값은 `"extra_block_unbalanced"`이고 `protocol.Version`은 2다.
- 판정: `go test ./internal/anki/planner ./internal/anki/protocol -run 'TestMigrate_UnpairedExtraMarker|TestRenderErrorMapping|TestExtraBlockUnbalancedCodeMatchesWireString' -count=1` 종료 코드 0.

#### AC-AKX-007 — Emacs 안내문과 heading 표시 (REQ-AKX-007)
- **Given** `imoogi-error-table`에 `extra_block_unbalanced` 항목이 있고,
  `tests/anki-error-test.el`의 `imoogi-error-test--go-emitted-codes`에 같은
  코드가 있으며, 새 ERT `imoogi-sync-error-test-extra-block-unbalanced-names-the-heading`
  (`tests/anki-sync-error-test.el`, 기존
  `imoogi-sync-error-test-keyed-error-names-the-heading-and-the-cause`를 본뜸)이
  key `a.org::0`에 이 코드를 실은 응답을 흉내 낸다.
- **When** `make test-elisp`를 돌리면,
- **Then** 종료 코드 0이고, `imoogi-error-test-go-codes-are-subset-of-table`,
  `imoogi-error-test-codes-match-source-file`,
  `imoogi-error-test-table-entries-are-all-paired-with-a-go-constant`,
  `imoogi-error-test-messages-contain-no-raw-diagnostics`가 통과하며, 새 ERT는
  보고문에 `a.org::0`과 `(imoogi-error-message "extra_block_unbalanced")`의
  반환값 전체가 그대로 들어 있고 Go의 원시 메시지는 없음을 확인한다. 부분 문자열
  `BEGIN_EXTRA`만으로는 부족하다. `multiline_answer_missing` 안내문에도 이 문자열이
  들어 있어서(`imoogi-error.el:68`), 코드를 잘못 매핑해도 통과할 수 있다.
- 판정: `make test-elisp` 종료 코드 0, 예상 밖 결과 0.

#### AC-AKX-008 — 비간섭과 기준선 (REQ-AKX-003, REQ-AKX-008)
- **Given** 기준선 `c010009`.
- **When** `go test ./internal/anki/... -count=1 -cover`와
  `git diff --stat c010009 -- internal/anki/orgdoc/extra_test.go internal/anki/orgdoc/golden_baseline_test.go internal/anki/orgdoc/multiline_golden_test.go internal/anki/orgdoc/testdata internal/anki/hashing`
  을 돌리면,
- **Then** 테스트는 종료 코드 0이고 `orgdoc` 커버리지는 100.0%, `planner`는
  92.8% 이상이다. diff 명령은 아무것도 출력하지 않는다.
  `TestMultilineSequentialExtraBlocksKeepEveryAnswer`와
  `TestRender_Basic_ExtraBlockStaysInlineAndEmitsNoBackExtra`는 수정 없이 통과한다.
- 판정: 위 두 명령의 출력.

## 5. Out of Scope

### Out of Scope — Nesting support

- 같은 이름의 보조 블록을 중첩해 읽는 기능. Org 9.7.11도 중첩하지 않으며
  (HISTORY v0.1.0), 중첩을 해석하면 Org 버퍼에서 보이는 구조와 카드가 달라진다.
  중첩된 모양은 REQ-AKX-001/002가 보고한다.

### Out of Scope — Changing the pairing rule

- 첫 BEGIN–다음 END 짝짓기, 대소문자 무시, 줄 머리 고정, 블록 여러 개의 이어
  붙이기를 바꾸는 일. 분리 패턴의 여는 줄 뒤 `[^\n]*` 관대함(예:
  `#+BEGIN_EXTRAS`도 여는 표식으로 셈)도 그대로 두며, 판정은 그 관대함을 그대로
  따른다(REQ-AKX-001.1).
- `#+BEGIN_SRC` 같은 다른 블록 안의 줄 머리 EXTRA 표식을 분리가 블록 경계로
  보는 기존 동작. 그 결과 src 블록 안에 홀로 선 줄 머리 표식은 이제 짝 없는
  표식으로 거부된다(`plan.md` § G R-2). 탈출구는 블록 안의 쉼표 이스케이프
  `,#+BEGIN_EXTRA`다.

### Out of Scope — Automatic repair

- 짝 없는 표식을 지우거나, 짝을 추측해 붙이거나, 남은 본문과 보조 내용 사이로
  내용을 옮기는 일. 이 SPEC은 보고만 한다.

### Out of Scope — The Basic note type

- Basic 렌더는 분리를 하지 않고 블록을 go-org의 special block으로 Back에
  그대로 싣는다(`internal/anki/orgdoc/extra_test.go:106-120`). 그 경로의 짝 없는
  표식 판정은 이 SPEC 밖이다.

### Out of Scope — Swift arrow cards (SPEC-ANKICARD-004)

- swift 화살표 카드의 렌더 규칙 자체. SPEC-ANKICARD-004(draft)의
  `### Out of Scope — The dangling supplementary marker`
  (`.moai/specs/SPEC-ANKICARD-004/spec.md:972-980`)가 이 문제를 t16에 넘겼다.
  이 SPEC은 swift 동작을 정하지 않는다. 다만 판정이 분리 바로 뒤에 놓이므로,
  swift 경로가 같은 분리를 거치면 REQ-AKX-001.4에 따라 판정도 거친다. 그때
  SPEC-ANKICARD-004의 REQ-SW-001.7 문구와 `acceptance.md:497` 행("Dangling
  `#+END_EXTRA` between two arrow lines … renders as visible literal text"),
  그리고 그 행이 인용하는 `acceptance.md:118`의 선례 문장을 맞추는 일은 그 SPEC의
  몫이다(`plan.md` DD-8).

## 6. Constraints

1. 새 노트 타입, 새 응답 필드, 프로토콜 버전 변경 없음.
2. `splitExtraBlocks`의 시그니처와 반환값은 바꾸지 않는다. 표식이 없는 본문을
   바이트 그대로 돌려주는 성질(`internal/anki/orgdoc/extra.go:23-27`)이 해시
   안정성의 근거다.
3. Emacs 쪽은 `modules/org/anki/imoogi-error.el`의 표 항목과 테스트만 바꾼다.

## 7. Traceability

| REQ | AC | 주요 파일 |
|---|---|---|
| REQ-AKX-001 | AC-AKX-001, AC-AKX-003 | `internal/anki/orgdoc/extra.go`, `orgdoc.go`, `multiline.go` |
| REQ-AKX-002 | AC-AKX-002 | `internal/anki/orgdoc/extra.go` |
| REQ-AKX-003 | AC-AKX-003, AC-AKX-008 | `internal/anki/orgdoc/extra.go`(짝짓기 불변) |
| REQ-AKX-004 | AC-AKX-001, AC-AKX-004 | `internal/anki/orgdoc/orgdoc.go`, `multiline.go` |
| REQ-AKX-005 | AC-AKX-005, AC-AKX-006 | `internal/anki/planner/multiline.go`, `planner.go`, `migrate.go` |
| REQ-AKX-006 | AC-AKX-006 | `internal/anki/protocol/protocol.go` |
| REQ-AKX-007 | AC-AKX-007 | `modules/org/anki/imoogi-error.el`, `tests/anki-error-test.el`, `tests/anki-sync-error-test.el` |
| REQ-AKX-008 | AC-AKX-003, AC-AKX-008 | golden 코퍼스, `extra_test.go`, `internal/anki/hashing` |

모든 요구사항은 인수 기준 하나 이상에 걸리고, 모든 인수 기준은 판정 명령을
가진다. `plan.md` § F가 요구사항을 마일스톤에 배정한다.
