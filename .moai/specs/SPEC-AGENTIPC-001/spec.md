---
id: SPEC-AGENTIPC-001
title: "Coding Agent -> Emacs IPC bridge MVP (imoogi-agent)"
version: "0.1.2"
status: completed
created: 2026-09-27
updated: 2026-09-27
author: jay
priority: P1
phase: "v0.x agent-ipc MVP"
module: "cmd/imoogi-agent, modules/development/30-agent.el"
lifecycle: spec-anchored
tags: "agent, ipc, emacsclient, cli, notification, air-gap"
tier: M
---

## HISTORY

### v0.1.2 (2026-09-27)

plan-audit 2회차(`.moai/reports/plan-audit/SPEC-AGENTIPC-001-review-2.md`, PASS 0.93)의 N1 반영. 요구사항·인수 기준 개수는 15/15 그대로다.

- N1: 에코 영역 알림 판정을 "`*Messages*` 마지막 줄"에서 "수신 호출 동안 X 와 같은 줄이 정확히 하나 추가됨"으로 바꿨다
  (`acceptance.md` § 0.1, AC-AIPC-001 And 절). 파일을 보여 주는 처리기는 표시 뒤에 알림을 낸다고 `plan.md` § 3.1 에 적었다.

### v0.1.1 (2026-09-27)

plan-audit 1회차(`.moai/reports/plan-audit/SPEC-AGENTIPC-001-review-1.md`, PASS 0.89) 지적 반영. 요구사항·인수 기준 개수는 15/15 그대로다.

- D1: `plan.md` D-1 의 `"error:<reason>: <detail>"` 허용을 삭제했다. 상태 문자열은 어디서나 정확히 `"ok"` 또는 `"error:<reason>"` 이다.
- D2: AC-AIPC-012 를 REQ-AIPC-011 과 맞춰 stderr 진단을 "정확히 한 줄"로 고정했다.
- D3: 선택 필드의 형식을 엄격히 검사한다(REQ-AIPC-002.4). 파싱 옵션을
  `:object-type 'alist :array-type 'array :null-object :null :false-object :false` 로 바꿔 `[]`·`{}`·`false` 가
  "없음"과 구별되게 했고, AC-AIPC-007 에 행 n·o 를 추가했다.
- D4: 알림 줄·로그 줄에 들어가는 값의 줄바꿈 처리를 정했다(REQ-AIPC-004.5, REQ-AIPC-007.1). AC-AIPC-001 에 줄바꿈 사례를 추가했다.
- O1: AC-AIPC-013 에서 `--timeout=N` 추가 인자 허용을 삭제했다(emacsclient 인자는 `--eval EXPR PATH` 뿐).
- O4: batch 테스트에서 에코 영역 알림을 `*Messages*` 로 관찰한다고 명시했다(`acceptance.md` § 0.1).
- O6: 경로 허용 목록·디렉터리 가두기를 두지 않는다는 결정을 Out of Scope 에 명시했다.

이번 개정에서 Emacs 30.2 `--batch -Q` 로 다음을 측정했다.

- `(json-parse-string "{\"project\":[],\"session\":{},\"a\":null,\"b\":false,\"c\":true,\"payload\":{}}" :object-type 'alist :array-type 'array :null-object :null :false-object :false)`
  → `((project . []) (session) (a . :null) (b . :false) (c . t) (payload))`. 같은 옵션에서 `{"k":1,"k":2}` → `((k . 1) (k . 2))`(중복 키 검사 가능).
- `(message "hello")` 뒤 `(current-message)` 는 `nil` 이다. 같은 문자열을 두 번 `message` 하면 `*Messages*` 에는 `hello [2 times]` 한 줄로 합쳐진다.

### v0.1.0 (2026-09-27)

최초 작성. 근거 자료는 같은 디렉터리의 `research.md`(4개 조사 렌즈 통합)이며,
plan 단계에서 다음을 직접 측정해 보강했다.

- Emacs 30.2 에서 `(require 'server)` 뒤 `server-eval-args-left` 가 바인딩됨을 확인.
- emacsclient 는 문자열 평가 결과를 stdout 에 `"ok"` + 개행(따옴표 포함, 내부 `"` 는 `\"`)으로
  출력하고 exit 0, elisp 오류는 stderr 에 `*ERROR*: <메시지>`(개행 없음)로 출력하고 exit 1,
  소켓을 찾지 못하면 stderr 에 `can't find socket` 계열 문구를 출력하고 exit 1 이다.
- emacsclient `--timeout=N` 은 서버가 오래 걸리는 평가를 하는 동안 호출을 끊지 않았다
  (4초 평가 + `--timeout=1` → 4초 뒤 exit 0). 서버 평가가 사용자 입력을 기다리면 호출은 무기한 멈췄다.
  따라서 호출 시간 상한은 CLI 가 직접 강제해야 한다(REQ-AIPC-014).

## 1. 개요

### 1.1 목적

Claude Code, Codex 같은 코딩 에이전트가 작업 중에 사용자의 실행 중인 Emacs 에 "알림을 띄워라",
"이 파일을 열어 보여라", "이 위치로 가라", "산출물이 생겼다", "작업이 끝났다"를 전달할 수 있게 한다.

- **데이터 평면은 파일이다.** 문서·diff·로그 같은 큰 내용은 에이전트가 파일로 쓰고, 이벤트에는 경로만 담는다.
- **제어 평면은 emacsclient 다.** Emacs 쪽 진입점은 정해진 수신 함수 하나뿐이며, 에이전트는 임의의 elisp 를 호출하지 않는다.
  이벤트 안의 어떤 값도 elisp·셸 코드로 평가되지 않는다.
- **IPC 는 최선 노력(best-effort)이다.** Emacs 가 없거나 응답하지 않아도 에이전트의 본 작업은 실패하지 않아야 하며,
  CLI 는 정해진 시간 안에 종료 코드로 결과만 알린다.

에이전트가 부르는 Go CLI `imoogi-agent` 는 이 저장소의 일곱 번째 CLI 이며, Emacs **안으로** 호출하는 첫 CLI 다
(기존 CLI 는 모두 Emacs 가 하위 프로세스로 실행한다).

### 1.2 범위

- Emacs 수신 모듈 `modules/development/30-agent.el`: 수신 진입점, 검증, 5개 이벤트 처리, 알림/로그 UI.
- Go CLI `cmd/imoogi-agent`(+ `internal/` 패키지): 하위 명령, 이벤트 조립, 이벤트 파일 전송, emacsclient 탐색, 종료 코드.
- 빌드·부팅·설치 연결: `Makefile`, `boot.el`, 모듈 로드 순서 테스트, `scripts/install.sh` 링크.

### 1.3 용어

| 용어 | 뜻 |
|------|----|
| 이벤트(event) | CLI 가 만들어 Emacs 로 보내는 JSON 문서 한 개. 공통 봉투(envelope) + 유형별 `payload` |
| 이벤트 파일 | CLI 가 이벤트 JSON 을 써 두는 임시 파일. 경로만 emacsclient 인자로 전달된다 |
| 수신기(receiver) | Emacs 쪽 진입점 `imoogi-agent-receive-file` / `imoogi-agent-receive-json` 과 그 뒤의 검증·처리 |
| 상태 문자열 | 수신기가 돌려주는 결과. `"ok"` 또는 `"error:<reason>"` |
| payload 경로 | `open-file.path`, `goto-location.path`, `artifact-created.path`, `task-finished.artifact` |
| 전달 실패 | 이벤트가 Emacs 수신기에 닿지 못한 모든 경우(종료 코드 1) |
| 거부 | 이벤트가 Emacs 에 닿았으나 수신기가 받아들이지 않은 경우(종료 코드 3) |

## 2. 이벤트 프로토콜 (계약)

이 절의 식별자·필드명·코드 값은 사용자가 확정한 외부 계약이다. 구현 방법은 `plan.md` 가 다룬다.

### 2.1 공통 봉투

```json
{"version":"1","type":"message","timestamp":"2026-09-27T10:47:00+09:00",
 "project":"imoogi-emacs","session":"s-42","payload":{"text":"빌드 완료"}}
```

| 필드 | 필수 | 형식 | 의미 |
|------|------|------|------|
| `version` | 예 | 문자열, 정확히 `"1"` | 프로토콜 버전 |
| `type` | 예 | 아래 5개 중 하나 | 이벤트 유형 |
| `timestamp` | 예 | 비어 있지 않은 문자열 | 발신 시각. 해석하지 않고 검증만 한다 |
| `project` | 아니오 | 문자열(없음·`null` 허용) | 표시·로그 전용. 프로젝트 조회에 쓰지 않는다 |
| `session` | 아니오 | 문자열(없음·`null` 허용) | 표시·로그 전용 |
| `payload` | 예 | 객체 | 유형별 내용 |

정의되지 않은 추가 필드는 봉투·payload 어디에 있든 무시한다(버전 1 안에서의 전방 호환).

선택 필드(`project`, `session`, § 2.2 의 선택 필드)는 키가 없거나 값이 `null` 일 때만 "없음"이다. 그 밖의 값은
정해진 형식이어야 하며, 빈 배열 `[]`·빈 객체 `{}`·숫자·`true`/`false` 처럼 형식이 다른 값은 `bad-field` 다(REQ-AIPC-002.4).

### 2.2 이벤트 유형과 payload

| `type` | payload 필수 필드 | payload 선택 필드 | 기본 동작 |
|--------|-------------------|-------------------|-----------|
| `message` | `text`: 비어 있지 않은 문자열 | — | 에코 영역 알림 + 로그 |
| `open-file` | `path`: 문자열 | — | 파일을 다른 창에 표시 |
| `goto-location` | `path`: 문자열, `line`: 1 이상 정수 | `column`: 1 이상 정수(기본 1) | 파일 표시 + 해당 위치로 이동 |
| `artifact-created` | `path`: 문자열 | `artifactType`: 문자열, `title`: 문자열 | 알림 + 파일 표시 |
| `task-finished` | `status`: `"success"` 또는 `"failed"` | `summary`: 문자열, `artifact`: 문자열(경로) | 완료/실패 알림(+ 산출물 표시) |

`line`·`column` 은 1부터 센다(컴파일러 출력 관례).

### 2.3 수신기 상태 문자열

수신기는 항상 상태 문자열 하나를 돌려준다. `<reason>` 은 다음 중 하나다.

| reason | 조건 |
|--------|------|
| `too-large` | 이벤트 파일 크기 또는 JSON 문자열 바이트 수가 1 MiB(1,048,576 바이트) 초과 |
| `unreadable` | 이벤트 파일이 없거나, 원격이거나, 상대 경로이거나, 일반 파일이 아니거나, 읽을 수 없음 |
| `parse` | JSON 문법 오류(깊이 초과·UTF-8 오류 포함) |
| `trailing-content` | 첫 JSON 값 뒤에 공백 외 내용이 남음 |
| `duplicate-key` | 봉투 또는 payload 객체 안에 같은 키가 두 번 이상 |
| `bad-version` | `version` 이 없거나 `"1"` 이 아님 |
| `unknown-type` | `type` 이 없거나 5개 유형이 아님 |
| `bad-field` | 필수 필드 누락, 형식 불일치(선택 필드 포함, `[]`·`{}` 포함), 허용되지 않는 `null`, 값 범위 위반 |
| `bad-path` | payload 경로가 경로 정책(REQ-AIPC-009)을 통과하지 못함 |
| `handler` | 처리 중 예상하지 못한 오류 |
| `not-loaded` | 실행 중인 Emacs 에 수신 함수가 정의되어 있지 않음. 수신기가 아니라 CLI 의 고정 평가식이 직접 돌려준다(REQ-AIPC-012) |

### 2.4 CLI 종료 코드

| 코드 | 의미 |
|------|------|
| 0 | 수신기가 `"ok"` 를 돌려줌(이벤트 수락) |
| 1 | 전달 실패: emacsclient 없음/실행 불가, 서버 연결 실패, 시간 상한 초과, 이벤트 파일 생성·쓰기 실패 |
| 2 | 잘못된 요청: 인자·옵션·payload 값이 전송 전에 CLI 검증을 통과하지 못함 |
| 3 | 프로토콜 오류: Emacs 에 닿았으나 거부됨(`"error:…"`, `*ERROR*:` 출력, 알 수 없는 출력) |

### 2.5 CLI 명령 형식

```text
imoogi-agent message TEXT
imoogi-agent open-file PATH
imoogi-agent goto PATH LINE [COL]
imoogi-agent artifact PATH [--type T] [--title T]
imoogi-agent finish success|failed [SUMMARY] [--artifact PATH]
imoogi-agent --version

공통 옵션: --project P  --session S  --timeout SECONDS
```

옵션은 위치 인자의 앞뒤 어디에나 올 수 있고, `--` 뒤의 인자는 모두 위치 인자로 취급한다.

### 2.6 산출물 파일 관례 (참고)

에이전트가 산출물을 쓸 때 권장 위치는 `<project-notes>/artifacts/agent/YYMMDD-HHMMSS-{type}.{ext}` 이다.
파일은 에이전트가 직접 쓴다. MVP 에서 Emacs 는 이 관례를 강제하지도, 디렉터리를 만들지도 않는다(§ 4 참조).

## 3. Requirements (GEARS)

요구사항 15개, 다섯 묶음. 각 요구사항의 정규 문장은 GEARS 영어 형식으로 쓰고, 이어지는 한국어 항목이
세부 조건을 정한다. `[Ubiquitous — negated]` 는 `shall not` 형식이다.

### 3.1 프로토콜과 검증 (Emacs 수신기)

#### REQ-AIPC-001 [Ubiquitous]

**The Emacs receiver shall expose exactly two entry points, `imoogi-agent-receive-file` (event-file path) and `imoogi-agent-receive-json` (JSON string), which share one validation and dispatch path and always return a status string.**

1. 두 진입점은 같은 입력에 대해 같은 상태 문자열과 같은 부수 효과를 낸다(차이는 입력을 얻는 방법뿐).
2. 반환값은 항상 `"ok"` 또는 `"error:<reason>"`(§ 2.3)이며, 진입점 밖으로 elisp 오류를 내보내지 않는다.
3. `imoogi-agent-receive-file` 은 이벤트 파일을 읽기만 하며 수정·삭제하지 않는다.

#### REQ-AIPC-002 [Ubiquitous]

**The receiver shall accept an event only when it conforms to the envelope and payload schema of § 2.1–§ 2.2.**

1. JSON `null` 은 선택 필드에서만 "없음"으로 취급하고, 필수 필드의 `null` 은 `bad-field` 다.
2. JSON `false` 는 어떤 필드에서도 참으로 해석되지 않는다.
3. 정의되지 않은 추가 필드는 무시하며 거부 사유가 되지 않는다.
4. 선택 필드는 키가 없거나 값이 `null` 이면 없음이고, 그 밖의 값은 § 2.1–§ 2.2 의 형식을 따라야 한다.
   빈 배열 `[]`, 빈 객체 `{}`, 숫자, `true`/`false` 처럼 형식이 다른 값은 없음으로 취급하지 않고 `bad-field` 로 거부한다
   (예: `"project":[]`, `"session":{}`, `"summary":[]`, `"title":{}`, `"project":false`).

#### REQ-AIPC-003 [When — event-detected]

**When an input is detected to violate the size limit, the event-file preconditions, JSON syntax, key uniqueness, or the schema, the receiver shall reject it with the matching `error:<reason>` of § 2.3, perform no dispatch and no display, and append one rejection line to the log.**

1. 크기 검사는 내용을 해석하기 전에 한다(파일은 읽기 전에 크기를 본다).
2. 이벤트 파일은 절대 경로이고 원격이 아니며 일반 파일이어야 한다. 아니면 `unreadable`.
3. 첫 JSON 값 뒤에 공백 외 내용이 있으면 `trailing-content`.
4. 봉투 또는 payload 객체의 키 중복은 `duplicate-key` 다(어느 값을 쓸지 고르지 않는다).
5. 여러 위반이 겹치면 이 순서의 첫 위반을 보고한다: 파일 조건 → 크기 → 문법 → 후행 내용 → 키 중복 → 버전 → 유형 → 필드 → 경로.
   파일 조건이 크기보다 먼저인 이유: 원격 경로의 크기를 묻는 것만으로도 원격 연결이 열리기 때문이다.

### 3.2 처리와 UI (Emacs 수신기)

#### REQ-AIPC-004 [When]

**When a valid `message` or `task-finished` event is received, the receiver shall show one notification line in the echo area and append one line to the log buffer.**

1. `message`: `text` 를 그대로 보인다. `%` 같은 문자는 서식 지시자로 해석되지 않고 글자 그대로 나타난다.
2. `task-finished` + `success`: `✓ Agent task finished` 로 시작하고, `summary` 가 있으면 뒤에 붙인다.
3. `task-finished` + `failed`: `✗ Agent task failed` 로 시작하고, `summary` 가 있으면 뒤에 붙인다.
4. `task-finished` 에 `artifact` 가 있으면 알림에 더해 그 파일을 REQ-AIPC-005 방식으로 표시한다.
5. 알림 줄(이 요구사항과 REQ-AIPC-006)과 로그 줄(REQ-AIPC-007)에 들어가는 이벤트 유래 값 — `text`, `summary`, `title`,
   `artifactType`, `project`, `session`, `path` 에서 얻은 파일 이름 — 안의 줄바꿈은 `⏎`(U+23CE) 한 글자로 바꾼다.
   CRLF 는 한 번의 줄바꿈으로 세어 `⏎` 하나가 되고, 단독 LF·단독 CR 도 각각 `⏎` 하나가 된다.
   따라서 알림은 언제나 한 줄이고 로그 항목은 언제나 한 줄이다.

#### REQ-AIPC-005 [When]

**When a valid `open-file` or `goto-location` event is received, the receiver shall display the file in a window other than the selected window without selecting it.**

1. 사용자의 선택된 창, 선택된 프레임, 선택된 창의 point 는 바뀌지 않는다.
2. 파일이 선택되지 않은 창에 이미 보이면 그 창을 다시 쓴다. 선택된 창에만 보이면 다른 창에 한 번 더 표시한다.
3. `goto-location`: 표시한 창의 point 를 `line`·`column`(1부터) 위치로 옮긴다. 줄이 버퍼 끝을 넘으면 버퍼 끝
   (`point-max`)이 있는 줄로, 열이 줄 끝을 넘으면 그 줄의 끝으로 맞춘다.
4. 파일은 payload 경로의 심볼릭 링크를 해석한 실제 경로로 연다.

#### REQ-AIPC-006 [When]

**When a valid `artifact-created` event is received, the receiver shall show `Agent artifact created: <title>` in the echo area, append one line to the log, and display the file as in REQ-AIPC-005.**

1. `title` 이 없으면 파일 이름(경로의 마지막 구성 요소)을 쓴다.
2. `artifactType` 은 로그에만 기록한다.
3. 알림 줄의 줄바꿈 처리는 REQ-AIPC-004.5 를 따른다.

#### REQ-AIPC-007 [Ubiquitous]

**The receiver shall record every accepted and every rejected event as one line appended to the buffer `*imoogi-agent*`.**

1. 한 줄에는 수신 시각, `type`, `project`, `session`, 결과(`ok` 또는 거부 reason)가 들어간다. 없거나 문자열이 아닌 값은 `-` 로 적는다.
   값 안의 줄바꿈은 REQ-AIPC-004.5 처럼 `⏎` 로 바꾸므로 한 이벤트는 정확히 한 줄이 된다.
2. 기존 줄은 지우거나 고치지 않는다(추가 전용). 버퍼는 읽기 전용이다.
3. 로그는 세션 안에서만 유지된다. 버퍼가 정리되면 다음 이벤트 때 새로 만든다(`plan.md` § 결정 D-4).

#### REQ-AIPC-008 [Ubiquitous — negated]

**The receiver shall not solicit user input, change the selected window or frame or the user's point, write to or execute any payload file, evaluate any event value as code, or interpret `project`/`session` beyond display.**

1. 파일을 여는 과정에서 간접적으로 생기는 질문도 금지한다: 큰 파일 경고, 파일 지역 변수 확인, 디스크에서 바뀐 파일 질문.
   질문이 하나라도 뜨면 서버 호출이 멈추므로(HISTORY 측정) 이 조항은 성능 문제가 아니라 기능 요건이다.
2. `project`·`session` 으로 프로젝트 노트 레지스트리를 찾거나 디렉터리를 만들지 않으며, 수신 모듈은
   `modules/project/26-project-notes.el` 에 의존하지 않는다.

### 3.3 경로 안전

#### REQ-AIPC-009 [When — event-detected]

**When a payload path is detected to be relative, remote, non-existent, or — after resolving symbolic links — not a regular file, the receiver shall reject the whole event with `error:bad-path` and display nothing.**

1. 일반 파일이 아닌 것: 디렉터리, 장치 파일, FIFO, 소켓, 해석 결과가 없는 심볼릭 링크.
2. 원격 판정은 연결을 새로 열지 않는 방식이어야 한다(TRAMP 경로 거부).
3. `task-finished.artifact` 가 정책을 통과하지 못하면 알림도 띄우지 않는다(이벤트 전체 거부).
4. 통과한 경로는 열어서 보여 주기만 한다.

### 3.4 CLI 와 전송

#### REQ-AIPC-010 [Ubiquitous]

**The `imoogi-agent` CLI shall accept the command grammar of § 2.5 and build exactly one event per invocation with `version` `"1"`, an RFC 3339 `timestamp`, and the payload the subcommand defines.**

1. 하위 명령 → 유형: `message`→`message`, `open-file`→`open-file`, `goto`→`goto-location`,
   `artifact`→`artifact-created`, `finish`→`task-finished`.
2. 상대 경로 인자(`PATH`, `--artifact`)는 CLI 의 현재 작업 디렉터리 기준 절대 경로로 바꿔서 보낸다.
   파일 존재 여부 판단은 Emacs 수신기(REQ-AIPC-009)가 단독으로 한다.
3. `--project`·`--session` 이 주어지면 봉투에 넣고, 없으면 필드를 생략한다. 선택 payload 필드도 주어질 때만 넣는다.
4. 만들어진 JSON 에는 중복 키가 없다.
5. 성공 시 stdout 에 아무것도 쓰지 않는다. `--version` 은 버전 한 줄을 쓰고 0 으로 끝난다.

#### REQ-AIPC-011 [When — event-detected]

**When an invocation is detected to be invalid before sending, the CLI shall exit with code 2, write one diagnostic line to stderr, create no event file, and not invoke emacsclient.**

잘못된 호출의 예: 하위 명령 없음·알 수 없음, 위치 인자 부족·초과, 알 수 없는 옵션, 옵션 값 누락,
빈 `TEXT`, 1 이상 정수가 아닌 `LINE`·`COL`, `success`/`failed` 가 아닌 상태, 양의 정수가 아닌 시간 상한,
완성된 이벤트가 1 MiB 초과.

#### REQ-AIPC-012 [Ubiquitous]

**The CLI shall deliver each event by writing it to an owner-only temporary file with a CLI-generated name and invoking emacsclient directly, without a shell, with a fixed evaluation expression that receives the file path as a separate argument through `server-eval-args-left`.**

1. 이벤트 파일 권한은 소유자 읽기·쓰기(0600)이고, 경로는 절대 경로다.
2. 평가식은 고정 문자열이며 이벤트 내용이나 경로를 문자열로 이어 붙이지 않는다.
3. 평가식은 다른 어떤 평가보다 먼저 경로 인자를 소비한다. 실행 중인 Emacs 에 수신 함수가 정의되어 있지 않아도
   경로가 Lisp 로 평가되지 않으며, 이때 결과는 `error:not-loaded` 거부(종료 코드 3)다.
4. CLI 는 emacsclient 가 끝나거나 강제 종료된 뒤 모든 경로(성공·거부·전달 실패·시간 초과)에서 이벤트 파일을 지운다.
   Emacs 는 이벤트 파일을 지우지 않는다.

#### REQ-AIPC-013 [Ubiquitous]

**The CLI shall locate emacsclient in this order and use the first match: `$EMACSCLIENT` (which must be executable, otherwise delivery fails), `PATH`, `/Applications/Emacs-${EMACS_VERSION:-31.1}.app/Contents/MacOS/bin/emacsclient`, `/Applications/Emacs.app/Contents/MacOS/bin/emacsclient`, then `/Applications/Emacs-*.app/Contents/MacOS/bin/emacsclient`.**

이 순서는 `scripts/imoogi-editor` 의 `find_emacsclient` 와 같다. 두 구현의 동기화 위험은 `plan.md` 가 다룬다.

#### REQ-AIPC-014 [Ubiquitous]

**The CLI shall map every outcome to the exit codes of § 2.4 and shall terminate the emacsclient call no later than the time limit, default 5 seconds, overridable by `--timeout` or `IMOOGI_AGENT_TIMEOUT`.**

1. emacsclient 가 exit 0 이고 stdout 이 Lisp 문자열 `"ok"` 이면 0.
2. stdout 이 `"error:<reason>"` 이거나, stderr 에 `*ERROR*:` 가 있거나, emacsclient 가 exit 0 이지만 출력이
   상태 문자열로 해석되지 않으면 3. 거부 reason 은 stderr 한 줄로 알린다.
3. emacsclient 를 찾지 못함, 실행 불가, 연결 실패(`*ERROR*:` 없는 비정상 종료), 시간 상한 도달, 이벤트 파일 생성·쓰기 실패는 1.
4. 시간 상한은 emacsclient 자체 옵션에 기대지 않고 CLI 가 직접 강제한다(HISTORY 측정: `--timeout` 은 느린 평가를 끊지 못함).
   CLI 는 상한 + 1초 안에 끝난다.
5. `--timeout` 이 환경 변수보다 우선한다.

### 3.5 빌드·부팅·설치

#### REQ-AIPC-015 [Ubiquitous]

**The repository shall build, load, and install the bridge through its existing conventions without adding any Emacs package or third-party Go module.**

1. `make build-agent` 가 `bin/imoogi-agent` 를 만들고, `build-all`·`.PHONY`·`help` 에 포함된다.
2. `modules/development/30-agent.el` 은 `imoogi-require` 로 시작해 `(provide 'imoogi-agent)` 로 끝나고,
   `boot.el` 모듈 목록과 `tests/module-layout-test.el` 의 기대 로드 순서에 `29-org-roam` 다음으로 들어간다.
   오프라인 부팅에서 `:error` 경고를 내지 않는다.
3. `scripts/install.sh` 는 `~/.local/bin/imoogi-agent` 를 `imoogi-editor` 와 같은 방식(저장소 링크 기준 심볼릭 링크)으로 만들고,
   셸 테스트가 이를 검증한다.
4. `go.mod`/`go.sum`, `go.work`, `packages.el`, `packages.lock`, `vendor/` 는 바뀌지 않는다(망분리 제약).

## 4. Out of Scope

이 SPEC 이 만들지 않는 것. 원 명세의 후속 단계(2~5)는 비목표로만 기록한다.

### Out of Scope — MVP 이후 이벤트

- `progress`(진행률) 이벤트.
- `show-diff`(diff 표시) 이벤트.
- 그 밖에 § 2.2 의 5개 이외의 모든 이벤트 유형.

### Out of Scope — 전송 방식

- 상주형 TCP 또는 Unix 소켓 리스너, JSON-RPC, ACP(Agent Client Protocol), 토큰 스트리밍.
- `server-name`·`server-socket-dir`·TCP 서버 설정의 추가나 변경. CLI 는 emacsclient 기본 소켓을 쓴다
  (`EMACS_SOCKET_NAME` 같은 환경 변수는 emacsclient 가 그대로 상속할 뿐 CLI 가 관리하지 않는다).
- 전송 재시도, 대기열, 이벤트 재전송.

### Out of Scope — 방향과 수명 주기

- Emacs → 에이전트 방향의 요청·응답, 사용자 승인 요청.
- 에이전트 세션 수명 주기 관리(시작·종료·목록), 에이전트 프로세스 관리.
- `project`·`session` 값을 이용한 프로젝트 조회, 프로젝트 노트 레지스트리 연동, `26-project-notes.el` 변경.

### Out of Scope — 원격과 플랫폼

- 원격 호스트, TRAMP 경로, SSH 포워딩을 통한 전달.
- Emacs 30.1 미만(`server-eval-args-left` 없음).
- REQ-AIPC-013 목록 밖의 플랫폼별 emacsclient 탐색 경로.

### Out of Scope — 경로 허용 목록

- payload 경로의 허용 목록(allowlist)이나 특정 디렉터리로 가두는(containment) 제한. MVP 수신기는 REQ-AIPC-009 를 통과한
  모든 로컬 일반 파일을 연다. 이는 사용자 결정이다.
- 근거: 발신자는 사용자와 같은 권한으로 동작하는 에이전트라 그 파일을 이미 직접 읽을 수 있다. emacsclient 기본 소켓은
  사용자 전용 디렉터리에 있어 다른 사용자가 이벤트를 보낼 수 없다. 수신기는 파일을 보여 주기만 하고 쓰거나 실행하지 않는다
  (REQ-AIPC-008). 허용 목록을 두려면 프로젝트 조회가 필요한데, 이는 REQ-AIPC-008.2 가 금지한 `26-project-notes` 의존을 부른다.

### Out of Scope — 산출물과 표시 장치

- Emacs 가 산출물 파일을 쓰거나, `artifacts/agent/` 디렉터리를 만들거나, 파일 이름 관례를 강제하는 일.
- 데스크톱 알림(`notifications-notify`, `alert`), 소리, 파일로 남는 영구 로그.

### Out of Scope — 문서 갱신

- "여섯 개의 CLI" 서술 등 프로젝트 문서 갱신. sync 단계에서 처리한다(`plan.md` § 7).

## 5. Constraints

- **망분리**: 새 elpa 패키지와 새 서드파티 Go 모듈을 추가하지 않는다. 부팅 경로에 네트워크 의존을 넣지 않는다(AGENTS.md § 0).
- **Emacs 버전**: 30.1 이상. 이 머신에는 30.2(`Emacs.app`)와 31.1(`Emacs-31.1.app`)이 함께 설치되어 있다.
- **JSON**: Emacs 쪽은 내장 파서만 쓴다. Go 쪽은 표준 라이브러리만 쓴다.
- **사용자 상호작용 금지**: 수신기는 어떤 경로로도 사용자 입력을 기다리지 않는다(REQ-AIPC-008).
- **작업 트리**: 커밋되지 않은 `modules/project/26-project-notes.el`, `tests/project-notes-test.el`, `CLAUDE.md` 수정분을
  건드리지 않으며, 구현이 그 파일들에 의존하지 않는다.

## 6. 변경 파일 (Brownfield Delta)

| 표시 | 경로 | 내용 |
|------|------|------|
| [NEW] | `modules/development/30-agent.el` | 수신기, 검증, 5개 이벤트 처리, 알림·로그 UI |
| [NEW] | `tests/agent-test.el` | ERT 테스트 |
| [NEW] | `cmd/imoogi-agent/main.go` | 얇은 `main()` + 테스트 가능한 `run(...)` |
| [NEW] | `cmd/imoogi-agent/main_test.go` | CLI 테스트(가짜 emacsclient) |
| [NEW] | `internal/agentipc/` | 이벤트 조립, 인자 해석, emacsclient 탐색, 전송, 결과 분류 + 테스트 |
| [MODIFY] | `boot.el` | 모듈 목록에 `"development/30-agent"` 추가 |
| [MODIFY] | `tests/module-layout-test.el` | 기대 로드 순서에 `"30-agent"` 추가 |
| [MODIFY] | `Makefile` | `build-agent` 타깃, `build-all`, `.PHONY`, `help` |
| [MODIFY] | `scripts/install.sh` | `~/.local/bin/imoogi-agent` 링크 |
| [MODIFY] | `tests/setup-toolchain-test.sh` | 설치 링크 검증 |
| [UNCHANGED] | `go.mod`, `go.sum`, `go.work`, `packages.el`, `packages.lock`, `vendor/` | 망분리 제약 |
| [UNCHANGED] | `modules/project/26-project-notes.el`, `modules/general/13-system.el`, `scripts/imoogi-editor` | 의존·수정 없음 |

## 7. Traceability

| REQ | AC |
|-----|----|
| REQ-AIPC-001 | AC-AIPC-001, AC-AIPC-006, AC-AIPC-010 |
| REQ-AIPC-002 | AC-AIPC-007 |
| REQ-AIPC-003 | AC-AIPC-006, AC-AIPC-007, AC-AIPC-008 |
| REQ-AIPC-004 | AC-AIPC-001, AC-AIPC-005 |
| REQ-AIPC-005 | AC-AIPC-002, AC-AIPC-003 |
| REQ-AIPC-006 | AC-AIPC-004 |
| REQ-AIPC-007 | AC-AIPC-001, AC-AIPC-006 |
| REQ-AIPC-008 | AC-AIPC-002, AC-AIPC-003, AC-AIPC-010 |
| REQ-AIPC-009 | AC-AIPC-009 |
| REQ-AIPC-010 | AC-AIPC-011 |
| REQ-AIPC-011 | AC-AIPC-012 |
| REQ-AIPC-012 | AC-AIPC-013, AC-AIPC-015 |
| REQ-AIPC-013 | AC-AIPC-014 |
| REQ-AIPC-014 | AC-AIPC-014, AC-AIPC-015 |
| REQ-AIPC-015 | AC-AIPC-015 |
