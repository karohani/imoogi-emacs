# SPEC-AGENTIPC-001 — Implementation Plan

Tier M. 요구사항 15개(`spec.md` § 3), 인수 기준 15개(`acceptance.md`).
이 문서는 **어떻게**를 다룬다. 결정은 바뀔 가능성이 큰 것(데이터 계약, 사용자에게 보이는 흐름)부터 적고,
기계적인 연결 작업은 뒤에 둔다.

## 1. 맥락

- 기준 커밋: `main` @ `fd7f0b0`. 커밋되지 않은 수정분 `modules/project/26-project-notes.el`, `tests/project-notes-test.el`,
  `CLAUDE.md` 가 있다. 이 SPEC 은 그 파일들을 읽지도 고치지도 않는다.
- 근거 자료: `research.md`(4개 렌즈). plan 단계에서 아래를 직접 측정해 조사 내용을 정정·보강했다.

| 항목 | 측정 명령 요지 | 관찰 결과 |
|------|----------------|-----------|
| `server-eval-args-left` | Emacs 30.2 batch, `(require 'server)` 뒤 `boundp` | `t` |
| 문자열 결과 출력 | 임시 데몬에 `--eval '"ok"'` | stdout `"ok"\n`, exit 0 |
| 따옴표 이스케이프 | 경로 인자 `/tmp/a b"c.json` 을 문자열로 돌려받음 | `"error:not-loaded /tmp/a b\"c.json"` (Lisp `prin1` 형식) |
| elisp 오류 | `--eval '(error "boom")'` | stdout 없음, stderr `*ERROR*: boom`(개행 없음), exit 1 |
| 수신 함수 미정의 | `--eval '(nosuch (pop server-eval-args-left))'` | stderr `*ERROR*: Symbol’s function definition is void: nosuch`, exit 1 — 인자 평가 전에 실패하므로 `pop` 이 먼저 와야 함 |
| 연결 실패 | `-s /nonexistent/sock` | stderr `can't find socket…` 외 2줄, exit 1 |
| `--timeout` | `--timeout=1` + 4초 평가 | 4초 뒤 `"late"`, exit 0 — **느린 평가를 끊지 못함** |
| 사용자 입력 대기 | `--timeout=1` + `(read-string …)` | **무기한 멈춤**(프로세스를 강제 종료해야 했음) |
| `inhibit-interaction` | batch 30.2, `(let ((inhibit-interaction t)) (y-or-n-p …))` / `read-string` | 둘 다 `inhibited-interaction` 신호 |
| D-5 창 표시 action | batch 30.2(프레임 80x25, 창 1개)에서 D-5 action 으로 파일 표시 | 창 2개로 분할, 돌려받은 창 ≠ 선택된 창, `selected-window` 불변 |
| ERT 선택 실행 | `acceptance.md` § 0.1 형식, 선택자 `^imoogi-module-reload-preserves` | `Ran 3 tests, 3 results as expected, 0 unexpected`, exit 0 |

- `research.md` 모순 1 해결: `modules/org/28-clipboard.el:98` 은 `:null-object nil :false-object nil` 을 넘긴다
  (`13-system.el:35`, `23-org-preview.el:525` 도 `:null-object nil`). 새 모듈은 이 조합을 **따르지 않는다**(v0.1.1, plan-audit D3):
  그 조합에서는 `[]`·`{}`·`null`·`false` 가 모두 `nil` 이 되어 선택 필드의 형식 불일치를 찾을 수 없다. 대신 § 3.1 의 엄격한 옵션을 쓴다.
- `research.md` 모순 8(줄 번호): 아래 § 5 의 줄 번호는 plan 단계에서 다시 확인한 값이다.

## 2. 결정 사항 (바뀔 가능성이 큰 순서)

### D-1 수신기 반환 계약

수신기는 문자열 하나를 돌려준다: 정확히 `"ok"` 또는 `"error:<reason>"`(reason 목록 `spec.md` § 2.3)이며, 뒤에 아무것도 붙이지 않는다.
사람이 읽을 설명(예: `handler` 오류의 원래 메시지)이 필요하면 `*imoogi-agent*` 로그 줄에만 남긴다(줄바꿈은 `⏎` 로 바꿔 한 줄 유지).
진입점 전체를 `condition-case` 로 감싸 예상 못 한 오류는 `"error:handler"` 로 바꾼다.

### D-2 알 수 없는 필드는 무시

버전 1 안에서 필드를 추가해도 기존 Emacs 가 깨지지 않게 한다. 거부 대상은 버전·유형·필수 필드·키 중복뿐이다.

### D-3 경로 정책 위반은 이벤트 전체 거부

`task-finished.artifact` 가 나쁘면 알림까지 막는다. 부분 수락(알림은 띄우고 산출물만 무시)은 종료 코드 의미를 흐리므로 쓰지 않는다.
경로 검증은 처리 전에 검증 단계에서 끝낸다.

### D-4 로그 버퍼는 세션 한정, buffer-terminator 예외를 두지 않음

`*imoogi-agent*` 는 보이지 않은 채 30분 비활성이면 `buffer-terminator`(`13-system.el:125-132`)가 지울 수 있다.
이를 받아들인다. 로그는 진단용이고, 다음 이벤트 때 버퍼를 새로 만든다. 예외 처리(`buffer-terminator-rules-alist` 에
보존 규칙 추가 — `buffer-terminator-predicate` 는 1.1.0 부터 폐기됨)는 `13-system.el` 수정과 모듈 간 결합을 부르므로 하지 않는다.
영구 로그가 필요해지면 후속 SPEC 으로 다룬다.

### D-5 창 표시 방식

`display-buffer` 에 명시적 action 을 넘긴다(선택하지 않음, 버퍼 기록 안 함):

```elisp
'((display-buffer-reuse-window
   display-buffer-use-some-window
   display-buffer-pop-up-window)
  (inhibit-same-window . t)
  (reusable-frames . visible))
```

`inhibit-same-window` 로 선택된 창은 후보에서 빠지므로, 파일이 선택된 창에만 보이고 있으면 다른 창에 한 번 더 보인다
(사용자 point 보호가 우선). `goto-location` 은 돌려받은 창에 `set-window-point` 만 한다. `pop-to-buffer`,
`switch-to-buffer`, `select-window`, `find-file` 은 쓰지 않는다.

### D-6 줄·열은 1부터, 열은 선택

컴파일러·린터 출력이 1부터 세므로 그대로 받는다. Emacs 쪽에서 `(forward-line (1- line))`, `(move-to-column (1- column))`.
범위를 넘으면 `forward-line`/`move-to-column` 의 자연스러운 멈춤이 곧 `spec.md` REQ-AIPC-005.3 의 맞춤 규칙이다.

### D-7 종료 코드 1 은 "전달되지 않음" 전체

사용자 결정의 "연결 실패" 를 "Emacs 수신기에 닿지 못한 모든 경우" 로 읽는다. 이벤트 파일 생성·쓰기 실패도 1이다.
`internal/cli` 의 `ExitError = 1`(일반 오류) 관례와도 모순되지 않는다(`research.md` 모순 4 해결).

### D-8 CLI 는 파일 존재를 검사하지 않음

경로 정책의 단일 판정자는 Emacs 수신기다. CLI 는 상대 경로를 절대 경로로 바꾸기만 한다. 두 곳에서 판정하면 판정이 갈릴 때
어느 쪽이 맞는지 정할 수 없고(예: 심볼릭 링크, 권한), CLI 쪽 사전 검사는 원격·장치 파일 판정을 흉내 낼 수도 없다.

### D-9 시간 상한은 CLI 가 강제, emacsclient `--timeout` 은 넘기지 않음

측정 결과 `--timeout` 은 느린 평가를 끊지 못한다. CLI 는 `exec.CommandContext` + `context.WithTimeout` 으로 emacsclient 를 죽이고,
`cmd.WaitDelay`(예: 500ms)로 파이프 정리까지 상한 안에 끝낸다. emacsclient 인자는 `--eval EXPR PATH` 셋뿐이다.
기본 5초, `--timeout SECONDS` > `IMOOGI_AGENT_TIMEOUT` > 기본값 순. 값은 양의 정수 초.

### D-10 시간 초과 뒤의 이벤트 파일

CLI 가 emacsclient 를 죽인 뒤 파일을 지우므로, 서버가 요청을 뒤늦게 처리하면 파일이 없어 `unreadable` 거부로 로그에 남는다.
재시도하지 않는다. 드물게 서버가 이미 파일을 읽고 표시하던 중에 CLI 가 시간 초과로 끝나면 표시는 되었는데 종료 코드는 1 일 수 있다.
따라서 1 의 의미는 "전달 확인 안 됨" 이다(에이전트는 어차피 무시하는 최선 노력 신호).

### D-11 키 중복

Go `encoding/json` 은 나중 값을, Emacs alist 는 앞의 값을 쓴다(`research.md` § 7). CLI 는 구조체로 직렬화하므로 중복 키가 생기지 않고,
Emacs 는 봉투와 payload 두 단계에서 키 중복을 찾아 거부한다(`imoogi-agent-receive-json` 을 직접 부르는 다른 발신자 대비).
검사는 alist 키 목록의 길이와 중복 제거 후 길이를 비교하는 수준이면 충분하다.

### D-12 환경 변수 이름

`EMACSCLIENT`(탐색, `imoogi-editor` 와 공유), `EMACS_VERSION`(앱 번들 후보, 공유), `IMOOGI_AGENT_TIMEOUT`(새 이름).
`EMACS_SOCKET_NAME` 은 emacsclient 가 그대로 상속한다(테스트·스모크에 사용).

## 3. 기술 접근

### 3.1 Emacs 수신 모듈 `modules/development/30-agent.el`

- 머리: `(imoogi-require "30-agent")` — 내장 기능만 쓰므로 패키지 인자가 없다. 끝: `(provide 'imoogi-agent)`.
- 진입점 두 개 → 공통 `imoogi-agent--process (json-string)` → 검증 → `imoogi-agent-dispatch (event)`.
- 파일 진입점 순서: 절대 경로·`file-remote-p` 아님·`file-regular-p`·읽기 가능 확인 → `file-attribute-size` 로 1 MiB 확인
  → 그 뒤에만 `insert-file-contents`(UTF-8 디코딩)로 읽는다.
- 파싱: `json-parse-buffer :object-type 'alist :array-type 'array :null-object :null :false-object :false`
  (v0.1.1, plan-audit D3 로 `28-clipboard.el:98` 선례에서 벗어남). 이 옵션에서 값은 서로 구별된다(Emacs 30.2 측정, `spec.md` HISTORY):
  `null` → `:null`, `false` → `:false`, `true` → `t`, `[]`·배열 → 벡터, `{}` → `nil`, 객체 → alist.
  - 키의 유무는 `assq` 로 본다(값이 `nil` 인 키 = 빈 객체 `{}` 가 들어온 것).
  - 선택 필드: 키가 없거나 값이 `:null` 이면 없음. 그 밖에는 문자열 필드는 `stringp`, 정수 필드는 `natnump`/범위 검사를 통과해야 하며,
    아니면 `bad-field`(REQ-AIPC-002.4). 따라서 `[]`(벡터), `{}`(`nil`), `:false`, `t`, 숫자는 모두 `bad-field`.
  - 필수 필드: 값이 `:null` 이거나 형식이 다르면 `bad-field`. `payload` 는 alist(`listp`, `:null`·벡터 아님)여야 한다.
  - `:false`·`:null` 은 `nil` 이 아니지만 어떤 검사도 이 값을 참 값으로 쓰지 않는다. 필드 값은 항상 형식 술어(`stringp` 등)를 거친 뒤에만 쓴다
    (REQ-AIPC-002.1-2). `null`=없음·`false`=참 아님이라는 의미는 v0.1.0 과 같고, 바뀐 것은 선택 필드의 `false`·`[]`·`{}` 가
    이제 생략이 아니라 `bad-field` 라는 점뿐이다.
  `json-error` 로 모든 파싱 오류(깊이 초과 포함)를 잡는다.
  파싱 뒤 point 에서 공백을 건너뛰고 `eobp` 가 아니면 `trailing-content`.
- 키 중복 검사 → 버전 → 유형 → 유형별 필드 → payload 경로 순으로 검증(`spec.md` REQ-AIPC-003.5 순서).
- 경로 정책: `file-name-absolute-p`, `(file-remote-p path)`(연결을 열지 않음), `file-truename` 뒤 `file-regular-p`.
  통과하면 실제 경로를 처리 단계로 넘긴다.
- 처리 중 질문 차단(REQ-AIPC-008) — 두 겹:
  1. 원인 제거: `find-file-noselect` 에 NOWARN `t`(바뀐 파일 질문 생략), `large-file-warning-threshold` `nil`,
     `enable-local-variables` `:safe`, `enable-local-eval` `nil` 을 동적 바인딩.
  2. 안전망: 처리 전체를 `(let ((inhibit-interaction t)) …)` 로 감싼다. 남은 질문은 멈추는 대신 `inhibited-interaction`
     신호가 되어 `"error:handler"` 로 끝난다(측정으로 확인).
- 줄바꿈 정리: 알림·로그에 넣는 모든 이벤트 유래 문자열을 한 도우미 함수에 통과시켜 `\r\n`, `\n`, `\r` 를 각각 `⏎`(U+23CE) 하나로 바꾼다
  (정규식 `"\r\n\\|[\n\r]"` 로 한 번에 치환하면 CRLF 가 `⏎` 하나가 된다; `spec.md` REQ-AIPC-004.5).
- 알림: `(message "%s" text)` — 페이로드를 서식 문자열로 쓰지 않는다. `text` 는 위 정리를 거친 값이다.
- 처리 순서(v0.1.2, plan-audit 2회차 N1): 파일을 보여 주는 처리기(`artifact-created`, `artifact` 가 있는 `task-finished`)는
  REQ-AIPC-005 방식의 표시를 먼저 끝내고 알림 `message` 를 맨 마지막에 부른다. 그래서 파일을 열 때 모드·훅이 `*Messages*` 에
  무언가를 남겨도 알림이 수신 호출의 마지막 메시지가 된다. 로그 줄 추가는 `message` 를 쓰지 않으므로 순서에 영향이 없다.
  인수 기준은 이 순서에 기대지 않는다(`acceptance.md` § 0.1 은 "추가된 줄 가운데 정확히 하나"로 판정).
- 로그: `*imoogi-agent*` 를 `special-mode` 계열로 만들고 `inhibit-read-only` 로 끝에 한 줄 추가.
  형식 예: `[2026-09-27 10:47:00] message project=imoogi-emacs session=s-42 ok 빌드 완료`.

### 3.2 Go CLI `cmd/imoogi-agent` + `internal/agentipc`

- `cmd/imoogi-agent/main.go`: `var version = "dev"`, `main()` 은 `os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))`.
  `run` 시그니처는 `cmd/imoogi-anki/main.go:52` 선례와 같다.
- `internal/agentipc`(표준 라이브러리만):
  - 인자 해석: `flag` 패키지 없이 손으로(저장소 관례). 옵션 위치 자유, `--` 종료자, 잘못된 입력은 사용 오류 타입으로 반환 → exit 2.
  - 이벤트 조립: 유형별 구조체, 필드 순서 고정, `json.Encoder` + `SetEscapeHTML(false)`, 선택 필드는 `omitempty`.
    직렬화 결과가 1 MiB 를 넘으면 사용 오류.
  - 탐색: `spec.md` REQ-AIPC-013 순서. 앱 번들 후보 목록은 패키지 변수로 두어 테스트가 임시 디렉터리로 바꿀 수 있게 한다.
    `exec.LookPath` 는 현재 디렉터리 PATH 항목을 거부한다(Go 1.19+).
  - 전송: `os.CreateTemp("", "imoogi-agent-*.json")` → `Chmod(0o600)`(umask 와 무관하게 명시) → 쓰기 → 닫기 → `defer os.Remove`.
    `exec.CommandContext(ctx, client, "--eval", evalExpr, path)`, 셸 없음. stdout·stderr 는 크기 제한 버퍼(예: 64 KiB)로 받는다.
  - 고정 평가식:
    ```elisp
    (let ((f (pop server-eval-args-left)))
      (if (fboundp 'imoogi-agent-receive-file)
          (imoogi-agent-receive-file f)
        "error:not-loaded"))
    ```
    `pop` 이 가장 먼저 실행되므로 수신 함수가 없어도 경로가 Lisp 로 평가되지 않는다(비슷한 식을 30.2 에서 측정).
  - 결과 분류: 시간 초과 → 1. 실행 실패(찾기·권한) → 1. stdout 을 Lisp 문자열로 읽기(`"` 로 감싸짐, `\"`·`\\` 만 해제) →
    정확히 `ok` 면 0, `error:` 로 시작하면 3(reason 은 `error:` 뒤 전체). stderr 에 `*ERROR*:` 가 있으면 3. 그 밖의 비정상 종료 → 1. exit 0 인데 해석 불가 → 3.
- 테스트 가짜 emacsclient: 헬퍼 프로세스 방식(테스트 바이너리를 `EMACSCLIENT` 로 지정하고 `TestMain` 이 환경 변수를 보고
  가짜로 동작, `m.Run` 전에 처리). 셸 의존 없이 인자·권한 비트·내용을 기록하고, § 1 측정값과 같은 바이트를 출력한다.
  셸 스크립트 가짜(`tests/setup-toolchain-test.sh:118-130` 방식)도 허용하나, 권한 비트 확인이 `stat` 방언에 묶인다.

### 3.3 빌드·부팅·설치

- `Makefile`: `build-notes`(`Makefile:132-134`) 형식을 복사한 `build-agent`, `build-all`(`:109`)·`.PHONY`(`:35`)·`help`(`:56-66` 근처) 추가.
- `boot.el`: 모듈 목록 끝(`"org/29-org-roam"` 다음)에 `"development/30-agent"`.
- `tests/module-layout-test.el`: 기대 목록(`:47-54`) 끝에 `"30-agent"`.
- `scripts/install.sh`: `EDITOR_LINK`(`:19`) 옆에 `AGENT_LINK="${EDITOR_BIN_DIR}/imoogi-agent"`, `imoogi-editor` 링크(`:88`)
  다음에 `ln -sfn "${CONFIG_LINK}/bin/imoogi-agent" "${AGENT_LINK}"`. 대상이 아직 없으면(빌드 전) 링크는 만들되
  `make build-agent` 안내 한 줄을 출력한다. 빌드는 하지 않는다.
- `tests/setup-toolchain-test.sh`: 기존 링크 검사(`:101-102`) 옆에 `imoogi-agent` 링크 존재와 대상 문자열 검사.

## 4. 마일스톤

바뀔 가능성이 큰 결정(프로토콜 계약, 화면 동작)을 먼저 확정하고 기계적인 연결을 뒤에 둔다. 각 마일스톤은 앞 마일스톤이 끝난 뒤 시작한다.

### M1 — Emacs 프로토콜·검증·알림·로그 (Priority High)

- 범위: 진입점 두 개, 크기·파일 조건, 파싱·후행 내용·키 중복, 스키마, `message`/`task-finished`(산출물 없는 경우) 알림, 로그 버퍼.
- 파일: `modules/development/30-agent.el` [NEW], `tests/agent-test.el` [NEW].
- REQ: 001, 002, 003, 004(1-3), 007. AC: 001, 005(a)(b), 006, 007, 008.
- 검증 중 필요한 임시 로드: 이 단계에서는 `boot.el` 에 아직 넣지 않으므로 테스트 파일이 모듈을 직접 `load` 한다
  (M5 에서 부팅 목록에 넣은 뒤에도 중복 로드가 무해하도록 `defvar`/`defun` 만 둔다).
- 완료: `ert_sel "^imoogi-agent-ac0[15678]-"` 통과.

### M2 — 경로 안전·창 표시·질문 차단 (Priority High)

- 범위: 경로 정책, `open-file`/`goto-location`/`artifact-created`, `task-finished` 산출물 표시, D-5 창 동작, § 3.1 질문 차단 두 겹.
- 파일: M1 과 같음.
- REQ: 004(4), 005, 006, 008, 009. AC: 002, 003, 004, 005(c), 009, 010.
- 완료: `ert_sel "^imoogi-agent-ac"` 전체 통과(001-010).

### M3 — Go CLI 문법·이벤트 조립·exit 2 (Priority High)

- 범위: 인자 해석, 이벤트 구조체, 상대 경로 절대화, 크기 검사, `--version`. 전송은 인터페이스 뒤에 두고 가짜로 대체.
- 파일: `cmd/imoogi-agent/main.go` [NEW], `cmd/imoogi-agent/main_test.go` [NEW], `internal/agentipc/*` [NEW].
- REQ: 010, 011. AC: 011, 012.
- 완료: `go test ./cmd/imoogi-agent/... ./internal/agentipc/... -run 'TestAC1[12]'` 통과, `go vet` 깨끗.

### M4 — 탐색·전송·종료 코드·시간 상한 (Priority High)

- 범위: 탐색 순서, 이벤트 파일 수명, 고정 평가식, 출력 분류, 시간 상한과 강제 종료, 헬퍼 프로세스 가짜.
- REQ: 012, 013, 014. AC: 013, 014.
- 권장 추가 테스트(AC 아님): `scripts/imoogi-editor` 를 읽어 Go 후보 목록과 같은 순서의 경로 문자열이 들어 있는지 확인하는
  동기화 감시 테스트(§ 6 R-2).
- 완료: `go test ./cmd/imoogi-agent/... ./internal/agentipc/... -run 'TestAC1[34]'` 통과, 커버리지 85% 이상.

### M5 — 부팅·빌드·설치 연결과 실제 연동 스모크 (Priority Medium)

- 범위: `boot.el`, `tests/module-layout-test.el`, `Makefile`, `scripts/install.sh`, `tests/setup-toolchain-test.sh`, 실제 데몬 스모크.
- REQ: 015 (+ 012·014 실제 연동). AC: 015.
- 완료: `acceptance.md` AC-AIPC-015 의 두 명령 묶음 결과를 progress.md § E.2 에 기록, `make ci-local` exit 0.

## 5. 참조 구현 (plan 단계에서 줄 번호 재확인)

| 대상 | 위치 | 가져올 것 |
|------|------|-----------|
| emacsclient 탐색 순서 | `scripts/imoogi-editor` `find_emacsclient`(5-37행) | 후보 순서, `EMACSCLIENT` 실행 불가 시 실패 |
| 가짜 emacsclient 테스트 | `tests/setup-toolchain-test.sh:118-130` | 인자 기록 방식 |
| 설치 링크 | `scripts/install.sh:19`, `:88-89` | `ln -sfn` 과 안내 출력 |
| 설치 링크 검사 | `tests/setup-toolchain-test.sh:101-102` | `-L`·`readlink` 검사 |
| Emacs 서버 기동 | `modules/general/13-system.el:113-122` | 기본 소켓, TCP 없음 |
| TMPDIR 동기화 | `modules/general/13-system.el:99-110` | GUI Emacs 로 `TMPDIR` 복사(R-7) |
| buffer-terminator | `modules/general/13-system.el:125-132` | 30분 비활성 정리(D-4) |
| 모듈 사전조건 | `boot.el:44-51` `imoogi-require` | 패키지 없는 호출도 동작 |
| 모듈 목록 | `boot.el:65-101` `dolist` | 끝에 추가 |
| 로드 순서 테스트 | `tests/module-layout-test.el:45-67` | 기대 목록 |
| 내장 JSON 파서 호출 형태 | `modules/org/28-clipboard.el:98` | `json-parse-buffer` 호출과 `json-error` 처리만. 옵션은 따르지 않고 § 3.1 의 엄격한 옵션(`:array-type 'array :null-object :null :false-object :false`)을 쓴다 |
| 테스트 가능한 `run` | `cmd/imoogi-anki/main.go:52` | `run(args, stdin, stdout, stderr) int` |
| 사용 오류 exit 2 | `cmd/imoogi-notes/main.go:21` | 사용 오류 관례 |
| 입력 크기 제한 선례 | `cmd/imoogi-notes/main.go:23` | 1 MiB |
| 빌드 타깃 형식 | `Makefile:132-134`, `:109`, `:35` | `build-notes` 복사 |
| ERT 러너 | `tests/run.el:38-43` | 선택자 advice 형식(`acceptance.md` § 0.1) |

## 6. 위험과 완화

| # | 위험 | 완화 |
|---|------|------|
| R-1 | emacsclient 는 연결 실패와 elisp 오류 모두 exit 1 → 1/3 구분 불가 | 수신기 상태 문자열 + stderr `*ERROR*:` 분류(D-1, § 3.2). 바이트 모양은 30.2 에서만 측정됨 → 31.1 서버로도 M5 스모크 반복 권장 |
| R-2 | 탐색 로직이 bash(`imoogi-editor`)와 Go 두 벌 → 어긋남 | 순서를 SPEC 에 고정(REQ-AIPC-013), M4 동기화 감시 테스트, Go 쪽 주석에 `scripts/imoogi-editor` 위치 명시(`imoogi-editor` 는 이 SPEC 에서 수정하지 않음 — § 9) |
| R-3 | Emacs 30.2(`Emacs.app`)와 31.1(`Emacs-31.1.app`) 동시 설치, 탐색은 31.1 을 먼저 고름 | emacsclient 는 서버 버전과 무관하게 같은 프로토콜을 쓴다. `server-eval-args-left` 는 서버 쪽 기능(30.1+)이라 두 버전 모두 해당. 31.1 서버 동작은 미측정 → 스모크에서 `EMACSCLIENT` 로 두 조합 확인 |
| R-4 | emacsclient `--timeout` 무력(측정), 질문이 뜨면 무기한 멈춤 | CLI 강제 종료(D-9), 수신기 질문 차단 두 겹(§ 3.1) |
| R-5 | 시간 초과 뒤 서버가 늦게 처리 | D-10: 파일이 없어 `unreadable` 로그, 재시도 없음, exit 1 = 전달 확인 안 됨 |
| R-6 | `*imoogi-agent*` 가 30분 뒤 정리됨 | D-4 로 수용. 로그는 세션 한정 진단용 |
| R-7 | 에이전트 셸과 GUI Emacs 의 `TMPDIR` 이 다르면 소켓을 못 찾음 | exit 1 로 보고. `exec-path-from-shell` 이 `TMPDIR` 을 맞춤(`13-system.el:99-110`). `EMACS_SOCKET_NAME` 으로 우회 가능. 이벤트 파일은 절대 경로라 영향 없음 |
| R-8 | 키 중복 해석 차이(Go 나중 값, Emacs 앞 값) | D-11: CLI 는 중복을 만들지 않고 Emacs 는 거부 |
| R-9 | 열린 파일의 모드 훅(eglot 등)이 느리거나 질문을 띄움 | `inhibit-interaction` 이 질문을 오류로 바꾼다. 느린 훅은 시간 상한 안에서 exit 1 이 될 수 있음(수용) |
| R-10 | 사용자 `display-buffer-alist` 가 action 을 덮어써 새 프레임 등 생성 | `display-buffer` 는 어떤 경우에도 창을 선택하지 않는다. AC-AIPC-002 가 선택 창·프레임 불변을 확인 |
| R-11 | 에이전트 셸 PATH 에 emacsclient 가 없음(측정: `which emacsclient` 없음) | 앱 번들 후보로 찾음(`Emacs.app` 번들에 존재 확인) |
| R-12 | 30-agent 모듈이 로드되지 않은 Emacs(부팅 실패 등) | 고정 평가식이 `error:not-loaded` 반환 → exit 3, 경로는 평가되지 않음 |
| R-13 | emacsclient 출력이 매우 큼 | 크기 제한 버퍼로 받음 |
| R-14 | 여러 에이전트가 동시에 호출 | 이벤트 파일 이름이 서로 다르고 서버가 요청을 순서대로 처리. 공유 상태 없음 |

## 7. sync 단계 후속 작업 (run 범위 아님)

"여섯 개의 CLI" 서술과 "모든 Go CLI 는 Emacs 가 하위 프로세스로 실행한다" 서술이 사실이 아니게 된다.
sync 단계에서 manager-docs 가 함께 고친다.

- `.moai/project/product.md:9`
- `.moai/project/structure.md:5`, `:86`, `:170`(디렉터리 트리에 `cmd/imoogi-agent`, `modules/development/30-agent.el` 추가 포함)
- `.moai/project/tech.md:8`, `:55`, `:66`(패키지 수), `:112`, `:155`
- `.moai/project/codemaps/overview.md:8`, `:14`(Emacs 안으로 호출하는 첫 CLI 예외 기술)
- `.moai/project/codemaps/entry-points.md:43`, `.moai/project/codemaps/data-flow.md:3`(흐름 추가)
- `README.md` 외부 편집기 절 근처에 `imoogi-agent` 사용법과 산출물 파일 관례(`spec.md` § 2.6) 기술

## 8. 미결 사항

없음. 사용자 결정 9개와 위 D-1 ~ D-12 로 run 단계 진입에 필요한 결정이 모두 정해졌다.

## 9. run 단계 제약 (PRESERVE)

- 수정 금지: `modules/project/26-project-notes.el`, `tests/project-notes-test.el`, `CLAUDE.md`, `modules/general/13-system.el`,
  `scripts/imoogi-editor`, `go.mod`, `go.sum`, `go.work`, `packages.el`, `packages.lock`, `vendor/`, 다른 SPEC 디렉터리.
- 스테이징은 경로를 지정해서만 한다(`git add -A` 금지).
- 권장 오케스트레이션 모드: `serial`(코딩 위주, 두 언어, 단일 manager-develop 이 M1→M5 순서로 진행).
