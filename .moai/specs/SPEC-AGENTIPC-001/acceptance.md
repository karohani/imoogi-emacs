# SPEC-AGENTIPC-001 — Acceptance Criteria

인수 기준 15개. 각 기준은 Given-When-Then 시나리오, 대응 REQ, 그리고 결과가 참/거짓으로 갈리는 검증 명령을 가진다.
요구사항 문장은 `spec.md` § 3 이 정본이며 여기서는 되풀이하지 않는다.

## 0. 검증 명령 공통 형식

### 0.1 ERT 선택 실행

`tests/run.sh` 는 선택자를 받지 않는다. 다음 형식은 `tests/run.el` 의 전체 부팅(boot-health 포함)을 그대로 거친 뒤
선택자에 맞는 테스트만 실행한다. plan 단계에서 기존 테스트 `^imoogi-module-reload-preserves` 로 동작을 확인했다
(`Ran 3 tests, 3 results as expected, 0 unexpected`, exit 0).

```bash
ert_sel() {
  IMOOGI_TEST_USER_DIR="$(mktemp -d)" "${EMACS:-emacs}" --batch -Q \
    --eval "(advice-add 'ert-run-tests-batch-and-exit :filter-args (lambda (_) (list \"$1\")))" \
    -l tests/run.el
}
```

통과 조건: exit 0 이고 출력 끝줄이 `Ran N tests, N results as expected, 0 unexpected` 이며 N ≥ 1.

ERT 테스트 이름은 `imoogi-agent-acNN-` 로 시작한다(NN = AC 번호 두 자리). Go 테스트 이름은 `TestACNN` 으로 시작한다.

창 표시 기준(AC-AIPC-002~005)은 `--batch` 의 초기 프레임(80x25, 창 1개)에서도 검증된다. plan 단계에서 `plan.md` D-5 의
`display-buffer` action 을 batch Emacs 30.2 에 적용해 확인했다: 창이 2개로 나뉘고, 돌려받은 창은 선택된 창이 아니며,
`selected-window` 는 그대로였다. 따라서 테스트는 창을 미리 나누지 않아도 된다. 각 테스트는 시작 전에
`delete-other-windows` 로 창을 1개로 되돌려 서로 영향을 주지 않게 한다.

에코 영역 관찰(AC-AIPC-001·004·005): `--batch` 에서는 `(current-message)` 가 `message` 뒤에도 `nil` 이므로 쓰지 않는다.
대신 `message` 가 기록하는 `*Messages*` 버퍼를 본다. 이 문서에서 "에코 영역 알림이 X 이다"는 "수신 호출 동안 `*Messages*` 에
X 와 같은 줄이 정확히 하나 추가된다"를 뜻한다. 테스트는 호출 전에 `*Messages*` 를 비우며, 같은 호출 동안 다른 줄(파일을 열 때
모드·훅이 남기는 메시지 등)이 함께 추가되어도 된다. "X 로 시작한다"는 "추가된 줄 가운데 X 로 시작하는 줄이 정확히 하나다"를,
이어지는 "Y 를 포함한다"는 바로 그 줄에 대한 판정을 뜻한다(마지막 줄인지는 보지 않는다, v0.1.2). Emacs 는 같은 문자열이 연달아 기록되면
`X [2 times]` 처럼 한 줄로 합치므로, 테스트는 각 수신 호출 전에 `*Messages*` 를 비운다(`inhibit-read-only` 로 `erase-buffer`).
(Emacs 30.2 batch 측정: `current-message` → `nil`, `hello` 두 번 → `hello [2 times]`.)

### 0.2 Go 선택 실행

```bash
go test ./cmd/imoogi-agent/... ./internal/agentipc/... -run 'TestACNN' -count=1
```

통과 조건: exit 0, 출력에 `ok` 이고 `no tests to run` 이 없음.

### 0.3 가짜 emacsclient 의 출력 모양

Go 테스트의 가짜 emacsclient 는 plan 단계에서 실제 emacsclient 30.2 로 측정한 바이트 모양을 그대로 흉내 낸다.

| 경우 | stdout | stderr | exit |
|------|--------|--------|------|
| 수락 | `"ok"` + `\n` | (없음) | 0 |
| 거부(수신기) | `"error:bad-path"` + `\n` | (없음) | 0 |
| elisp 오류 | (없음) | `*ERROR*: boom` (개행 없음) | 1 |
| 연결 실패 | (없음) | `…: can't find socket; have you started the server?` 외 2줄 | 1 |

---

## 1. Emacs 수신기

### AC-AIPC-001 — message 이벤트가 알림과 로그로 나타난다 (REQ-AIPC-001, REQ-AIPC-004, REQ-AIPC-007)

**Given** 30-agent 모듈이 로드되어 있고, `text` 가 `50% 완료 %s` 인 유효한 `message` 이벤트를 담은 이벤트 파일이 있을 때
**When** `imoogi-agent-receive-file` 에 그 파일 경로를 넘기면
**Then** 반환값은 `"ok"` 이고, 에코 영역에 `50% 완료 %s` 가 글자 그대로 나타나며(서식 오류 없음),
`*imoogi-agent*` 에 `message`·project·session·`ok` 를 담은 줄이 정확히 하나 늘고, 버퍼는 읽기 전용이며,
같은 이벤트를 `imoogi-agent-receive-json` 에 문자열로 넘겨도 같은 반환값과 같은 로그 줄 형식이 나온다.
이벤트 파일의 내용과 수정 시각은 호출 전후가 같다.
**And** `text` 가 `첫 줄` LF `둘째 줄` CRLF `셋째 줄` CR `넷째 줄`(JSON `"첫 줄\n둘째 줄\r\n셋째 줄\r넷째 줄"`)이고 `project` 가
`p` LF `q`(JSON `"p\nq"`)인 message 이벤트를 받으면, 반환값은 `"ok"`, 에코 영역 알림은 정확히 `첫 줄⏎둘째 줄⏎셋째 줄⏎넷째 줄`
(수신 호출 동안 `*Messages*` 에 추가된 줄, 줄바꿈 없음)이며, `*imoogi-agent*` 의 줄 수는 정확히 1 늘고 그 줄에 `p⏎q` 가 들어 있다.

- 검증: `ert_sel "^imoogi-agent-ac01-"`

### AC-AIPC-002 — open-file 은 초점을 빼앗지 않는다 (REQ-AIPC-005, REQ-AIPC-008)

**Given** 창이 하나인 프레임에서 사용자가 버퍼 A 의 point 10 에 있고, 유효한 일반 파일 F 가 있을 때
**When** `open-file`(path = F)을 수신하면
**Then** 반환값은 `"ok"` 이고, F 가 선택되지 않은 다른 창에 보이며, `selected-window`·`selected-frame` 은 호출 전과 같은 객체이고,
그 창의 버퍼는 여전히 A 이며 point 는 10 이다.
**And** F 가 이미 다른 창에 보이는 상태에서 다시 수신하면 새 창을 만들지 않고 그 창을 다시 쓴다(창 개수 불변).

- 검증: `ert_sel "^imoogi-agent-ac02-"`

### AC-AIPC-003 — goto-location 은 대상 창의 point 만 옮긴다 (REQ-AIPC-005, REQ-AIPC-008)

**Given** 5줄짜리 일반 파일 F 와, 사용자가 버퍼 A 의 point 10 에 있는 상태에서
**When** 다음을 차례로 수신하면
- (a) `line` 3, `column` 4
- (b) `line` 3, `column` 없음
- (c) `line` 99
- (d) `line` 2, `column` 999

**Then** 각 경우 반환값은 `"ok"` 이고 F 를 보이는 창의 point 가
(a) 3번째 줄 4번째 열(0 기준 열 3), (b) 3번째 줄 첫 열, (c) `point-max` 가 있는 줄의 첫 열, (d) 2번째 줄 끝에 있으며,
사용자의 선택된 창과 그 point(버퍼 A, 10)는 매번 그대로다.

- 검증: `ert_sel "^imoogi-agent-ac03-"`

### AC-AIPC-004 — artifact-created 는 제목과 함께 알리고 파일을 보여 준다 (REQ-AIPC-006)

**Given** 일반 파일 `/…/260927-104700-report.md` 가 있을 때
**When** (a) `title` = `주간 보고`, `artifactType` = `report` 로 수신하고, (b) `title` 없이 수신하면
**Then** 둘 다 반환값은 `"ok"` 이고, 에코 영역 알림은 (a) `Agent artifact created: 주간 보고`,
(b) `Agent artifact created: 260927-104700-report.md` 이며, 파일은 AC-AIPC-002 와 같은 방식(선택되지 않은 창)으로 보이고,
(a) 의 로그 줄에는 `report` 가 들어 있다.

- 검증: `ert_sel "^imoogi-agent-ac04-"`

### AC-AIPC-005 — task-finished 는 성공/실패를 구분해 알린다 (REQ-AIPC-004)

**Given** 유효한 일반 파일 R 이 있을 때
**When** 다음을 수신하면
- (a) `status` = `success`, `summary` = `테스트 42개 통과`
- (b) `status` = `failed`, `summary` = `빌드 실패`
- (c) `status` = `success`, summary 없음, `artifact` = R

**Then** 모두 반환값은 `"ok"` 이고, 에코 영역 알림은
(a) `✓ Agent task finished` 로 시작해 `테스트 42개 통과` 를 포함하고,
(b) `✗ Agent task failed` 로 시작해 `빌드 실패` 를 포함하며,
(c) `✓ Agent task finished` 로 시작하고 R 이 선택되지 않은 창에 보인다.

- 검증: `ert_sel "^imoogi-agent-ac05-"`

### AC-AIPC-006 — 문법 오류·후행 내용·키 중복은 거부된다 (REQ-AIPC-001, REQ-AIPC-003, REQ-AIPC-007)

**Given** 30-agent 모듈이 로드되어 있을 때
**When** 다음 입력을 `imoogi-agent-receive-json` 과 `imoogi-agent-receive-file` 양쪽으로 넘기면
- (a) `{"version":"1",` (잘린 JSON)
- (b) `[` 10001개 (깊이 초과)
- (c) 유효한 message 이벤트 뒤에 `{}` 가 붙은 입력
- (d) `{"version":"1","type":"message","type":"open-file",…}` (봉투 키 중복)
- (e) payload 안 `"text"` 키 중복

**Then** 반환값은 (a)(b) `"error:parse"`, (c) `"error:trailing-content"`, (d)(e) `"error:duplicate-key"` 이고,
어느 경우에도 elisp 오류가 진입점 밖으로 나오지 않으며, 알림·창 변화가 없고,
`*imoogi-agent*` 에 해당 reason 을 담은 거부 줄이 하나씩 늘어난다.

- 검증: `ert_sel "^imoogi-agent-ac06-"`

### AC-AIPC-007 — 스키마 위반은 정해진 reason 으로 거부된다 (REQ-AIPC-002, REQ-AIPC-003)

**Given** 30-agent 모듈이 로드되어 있을 때
**When** 다음 이벤트를 수신하면

| # | 입력 | 기대 반환값 |
|---|------|-------------|
| a | `version` = `"2"` | `"error:bad-version"` |
| b | `version` = `1` (숫자) | `"error:bad-version"` |
| c | `type` = `"progress"` | `"error:unknown-type"` |
| d | `type` 없음 | `"error:unknown-type"` |
| e | `message` 에 `text` 없음 | `"error:bad-field"` |
| f | `text` = `""` | `"error:bad-field"` |
| g | `text` = `null` | `"error:bad-field"` |
| h | `goto-location` 의 `line` = `0` / `"3"` / `2.5` | `"error:bad-field"` |
| i | `task-finished` 의 `status` = `"done"` / `false` | `"error:bad-field"` |
| j | `timestamp` 없음 / `""` | `"error:bad-field"` |
| k | `payload` 가 배열 | `"error:bad-field"` |
| l | `project` = `null`, `session` 없음, 봉투와 payload 에 정의되지 않은 필드 `"extra":1` | `"ok"` |
| m | 버전 위반과 필드 위반이 겹침 | `"error:bad-version"` |
| n | 봉투 선택 필드가 형식이 다름: `project` = `[]` / `session` = `{}` / `project` = `false` / `session` = `3` (각각 따로) | `"error:bad-field"` |
| o | payload 선택 필드가 형식이 다름: `task-finished` 의 `summary` = `[]` / `artifact-created` 의 `title` = `{}` / `goto-location` 의 `column` = `"2"` (각각 따로) | `"error:bad-field"` |

**Then** 반환값이 표와 같고, (l) 을 뺀 모든 경우 알림·창 변화가 없다.

- 검증: `ert_sel "^imoogi-agent-ac07-"`

### AC-AIPC-008 — 크기 제한과 이벤트 파일 조건 (REQ-AIPC-003)

**Given** 30-agent 모듈이 로드되어 있을 때
**When** 다음을 수신하면
- (a) 1,048,577 바이트 이벤트 파일(내용은 유효한 JSON 이 아니어도 됨)
- (b) 1,048,577 바이트 JSON 문자열을 `imoogi-agent-receive-json` 으로
- (c) 정확히 1,048,576 바이트의 유효한 message 이벤트 파일
- (d) 존재하지 않는 이벤트 파일 경로
- (e) 상대 경로 `event.json`
- (f) 디렉터리 경로
- (g) `/ssh:host:/tmp/e.json`

**Then** (a)(b) `"error:too-large"`, (c) `"ok"`, (d)~(g) `"error:unreadable"` 이다.
(a) 에서는 파일 내용을 버퍼로 읽어 들이지 않는다(`insert-file-contents` 계열 호출 0회 — 테스트에서 호출 횟수를 센다).
(g) 에서는 원격 연결을 시도하지 않는다.

- 검증: `ert_sel "^imoogi-agent-ac08-"`

### AC-AIPC-009 — payload 경로 정책 (REQ-AIPC-009)

**Given** 임시 디렉터리에 일반 파일 F, 디렉터리 D, F 를 가리키는 링크 LF, D 를 가리키는 링크 LD, 끊긴 링크 LX 가 있을 때
**When** `open-file` 의 `path` 를 다음으로 바꿔 수신하면

| # | path | 기대 |
|---|------|------|
| a | F (절대 경로) | `"ok"`, F 표시 |
| b | LF | `"ok"`, 표시된 버퍼의 파일 이름이 F 의 실제 경로 |
| c | `F` 의 상대 표기 | `"error:bad-path"` |
| d | 존재하지 않는 절대 경로 | `"error:bad-path"` |
| e | D | `"error:bad-path"` |
| f | LD | `"error:bad-path"` |
| g | LX | `"error:bad-path"` |
| h | `/dev/null` | `"error:bad-path"` |
| i | `/ssh:host:/etc/hosts` | `"error:bad-path"` |

**And** 같은 정책이 `goto-location.path`, `artifact-created.path`, `task-finished.artifact` 에도 적용된다
(각 필드에 d 를 넣으면 `"error:bad-path"`, `task-finished` 는 알림도 뜨지 않음).
**Then** 거부된 모든 경우 창 구성과 버퍼 목록이 호출 전과 같고, 원격 연결 시도가 없다.

- 검증: `ert_sel "^imoogi-agent-ac09-"`

### AC-AIPC-010 — 수신기는 어떤 경로로도 사용자 입력을 기다리지 않는다 (REQ-AIPC-001, REQ-AIPC-008)

**Given** `read-string`, `read-from-minibuffer`, `y-or-n-p`, `yes-or-no-p`, `read-key`, `read-char`, `read-event` 가
호출되는 즉시 테스트를 실패시키도록 바꿔 둔 상태에서
**When** 다음 파일에 대해 `open-file` 과 `goto-location` 을 수신하면
- (a) `large-file-warning-threshold` 보다 큰 파일
- (b) 첫 줄에 `-*- eval: (message "x") -*-` 와 안전하지 않은 파일 지역 변수가 있는 파일
- (c) 이미 버퍼로 열려 있고 그 뒤 디스크에서 내용이 바뀐 파일

**Then** 세 경우 모두 반환값은 `"ok"` 이고, 입력 함수 호출은 0회이며, (b) 의 `eval` 지역 변수는 실행되지 않는다.
**And** 5개 이벤트 유형을 모두 처리하는 동안 `imoogi-project-notes-` 로 시작하는 함수 호출은 0회이고,
`modules/development/30-agent.el` 에 `26-project-notes`·`imoogi-project-notes` 문자열이 없다.

- 검증: `ert_sel "^imoogi-agent-ac10-"` 그리고
  `grep -c -e '26-project-notes' -e 'imoogi-project-notes' modules/development/30-agent.el` 가 `0` 을 출력.

---

## 2. CLI 와 전송

### AC-AIPC-011 — 하위 명령이 정해진 이벤트를 만든다 (REQ-AIPC-010)

**Given** 받은 이벤트 파일 내용을 기록하는 가짜 emacsclient 가 `EMACSCLIENT` 로 지정되어 있고 수락(`"ok"`)을 돌려줄 때
**When** 작업 디렉터리 W 에서 다음을 실행하면
- (a) `imoogi-agent message "안녕 %s"`
- (b) `imoogi-agent open-file notes/a.md`
- (c) `imoogi-agent goto src/x.go 12 5 --project p1 --session s1`
- (d) `imoogi-agent goto src/x.go 12`
- (e) `imoogi-agent artifact out/r.md --type report --title "주간 보고"`
- (f) `imoogi-agent finish failed "빌드 실패" --artifact log.txt`
- (g) `imoogi-agent message -- --not-a-flag`
- (h) `imoogi-agent --version`

**Then** (a)~(g) 는 exit 0, stdout 비어 있음이며, 기록된 이벤트는
- `version` `"1"`, RFC 3339 로 해석되는 `timestamp`, 기대 `type`
  (`message`/`open-file`/`goto-location`/`goto-location`/`artifact-created`/`task-finished`/`message`)을 가진다.
- 상대 경로는 `W` 기준 절대 경로(`filepath.Join(W, …)`)로 바뀌어 있다.
- (c) 에만 `project`·`session` 이 있고 나머지에는 두 키가 없다.
- (d) 에는 `column` 키가 없다. (e) 는 `artifactType`·`title` 이 있다. (f) 는 `status` `failed`, `summary`, 절대 경로 `artifact` 를 가진다.
- (g) 의 `text` 는 `--not-a-flag` 이다.
- 모든 기록에서 같은 객체 안에 중복 키가 없다(토큰 단위로 검사).

(h) 는 버전 한 줄을 stdout 에 쓰고 exit 0 이며 emacsclient 를 부르지 않는다.

- 검증: `go test ./cmd/imoogi-agent/... ./internal/agentipc/... -run 'TestAC11' -count=1`

### AC-AIPC-012 — 잘못된 호출은 전송 전에 exit 2 로 끝난다 (REQ-AIPC-011)

**Given** 호출되면 표시 파일을 남기는 가짜 emacsclient 가 지정되어 있고 `TMPDIR` 이 빈 임시 디렉터리일 때
**When** 다음을 실행하면: 인자 없음, `frobnicate`, `message`(TEXT 없음), `message ""`, `message a b`,
`goto f.go`, `goto f.go 0`, `goto f.go x`, `goto f.go 3 0`, `goto f.go 3 4 5`, `finish done`,
`message hi --bogus`, `message hi --project`, `message hi --timeout 0`, `IMOOGI_AGENT_TIMEOUT=abc message hi`,
1 MiB 를 넘는 TEXT
**Then** 모든 경우 exit 2, stderr 에 진단 정확히 한 줄(개행으로 끝나는 줄 1개), stdout 비어 있음, 가짜 emacsclient 표시 파일이 없고,
`TMPDIR` 에 남은 파일이 없다.

- 검증: `go test ./cmd/imoogi-agent/... ./internal/agentipc/... -run 'TestAC12' -count=1`

### AC-AIPC-013 — 이벤트 파일 전달 방식과 수명 (REQ-AIPC-012)

**Given** 받은 인자 목록, 이벤트 파일의 권한 비트와 내용을 기록하는 가짜 emacsclient 가 지정되어 있을 때
**When** 가짜가 (a) 수락, (b) 거부(`"error:bad-path"`), (c) `*ERROR*:` + exit 1, (d) 연결 실패, (e) 30초 대기 후 수락을
하도록 바꿔 가며 `imoogi-agent message hi --timeout 1` 을 실행하면
**Then** 모든 경우 가짜가 받은 인자 목록은 정확히 세 개 — `--eval`, 고정 평가식 하나, 절대 경로 하나 — 이고
(`--timeout` 을 포함한 다른 인자는 없음, `plan.md` D-9), 고정 평가식은 모든 호출에서 바이트 단위로 같으며
이벤트 내용이나 경로를 포함하지 않는다.
**And** 가짜가 관찰한 이벤트 파일 권한은 `0600` 이고, CLI 종료 후 그 경로에 파일이 없다((e) 포함).
**And** 고정 평가식은 `server-eval-args-left` 에서 경로를 꺼내는 일을 수신 함수 존재 확인보다 먼저 한다
(평가식 문자열에서 `pop` 이 `fboundp`·수신 함수 이름보다 앞에 나오는지 검사).

- 검증: `go test ./cmd/imoogi-agent/... ./internal/agentipc/... -run 'TestAC13' -count=1`

### AC-AIPC-014 — emacsclient 탐색 순서와 종료 코드 대응 (REQ-AIPC-013, REQ-AIPC-014)

**Given** 가짜 emacsclient 들과, `/Applications` 후보 목록을 임시 디렉터리로 바꿀 수 있는 테스트 훅이 있을 때
**When** 탐색 조건을 다음처럼 바꾸면
- (a) `EMACSCLIENT` 가 실행 가능 파일 → 그것을 쓴다
- (b) `EMACSCLIENT` 가 실행 불가 파일 → exit 1, PATH 후보가 있어도 쓰지 않는다
- (c) `EMACSCLIENT` 없음, PATH 와 앱 후보 둘 다 있음 → PATH 쪽
- (d) PATH 없음, `Emacs-31.1.app` 과 `Emacs.app` 후보 둘 다 있음 → `Emacs-31.1.app`
- (e) (d) 에서 `EMACS_VERSION=30.2` 이고 `Emacs-30.2.app` 후보 없음 → `Emacs.app`
- (f) `Emacs-29.4.app` 만 있음 → 그것(글롭)
- (g) 후보 없음 → exit 1

**Then** 괄호 안 결과와 같다.
**And** § 0.3 의 모양으로 가짜 출력을 바꾸면 종료 코드는 수락 0, 수신기 거부 3(stderr 에 `bad-path` 포함),
elisp 오류 3, 연결 실패 1, exit 0 + stdout `nil` 3, exit 0 + stdout 비어 있음 3 이다.
**And** 30초 대기하는 가짜로 `--timeout 1` 이면 exit 1 이고 CLI 가 2초 안에 끝나며,
`IMOOGI_AGENT_TIMEOUT=1` 에 `--timeout 3` 을 함께 주고 2초 대기하는 가짜면 exit 0 이다(옵션 우선),
아무 설정 없이 7초 대기하는 가짜면 exit 1 이고 6초 안에 끝난다(기본 5초).
**And** `TMPDIR` 을 쓰기 불가 디렉터리로 두면 exit 1 이고 emacsclient 를 부르지 않는다.

- 검증: `go test ./cmd/imoogi-agent/... ./internal/agentipc/... -run 'TestAC14' -count=1`

---

## 3. 빌드·부팅·설치와 실제 연동

### AC-AIPC-015 — 빌드·부팅·설치 연결과 실제 Emacs 연동 (REQ-AIPC-012, REQ-AIPC-014, REQ-AIPC-015)

**Given** 구현이 끝난 작업 트리에서
**When** 다음을 실행하면

```bash
make build-agent && test -x bin/imoogi-agent
make -n build-all | grep -c 'imoogi-agent'                  # 1 이상
grep -c 'build-agent' Makefile                              # .PHONY·build-all·타깃·help 포함 4 이상
ert_sel "^imoogi-module-reload-preserves-numbered-load-order"
bash tests/setup-toolchain-test.sh                          # 설치 링크 검증 포함
git diff --quiet HEAD -- go.mod go.sum go.work packages.el packages.lock vendor/ && echo unchanged
make ci-local
```

**Then** 모든 명령이 exit 0 이고, `setup-toolchain-test.sh` 는 `~/.local/bin/imoogi-agent` 가
`~/.config/imoogi-emacs/bin/imoogi-agent` 를 가리키는 심볼릭 링크임을 검사하며, 마지막 줄 전 명령이 `unchanged` 를 출력한다.

**And** 실제 Emacs 30.2 서버로 다음 스모크를 실행하면

```bash
EB=/Applications/Emacs.app/Contents/MacOS
"$EB/Emacs" -Q --daemon=aipc-smoke --eval '(defun imoogi-require (&rest _) nil)' \
  -l "$PWD/modules/development/30-agent.el"
"$EB/Emacs" -Q --daemon=aipc-bare
EMACSCLIENT="$EB/bin/emacsclient" EMACS_SOCKET_NAME=aipc-smoke bin/imoogi-agent message smoke;  echo "exit=$?"
EMACSCLIENT="$EB/bin/emacsclient" EMACS_SOCKET_NAME=aipc-smoke bin/imoogi-agent open-file /no/such; echo "exit=$?"
EMACSCLIENT="$EB/bin/emacsclient" EMACS_SOCKET_NAME=aipc-bare  bin/imoogi-agent message smoke;  echo "exit=$?"
EMACSCLIENT="$EB/bin/emacsclient" EMACS_SOCKET_NAME=aipc-none  bin/imoogi-agent message smoke;  echo "exit=$?"
"$EB/bin/emacsclient" -s aipc-smoke --eval '(kill-emacs)'; "$EB/bin/emacsclient" -s aipc-bare --eval '(kill-emacs)'
```

**Then** 차례로 `exit=0`, `exit=3`(stderr 에 `bad-path`), `exit=3`(stderr 에 `not-loaded`), `exit=1` 이 나오고,
네 호출 뒤 `TMPDIR` 에 `imoogi-agent` 이벤트 파일이 남아 있지 않다.

- 검증: 위 두 명령 묶음(스모크는 수동 실행, 결과를 run 단계 증거로 progress.md § E.2 에 남긴다).

---

## 4. Edge cases (기준에 포함된 경계 조건 모음)

| 경계 | 담당 AC |
|------|---------|
| `%` 가 들어간 알림 문자열 | AC-AIPC-001, AC-AIPC-011 |
| 알림·로그 값 안의 줄바꿈(LF·CRLF·CR) | AC-AIPC-001 |
| 선택 필드의 `[]`·`{}`·`false`·숫자 | AC-AIPC-007 |
| 정확히 1 MiB / 1 바이트 초과 | AC-AIPC-008 |
| 줄·열이 버퍼 범위를 넘음 | AC-AIPC-003 |
| 심볼릭 링크(파일/디렉터리/끊김), 장치 파일, TRAMP | AC-AIPC-009 |
| 파일 열기에서 생기는 간접 질문 | AC-AIPC-010 |
| `-` 로 시작하는 TEXT | AC-AIPC-011 |
| 수신 함수 미정의 Emacs | AC-AIPC-013, AC-AIPC-015 |
| emacsclient 가 멈춤 | AC-AIPC-013, AC-AIPC-014 |
| 이벤트 파일 생성 불가 | AC-AIPC-014 |

## 5. Quality Gate

- `make ci-local`(fmt-check → go vet → verify-vendor → test-elisp → test-go → test-shell) exit 0.
- `gofmt -l cmd/imoogi-agent internal/agentipc` 출력 없음.
- 새 Go 패키지 커버리지: `go test -cover ./cmd/imoogi-agent/... ./internal/agentipc/...` 각 85% 이상.
- 오프라인 부팅에서 `:error`/`:emergency` 경고 0건(`tests/run.sh` 의 boot-health 단계).
- 30-agent.el 은 `check-parens` 를 통과한다(`tests/run.sh` 1단계).

## 6. Definition of Done

- [ ] AC-AIPC-001 ~ AC-AIPC-015 전부 통과, 각 검증 명령과 출력이 progress.md § E.2 에 기록됨.
- [ ] § 5 품질 게이트 통과.
- [ ] spec.md § 6 의 [UNCHANGED] 파일에 변경 없음(`git diff --stat` 로 확인).
- [ ] 커밋되지 않은 기존 수정분(`26-project-notes.el`, `project-notes-test.el`, `CLAUDE.md`)이 이 작업의 커밋에 섞이지 않음(경로 지정 스테이징).
- [ ] sync 단계 문서 후속 작업 목록(plan.md § 7)이 progress.md 에 이월됨.
