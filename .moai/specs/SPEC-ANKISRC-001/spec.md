---
id: SPEC-ANKISRC-001
title: "Open the originating Org heading from an imoogi Anki card"
version: "0.1.2"
status: draft
created: 2026-10-12
updated: 2026-10-12
author: jay
priority: P2
phase: "v0.2.0 target"
module: "modules/org/anki, internal/anki/planner, internal/anki/protocol, cmd/imoogi-anki, internal/agentipc"
lifecycle: spec-anchored
tags: "anki, org-mode, org-id, url-scheme, emacsclient, macos"
tier: M
related_specs: [SPEC-ANKICARD-005, SPEC-AGENTIPC-002]
---

## HISTORY

### v0.1.2 (2026-10-12)

plan-audit 2회차(`.moai/reports/t25/plan-audit.md`, **FAIL 0.86**)의 세 모순에 lane 판정을 적용했다.
REQ 13개, AC 16개 그대로.

- R1: 수신 함수 결과 `"ok"`·`"not-found"`·`"duplicate"` 는 모두 정상 → 처리기 코드 0, macOS 알림 없음. 그 밖의 결과와 eval 오류만 알림 + 0 이 아닌 코드(REQ-010.2, AC-015 (d)(f)). M4 는 자체 분류기를 쓴다.
- R2: REQ-001.6 — ID 를 확보하지 못한 heading 만 `org_id` 없이 나가고, 디스크에 `:ID:` 가 있는 heading 은 그대로 싣는다.
- R3: `org-id-locations` 저장 1회는 `imoogi-sync` 끝, 보낸 모든 ID 의 REQ-005 등록 뒤로 옮겼다.
- O1: 10초 eval 예산을 위험(R-7)과 완화책(파일별 원문 정규식 검색)으로 기록.
- O2: ID 확보 단계는 결코 사용자에게 묻지 않는다. 닫는 버퍼는 이 단계가 연, 수정되지 않은 버퍼뿐이다.
- O3: 등록 대상 밖의 캐시 파일은 존재하고 heading-local 일치가 정확히 1개일 때만 받아들이며, 유일성은 대상 ∪ 그 파일에서 확인한다(AC-009 (b)).

### v0.1.1 (2026-10-12)

plan-audit 1회차(`.moai/reports/t25/plan-audit.md`, **FAIL 0.75**) 지적에 lane 판정을 적용했다.
D1–D8 은 다시 열지 않았다. REQ 13개, AC 16개.

- F1: REQ-005 는 Org 가 기록하는 형식(`~/` 축약 가능)의 "해석 가능한 경로"를 요구한다. AC 는 `expand-file-name`/`file-truename` 동치로 비교한다.
- F2: 허용 문자 집합에 `@` 추가(`org-id-include-domain`). §1.3 의 범위 주장을 바로잡았다. 링크 `href` 는 ID 를 퍼센트 인코딩하고 처리기는 디코딩 뒤 검증한다.
- F3: 진단 코드 `org_id_invalid` 를 Go 상수 + Elisp 메시지 표 항목으로 명명했다(REQ-006.3).
- F4: eval 실패의 가시 알림(REQ-010.2), 수신 함수의 모듈 로드 시 적재(REQ-011.7), 모든 Emacs 쪽 결과에서 프레임 활성화(REQ-011.8).
- F5: imoogi 렌더러를 거치는 stock `Basic`/`Cloze` 도 링크를 받는다(REQ-006).
- F6: ID 확보 단계의 파일별 실패 격리(REQ-001.6, AC-003 (b)).
- F7: 모든 emacsclient 호출에 호출별 시간 제한(REQ-010.3).
- F8: 열기 시 캐시 적중이어도 등록 대상 전체 스캔으로 유일성을 확인한다(REQ-011.2). 동기화 시 중복 집합을 따로 저장하는 방식보다 새 영속 상태가 없어 단순하므로 이쪽을 택했다.
- F9, F10: AC-004 는 중복 ID entry 가 여전히 전송됨을, AC-001 은 제외 파일 불변을 확인한다.
- F11: 운영자 클릭 확인과 기준선 비회귀를 Definition of Done 으로 옮겼다.
- F12–F15: REQ-012 이중 부정 제거, REQ-001.3 버퍼 정책 명시, REQ-003.1 에 `omitempty`/Version 2 예외 근거, REQ-013.1 바이너리 경로 인용·PATH 비의존.
- F16: `plan.md` R-4 위험과 운영자 확인 공백으로 기록(계획 변경 없음).

### v0.1.0 (2026-10-12)

최초 작성. 카드 t25("Emacs에서 원본 열기")를 요구사항으로 옮긴다. 설계 결정 D1–D8 과
사실 목록은 `.moai/reports/t25/progress.md` § "Stage 2 — design decisions" 에서 lane-5 가
이미 내렸고, 이 문서는 그것을 다시 열지 않고 요구사항으로 옮긴다. 기준선은 `fde2122`
(7bf2920 Org ID prerequisite 병합 포함)이며, 같은 진행 기록에 측정값이 있다
(ERT anki 33/33, `go test ./internal/anki/... ./cmd/imoogi-anki/...` ok).

## 1. 개요

### 1.1 목적

imoogi 가 만든 Anki 카드 뒷면의 "Emacs에서 원본 열기" 링크를 누르면, 그 카드를 만든 Org
heading(heading 자신의 `:ID:` 로 식별)이 Emacs 에서 열린다. 원본을 찾지 못하면 빈 파일을
만들지 않고 "찾을 수 없음" 안내와 재동기화 안내를 보여 준다.

### 1.2 범위

- 보내기 전 Org ID 확보(Elisp), ID ↔ 파일 위치 등록(`org-id-locations`), 프로토콜 `org_id`
- 백엔드 planner 의 링크 삽입(뒷면 필드)
- 여는 쪽: Go `open` 처리기 + macOS URL 처리 `.app` 생성/설치 명령 + Elisp 해석기
- 문서(README)

### 1.3 용어

- **heading-local ID**: heading 자기 property drawer 의 `:ID:`. 상속으로 얻은 부모 ID 는 해당하지 않는다.
- **링크**: `imoogi-anki://open?id=<퍼센트 인코딩된 ID>` 를 가리키는 HTML `<a>` 앵커(스크립트 없음).
- **허용 문자 집합**: `^[A-Za-z0-9._:@-]{1,128}$`. 기본 설정의 `org-id-method` 세 형식 `uuid`(16진수와 `-`),
  `org`(`prefix:random`), `ts`(`YYYYMMDDTHHMMSS.ffffff`)를 덮고, `org-id-include-domain` 이 켜졌을 때
  붙는 `@FQDN` 도 덮는다. 사용자가 `org-id-prefix` 에 이 집합 밖 문자를 넣었거나 손으로 쓴 ID 는
  덮지 않는다(그런 heading 은 REQ-006.3 으로 링크 없이 처리). `%` `&` `=` `/` `"` 공백·제어 문자는 막는다.
- **등록 대상**: `imoogi-targets-load` 가 돌려주는 폴더/파일과 legacy root(`imoogi-sync-root`).

## 2. Requirements (GEARS)

### 2.1 동기화 쪽 — ID 확보와 등록

#### REQ-ASRC-001 [When] 보내기 전 ID 확보 (D1)
When `imoogi-sync` 가 실행되면, the sync front end shall 스캔보다 먼저, 스캔이 동기화 대상으로
읽을 모든 파일의 `ANKI_NOTE_TYPE` heading 가운데 heading-local `:ID:` 가 없는 것에 새 ID 를
만든다.
- 001.1 heading-local 여부는 상속을 끈 조회로만 판정한다. 부모의 ID 는 결코 물려받지 않는다.
- 001.2 이미 있는 `:ID:` 는 한 글자도 바꾸지 않는다. 이미 동기화된(`ANKI_NOTE_ID` 있음) heading 에 ID 가 없으면 만들기만 한다.
- 001.3 버퍼 정책: 그 파일을 방문 중인 버퍼가 있으면 재사용하고, 수정되지 않은 버퍼만 저장하며, 수정된 버퍼는 저장하지 않고 needs-save 로 보고한다. 이 단계가 스스로 연 버퍼는 저장 후 닫고, 사용자가 이미 방문 중이던 버퍼는 닫지 않는다(writeback 과 달리 닫는 점이 다르다). 이 단계는 결코 사용자에게 묻지 않는다: 닫는 버퍼는 이 단계가 연 것 가운데 수정되지 않은 상태(저장 성공, 또는 실패 경로에서 이 단계의 변경을 버린 상태)인 것뿐이다.
- 001.4 이 단계는 추가 단계다. `added` 뒤 writeback 의 ID 확보(7bf2920)는 그대로 남는다(멱등 보조 장치).
- 001.5 동기화 제외 패턴에 걸린 파일(census 전용)은 건드리지 않는다.
- 001.6 파일 또는 heading 하나에서 ID 를 확보하지 못하면(읽기 전용 파일, 저장 오류, ID 생성 오류) 그 실패를 그 파일 범위에 가두고 보고하며, ID 를 확보하지 못한 heading 들은 `org_id` 없이 동기화된다. 디스크에 이미 `:ID:` 가 있는 heading 은 REQ-003 대로 그 ID 를 싣는다. 이 단계의 실패는 `imoogi-sync` 를 중단시키지 않는다.

#### REQ-ASRC-002 [Ubiquitous] 두 식별자의 분리
The sync front end shall `ANKI_NOTE_ID` 와 `:ID:` 를 서로 독립적으로 관리한다. 어느 쪽도 다른
쪽에서 유도하지 않으며, 한쪽을 쓰는 동작이 다른 쪽 값을 바꾸지 않는다.

#### REQ-ASRC-003 [When] 스캔과 프로토콜 전달 (D4)
When 스캔이 동기화 대상 heading 을 읽으면, the scanner shall 디스크 내용의 heading-local `:ID:`
를 읽어 요청 entry 의 선택 필드 `org_id` 로 싣는다.
- 003.1 `org_id` 는 값이 없으면 JSON 에서 생략하고(`omitempty`), 프로토콜 `Version` 은 2 로 유지한다. 이는 protocol 패키지 주석의 "omitempty 없음" 및 Version 정책에 대한 의도된 예외다. 근거: decoder 가 관대해 모르는 필드를 무시하므로 구 백엔드는 링크 없이 그대로 동작하고, 필드가 선택적이라 구 프런트엔드는 링크 없는 기존 동작을 유지한다. 구현은 패키지 주석에 이 예외를 기록한다.
- 003.2 수정된 버퍼에서 만들어져 아직 디스크에 없는 ID 는 이번 동기화에 실리지 않는다. 그 heading 은 링크 없이 나가고 needs-save 로 보고되며, 저장 후 다음 동기화에서 링크가 붙는다.

#### REQ-ASRC-004 [When] 중복 ID (D8)
When 한 번의 동기화가 읽는 heading 들 사이에 같은 `:ID:` 가 둘 이상 발견되면, the sync front end
shall 그 heading 들 모두를 `org_id` 없이 그러나 평소대로 전송하고(제외하거나 실패로 표시하지
않는다) ID 와 해당 파일 경로들을 보고한다. ID 는 다시 쓰지 않고, 그중 첫 번째를 조용히 고르지 않는다.

#### REQ-ASRC-005 [When] ID ↔ 위치 등록 (요구 2, 7)
When 동기화가 `org_id` 를 실은 entry 를 보내면, the sync front end shall Org 표준
`org-id-locations` 캐시에 그 ID 와 파일 위치를 Org 가 기록하는 형식(`abbreviate-file-name` 에
따른 `~/` 축약 가능)의 해석 가능한 경로로 등록하고, 보낸 모든 ID 의 등록이 끝난 뒤 `imoogi-sync` 끝에서 동기화마다
한 번 `org-id-locations-file` 에 저장한다(ID 확보 단계는 등록만 하고 저장하지 않는다).
- 005.1 별도 색인을 만들지 않고 Org 의 `org-id-locations` 를 재사용한다.
- 005.2 Anki 쪽 링크에는 ID 만 들어가고 파일 경로는 결코 들어가지 않는다.

### 2.2 백엔드 — 링크 삽입

#### REQ-ASRC-006 [When] 뒷면 링크 (D2, D3, D5)
When entry 가 허용 문자 집합을 통과한 `org_id` 를 가지면, the planner shall 렌더와 미디어
재작성 뒤, 해시 계산 전에 렌더된 뒷면 필드 끝에 링크 앵커를 덧붙인다. 뒷면 필드는 Basic 형태의
`Back`, Cloze 형태의 `Back Extra` 이며, imoogi-Basic/imoogi-Cloze 뿐 아니라 같은 렌더러를 거치는
stock `Basic`/`Cloze` entry 에도 똑같이 적용된다.
- 006.1 앞면 필드(`Front`, `Text`)는 바꾸지 않는다. 새 모델 필드를 추가하지 않는다.
- 006.2 `org_id` 가 없는 entry 의 필드는 기준선과 바이트 단위로 같다.
- 006.3 허용 문자 집합을 통과하지 못한 `org_id` 는 링크 없이 처리하고, 그 entry 의 key 를 단 진단 코드 `org_id_invalid` 를 남긴다. 이 코드는 protocol 진단 코드 상수 블록의 상수이며 프런트엔드 메시지 표에 항목을 가진다. 노트 추가·갱신 자체는 막지 않는다.

#### REQ-ASRC-007 [Ubiquitous] 해시 일관성
The planner shall 링크를 렌더된 필드 내용의 일부로 해시한다. 그래서 orphan-delete 확인의
`notesInfo` 재해시가 같은 값을 얻고, 링크가 붙은 뒤의 재동기화는 변경이 없다.

#### REQ-ASRC-008 [When] 기존 노트의 1회 갱신 (D2 귀결)
When 링크가 없던 기존 노트의 entry 가 처음 `org_id` 와 함께 동기화되면, the planner shall 그
노트를 같은 note ID 로 한 번 갱신(update)한다. 삭제 후 추가하지 않으며, 다음 동기화는 변경이 없다.

### 2.3 여는 쪽

#### REQ-ASRC-009 [When] URL 처리기 (D5)
When `imoogi-anki open <url>` 이 호출되면, the handler shall scheme `imoogi-anki`, host `open`,
`id` 질의 매개변수 정확히 1개를 확인하고, 그 값을 퍼센트 디코딩한 뒤 허용 문자 집합으로 검증하고
나서야 Emacs 를 부른다.
- 009.1 검증 실패 시 emacsclient 를 부르지 않고, 0 이 아닌 코드로 끝나며, 보이는 macOS 알림을 띄운다.
- 009.2 emacsclient 는 `internal/agentipc` 의 탐색 규칙($EMACSCLIENT, PATH, 앱 번들 후보)으로 찾고 shell 없이 실행한다. shim 에서 호출될 때는 사용자 PATH 와 `$EMACSCLIENT` 가 없을 수 있으므로 앱 번들 후보로 찾게 되며, 여러 Emacs 번들이 있을 때는 emacsclient 를 찾은 그 번들을 실행·활성화 대상으로 쓴다.
- 009.3 `--eval` 식은 고정된 Elisp 수신 함수 호출이며 ID 는 올바르게 인용된 Lisp 문자열 리터럴로만 들어간다(1차 방어: 문자 집합, 2차 방어: 문자열 인용).

#### REQ-ASRC-010 [When] Emacs 미실행과 호출 실패 (D6)
When emacsclient 탐침이 실패하면, the handler shall Emacs 를 실행하고(shell 없이) 정해진 상한
시간 동안 emacsclient 를 다시 탐침한다. 상한을 넘기면 보이는 macOS 알림/대화상자를 띄우고 0 이
아닌 코드로 끝난다. 처리기의 어떤 실패 경로도 조용히 끝나지 않는다.
- 010.1 emacsclient 를 어디에서도 찾지 못하면 Emacs 실행을 시도하지 않고, 즉시 보이는 macOS 알림을 띄우며 0 이 아닌 코드로 끝난다.
- 010.2 수신 함수의 정상 결과는 `"ok"`, `"not-found"`, `"duplicate"` 세 가지이며, 셋 모두 처리기를 코드 0 으로, macOS 알림 없이 끝낸다(0건·중복 안내는 초점을 받은 Emacs 프레임에 보인다, REQ-011.8). eval 이 0 이 아닌 코드로 끝나거나, 오류를 출력하거나(수신 함수 미정의 포함), 세 정상 결과 밖의 값을 돌려주면, 그 오류 내용을 담은 보이는 macOS 알림을 띄우고 0 이 아닌 코드로 끝난다.
- 010.3 탐침과 eval 을 포함한 모든 emacsclient 호출은 호출별 시간 제한을 가지며(테스트에서 주입 가능), 시간 초과는 보이는 알림과 0 이 아닌 코드로 끝난다. 처리기 전체 실행 시간은 실행 대기 상한 + 호출별 제한의 합으로 유계다.

#### REQ-ASRC-011 [When] 열기 쪽 해석 (D7)
When 수신 함수가 ID 를 받으면, the resolver shall 다음 순서로 원본을 찾는다.
- 011.1 수신 함수도 ID 를 허용 문자 집합으로 다시 검증하고, 실패하면 아무 파일도 열지 않는다.
- 011.2 `org-id-locations` 를 먼저 보고, 그 위치를 검증한다: 파일이 존재하고, heading-local `:ID:` 가 그 ID 인 heading 이 그 파일에 정확히 1개다. 캐시 적중이어도 이동하기 전에 등록 대상 전체의 동기화 대상 파일을 스캔해 그 ID 가 전체에서 유일한지 확인한다(F8: 동기화 시 중복 집합을 저장하지 않고 열 때 전체를 확인한다 — 새 영속 상태가 없어 더 단순하다). 캐시 파일이 등록 대상 밖에 있으면, 그 파일이 존재하고 heading-local 일치가 정확히 1개일 때만 받아들이며, 유일성은 등록 대상 ∪ 그 파일에서 확인한다. 그 파일을 방문 중인 버퍼가 있으면 그 버퍼를 쓴다.
- 011.3 캐시가 없거나 검증에 실패하면 같은 전체 스캔 결과로 찾는다.
- 011.4 0건: ID 와 재동기화(`M-x imoogi-sync`)·확인 안내를 담은 메시지를 보이고, 파일·버퍼를 만들지 않는다.
- 011.5 1건: `org-id-locations` 에 위치를 등록해 저장하고, 그 heading 으로 이동한다.
- 011.6 2건 이상: 이동하지 않고 후보(파일 경로와 heading 제목) 목록을 보인다.
- 011.7 수신 함수는 Anki 모듈이 적재될 때 함께 적재된다(autoload 쿠키에 기대지 않는다).
- 011.8 열기·찾을 수 없음·중복의 모든 결과에서 Emacs 프레임을 앞으로 가져와 입력 초점을 준다. 사용자가 Anki 를 보고 있어도 안내가 보여야 하기 때문이다.

#### REQ-ASRC-012 [Ubiquitous] 사용자 편집 보호
The resolver shall 사용자 버퍼를 저장·되돌리기·수정 없이 그대로 두고, `:ID:` 와 `ANKI_NOTE_ID`
를 쓰지 않는다. 열기 쪽이 쓰는 것은 `org-id-locations-file` 뿐이다.

#### REQ-ASRC-013 [When] macOS 처리기 설치 (D5)
When 사용자가 처리기 설치 명령을 실행하면, the installer shall 얇은 AppleScript shim `.app` 을
생성하고(`Info.plist` 에 `CFBundleURLTypes` / `CFBundleURLSchemes` = `imoogi-anki`) Launch
Services 에 등록한다.
- 013.1 shim 은 생성 시점에 고정한 `imoogi-anki` 의 절대 경로를 부르며 사용자 PATH 에 의존하지 않는다. 그 경로와 URL 은 각각 따로 셸 인용된다(경로에 공백이 있어도 안전). 검증은 Go 처리기가 다시 한다.
- 013.2 생성과 등록은 분리되어, 등록 없이 임시 디렉터리에 생성만 할 수 있다.

## 3. Out of Scope

### Out of Scope — 모델 변경
- 새 `OrgId` 필드나 모델 필드 마이그레이션(D2 에서 기각)
- 앞면 링크, JavaScript 기반 링크

### Out of Scope — 다른 진입 경로와 플랫폼
- `org-protocol`, Anki add-on
- Linux/Windows URL 처리기, macOS 외 Emacs 실행 방식
- `migrate` 하위 명령 경로의 링크 삽입(재배치된 노트는 다음 일반 동기화에서 링크를 얻는다)

### Out of Scope — ID 자동 수리
- 중복 ID 의 자동 병합·재작성, 기존 ID 의 재작성
- 허용 문자 집합 밖 기존 ID 의 정규화

## 4. Constraints

- macOS + 현재 Emacs/emacsclient 우선. 처리기 실행 파일은 shell 을 거치지 않는다.
- 테스트는 `org-id-locations-file` 과 메모리의 ID 표를 임시 상태로 격리한다. 사용자의 실제 캐시를 읽거나 쓰지 않는다.
- 기준선 테스트(ERT anki 33, Go anki 패키지)는 계속 통과한다.

## 5. Traceability

| 카드 요구 | REQ | AC (`acceptance.md`) |
|---|---|---|
| (1) 보내기 전 heading-local ID | 001, 002 | 001, 002, 003 |
| (2) ID ↔ 경로 등록, 경로 없는 링크 | 005 | 005, 006 |
| (3) Org ID 전달 + 뒷면 링크 | 003, 006, 007, 008 | 006, 007, 008 |
| (4) 처리기 → Emacs 이동 | 009, 011, 013 | 009, 014, 015, 016, DoD 운영자 확인 |
| (5) 캐시 갱신 / 없음 안내 | 011 | 010, 011 |
| (6) 이미 동기화됐지만 ID 없는 heading, 중복 | 001.2, 004, 011.2, 011.6 | 002, 004, 012 |
| (7) 미저장 버퍼, Emacs 미실행, 캐시 저장 시점 | 001.3, 001.6, 003.2, 005, 010, 012 | 003, 005, 013, 015 |
