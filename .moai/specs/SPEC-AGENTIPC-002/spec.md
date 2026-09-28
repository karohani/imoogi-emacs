---
id: SPEC-AGENTIPC-002
title: "imoogi-agent hardening: event-file trust, payload size cap, bare EMACSCLIENT lookup"
version: "0.1.1"
status: completed
created: 2026-09-27
updated: 2026-09-28
author: jay
priority: P2
phase: "v0.x agent-ipc hardening"
module: "internal/agentipc, modules/development/30-agent.el"
lifecycle: spec-anchored
tags: "agent, ipc, emacsclient, security, hardening"
tier: S
depends_on: [SPEC-AGENTIPC-001]
---

## HISTORY

### v0.1.1 (2026-09-27)

plan-audit 1회차(`.moai/reports/plan-audit/SPEC-AGENTIPC-002-review-1.md`, FAIL 0.86) 지적 반영. 요구사항·인수 기준 개수는 8/8 그대로다.

- D1: REQ-AIPH-007 정규 문장과 007.3 을 `exec.LookPath` 의 "처음 찾아진 항목이 결정" 규칙에 맞췄다. 처음 찾아진 항목이 빈 항목이나
  상대 항목이면 해석 실패이며 뒤의 절대 경로 항목으로 넘어가지 않는다. AC-AIPH-008 에 (c)(`.` 다음 `ec` 가 있는 절대 경로 디렉터리)를 추가했다.
- D2: `plan.md` D-1 의 근거를 고쳤다(사용자 소유·다른 사람 쓰기 불가 파일도 내용은 공격자가 고른 것일 수 있음). 잔여 위험 R-7 을 추가했다.
- D3: REQ-AIPH-007.4 와 `plan.md` D-4 에 "결과는 절대 경로여야 함"을 추가했다(`GODEBUG=execerrdot=0` 대비). AC-AIPH-008 (d) 로 확인한다.
  `os.Setenv("GODEBUG", …)` 이 실행 중에 반영된다는 근거는 go1.26.4 `runtime/runtime.go` 의 `syscall_runtimeSetenv`(202-209행)를 읽은 것이며 실행하지는 않았다.
- D4: AC-AIPH-001 이 거부 로그 줄의 `type`·`project`·`session` 이 `-` 임을 확인한다.
- D5: AC-AIPH-006 이 연달아 실행한 두 호출의 디렉터리 이름이 다름을 확인한다(REQ-AIPH-006.1).
- D6: `findEmacsclient` 인용 범위를 120-149행으로 바로잡았다(`$EMACSCLIENT` 분기는 120-127행).
- D7: plan-audit 가 희소 파일 생성을 측정했다(Emacs 30.2 batch, `dd … seek=104857601` exit 0, 크기 104857601). `plan.md` R-3·`progress.md` 갱신.
- D8: 이 SPEC 에 쓰이지 않는 `[Ubiquitous — negated]` 설명을 지웠다.
- D9: § 2.4 표에 REQ-AIPC-009.4 행을 추가했다.

### v0.1.0 (2026-09-27)

최초 작성. 완료된 SPEC-AGENTIPC-001 의 sync 단계 4차원 리뷰(`.moai/reports/sync-audit/SPEC-AGENTIPC-001-4dim.json`,
PASS 0.873)가 Security 차원에서 남긴 minor 지적 세 건과 백로그 카드 t18·t19·t20 을 한 SPEC 으로 묶었다.
결정은 모두 사용자가 내렸으며(§ 1.3), 이 문서는 그 결정을 요구사항과 인수 기준으로 옮긴다.

plan 단계에서 다음을 직접 측정했다.

- Emacs 30.2 `--batch -Q`: `make-temp-file` 로 만든 파일은 모드 `600`, `(file-attribute-user-id (file-attributes f 'integer))`
  → `501`, `(user-uid)` → `501`. `set-file-modes` 로 `#o620`·`#o602` 를 준 뒤 `(logand (file-modes f) #o022)` 는 둘 다 0 이 아니었다.
- Emacs 30.2 `--batch -Q`: 바이트 컴파일된 함수 안의 `(user-uid)` 호출도 `cl-letf` 로 `symbol-function` 을 바꾸면 바뀐 값(`4242`)을 돌려받았다.
  따라서 "다른 사용자 소유" 파일은 root 없이 시험할 수 있다.
- Go: `go version` → `go1.26.4 darwin/arm64`, `go.mod` 의 `go 1.26`. `go doc os/exec.ErrDot` 는 "ErrDot indicates that a path lookup
  resolved to an executable in the current directory due to ‘.’ being in the path, either implicitly or explicitly." 라고 적는다.
  `exec.LookPath` 가 이런 결과를 오류로 돌려준다는 것은 문서에 따른 것이며 plan 단계에서 직접 실행해 보지는 않았다(AC-AIPH-008 (b) 가 확인한다).
- 측정하지 못한 것: 100 MiB 를 넘는 희소 파일 생성(`dd … seek=`)은 plan 단계 셸에서 허용되지 않아 실행하지 못했다. `plan.md` § 6 R-3 참조.
  (v0.1.1 에서 plan-audit 측정으로 해소됨.)

## 1. 개요

### 1.1 목적

SPEC-AGENTIPC-001 이 만든 `imoogi-agent` 브리지의 세 가지 틈을 막는다.

- **t18 이벤트 파일 신뢰**: 수신기가 이벤트 파일의 소유자와 권한을 보지 않는다. SPEC-AGENTIPC-001 `plan.md` D-10 은 CLI 가 시간 초과로
  이벤트 파일을 지운 뒤 서버가 늦게 읽을 수 있음을 받아들였다. 모두가 쓸 수 있는 공유 TMPDIR(리눅스에서 TMPDIR 이 없을 때의 `/tmp`)에서는
  그 사이 다른 로컬 사용자가 같은 이름의 파일을 만들어 넣을 수 있고, 수신기는 그 내용을 진짜 이벤트로 처리한다.
  수신기(소유자·권한 검사)와 CLI(사용자 전용 디렉터리) 두 겹으로 막는다.
- **t19 맨 이름 `EMACSCLIENT`**: `EMACSCLIENT` 에 경로 구분자가 없는 이름(예: `ec`)이 들어오면 CLI 가 현재 작업 디렉터리 기준으로
  절대 경로를 만들어 `./ec` 를 실행한다. 에이전트의 작업 디렉터리가 신뢰할 수 없는 저장소라면 그 안에 심어 둔 실행 파일이 돈다.
  맨 이름은 PATH 에서만 찾는다.
- **t20 payload 파일 크기**: 1 MiB 상한은 이벤트 자체에만 걸린다. 수신기는 payload 경로의 파일을 크기와 관계없이 열며,
  열 때 큰 파일 경고(`large-file-warning-threshold`)를 꺼 두었다(SPEC-AGENTIPC-001 REQ-AIPC-008.1). 따라서 거대한 파일 하나로
  Emacs 서버를 오래 붙잡을 수 있다. 경고 대신 고정 상한을 둔다.

### 1.2 범위

- Emacs 수신 모듈 `modules/development/30-agent.el`: 이벤트 파일 신뢰 검사(파일 진입점만), payload 파일 크기 상한.
- Go 패키지 `internal/agentipc`: 이벤트 파일을 사용자 전용 임시 디렉터리 안에 쓰고 지우기, 맨 이름 `EMACSCLIENT` 탐색.
- 두 쪽의 테스트: `tests/agent-test.el`, `internal/agentipc/*_test.go`.

### 1.3 근거와 사용자 결정

| 카드 | 리뷰 지적(`SPEC-AGENTIPC-001-4dim.json` findings) | 사용자 결정 |
|------|---------------------------------------------------|-------------|
| t18 | Security #2: 수신기가 이벤트 파일 소유자·모드를 검사하지 않음 | 수신기는 소유자 uid ≠ `(user-uid)` 또는 그룹·기타 쓰기 가능이면 `untrusted` 로 거부. CLI 는 `os.MkdirTemp` 로 0700 디렉터리를 만들고 그 안에 0600 파일을 쓴 뒤 둘 다 지운다 |
| t19 | Security #3: 맨 이름 `EMACSCLIENT` 을 현재 디렉터리 기준으로 해석 | 경로 구분자가 없으면 `exec.LookPath` 로만 찾는다(현재 디렉터리 PATH 항목 거부). 못 찾으면 전달 실패(종료 코드 1) |
| t20 | Security #4: payload 파일 크기 상한 없음 | 100 MiB(104,857,600 바이트, `modules/general/00-defaults.el:40` 의 `large-file-warning-threshold` 와 같은 값) 초과면 이벤트 전체를 `payload-too-large` 로 거부 |

카드 t20 의 문구는 "`error:too-large` 추가"지만, 사용자는 이벤트 크기용 기존 reason `too-large` 와 구별되는 새 reason
`payload-too-large` 를 정했다. `too-large` 의 뜻(이벤트 1 MiB 초과)은 바뀌지 않는다.

### 1.4 용어

SPEC-AGENTIPC-001 § 1.3 의 용어를 그대로 쓴다. 추가 용어는 다음과 같다.

| 용어 | 뜻 |
|------|----|
| 신뢰 검사 | 이벤트 파일의 소유자와 권한 비트를 보는 검사. 결과가 거짓이면 `untrusted` |
| 그룹·기타 쓰기 가능 | 파일 모드와 `#o022` 의 비트 논리곱이 0 이 아님 |
| payload 크기 상한 | 104,857,600 바이트. payload 파일 크기가 이 값을 **넘으면** 거부, 같으면 수락 |
| 맨 이름 | 경로 구분자 `/` 가 한 번도 들어 있지 않은 `EMACSCLIENT` 값 |
| 사용자 전용 디렉터리 | CLI 가 호출마다 새로 만드는 모드 0700 임시 디렉터리. 이벤트 파일은 이 안에 만든다 |

## 2. 프로토콜 변경 (SPEC-AGENTIPC-001 대비)

### 2.1 수신기 상태 문자열 추가

SPEC-AGENTIPC-001 § 2.3 표에 두 줄을 더한다. 기존 reason 의 뜻은 바뀌지 않는다.

| reason | 조건 |
|--------|------|
| `untrusted` | 이벤트 파일(심볼릭 링크를 해석한 실제 파일)의 소유자 uid 가 `(user-uid)` 와 다르거나, 그룹 또는 기타 사용자가 쓸 수 있음. `imoogi-agent-receive-file` 에만 해당 |
| `payload-too-large` | payload 경로가 경로 정책(REQ-AIPC-009)은 통과했으나 그 파일 크기가 104,857,600 바이트를 넘음 |

### 2.2 CLI 종료 코드

새 표는 없다. 두 reason 모두 수신기가 돌려주는 `"error:<reason>"` 이므로 SPEC-AGENTIPC-001 § 2.4 에 따라 종료 코드 **3** 이다.
맨 이름 `EMACSCLIENT` 을 찾지 못한 경우는 "emacsclient 없음/실행 불가" 에 속하므로 종료 코드 **1** 이다.

### 2.3 첫 위반 보고 순서

SPEC-AGENTIPC-001 REQ-AIPC-003.5 의 순서를 다음으로 넓힌다. 여러 위반이 겹치면 이 순서의 첫 위반을 보고한다.

```text
파일 조건(unreadable) → 신뢰(untrusted) → 크기(too-large) → 문법(parse) → 후행 내용(trailing-content)
→ 키 중복(duplicate-key) → 버전(bad-version) → 유형(unknown-type) → 필드(bad-field)
→ 경로(bad-path) → payload 크기(payload-too-large)
```

- 신뢰 검사를 크기 검사보다 앞에 두는 이유: 신뢰할 수 없는 파일에 대해서는 크기를 포함한 어떤 판정도 결과로 내보내지 않는다.
- `imoogi-agent-receive-json` 에는 파일이 없으므로 `unreadable`·`untrusted` 단계가 없다.
- 한 이벤트의 payload 경로는 많아야 하나다(`path` 또는 `artifact`). 그 경로에 대해 `bad-path` 를 먼저 보고 `payload-too-large` 를 본다.

### 2.4 SPEC-AGENTIPC-001 조항과의 관계

SPEC-AGENTIPC-001 은 `completed` 상태이며 수정하지 않는다. 이 SPEC 이 구현되면 아래 조항은 이 SPEC 의 문장으로 좁혀 읽는다.

| SPEC-AGENTIPC-001 조항 | 이 SPEC 에서의 변경 |
|------------------------|---------------------|
| § 2.3 reason 표 | `untrusted`, `payload-too-large` 추가(§ 2.1) |
| § 2.4 종료 코드 | 변경 없음. 두 reason 은 기존 분류로 3(§ 2.2, REQ-AIPH-005) |
| REQ-AIPC-003.5 검사 순서 | § 2.3 의 순서로 확장(REQ-AIPH-002, REQ-AIPH-003) |
| REQ-AIPC-008.1 큰 파일 경고 금지 | 그대로 유지. 경고 대신 payload 크기 상한이 한계가 된다(REQ-AIPH-004) |
| REQ-AIPC-009 · 009.4 통과한 경로는 열어서 보여 주기만 함 | 경로 정책을 통과한 경로라도 파일이 상한을 넘으면 `payload-too-large` 로 거부한다(REQ-AIPH-003) |
| REQ-AIPC-012 · 012.1 · 012.4 이벤트 파일 | 0600 파일을 호출마다 새로 만드는 0700 디렉터리 안에 쓰고, 모든 경로에서 파일과 디렉터리를 함께 지운다(REQ-AIPH-006) |
| REQ-AIPC-013 탐색 순서 | 순서는 그대로. `$EMACSCLIENT` 가 맨 이름이면 PATH 에서만 찾는다(REQ-AIPH-007, 008). 이 경우는 `scripts/imoogi-editor` 와 의도적으로 달라진다 |

## 3. Requirements (GEARS)

요구사항 8개, 두 묶음. 정규 문장은 GEARS 영어 형식이고, 이어지는 한국어 항목이 세부 조건을 정한다.

### 3.1 Emacs 수신기 (t18a, t20)

#### REQ-AIPH-001 [When — event-detected]

**When the event file passed to `imoogi-agent-receive-file` is detected to be owned by a user ID other than `(user-uid)` or to be writable by its group or by others, the receiver shall reject the event with `error:untrusted`, read none of its content, perform no dispatch and no display, and append one rejection line to the log.**

1. "그룹 또는 기타 쓰기 가능"은 파일 모드와 `#o022` 의 비트 논리곱이 0 이 아님을 뜻한다. 읽기·실행 비트는 판정에 쓰지 않는다.
2. 거부 줄은 SPEC-AGENTIPC-001 REQ-AIPC-007 형식을 따르며 결과 자리에 `untrusted` 가 들어간다. 내용을 읽지 않았으므로 `type`·`project`·`session` 은 `-` 다.
3. 에코 영역 알림은 내지 않는다.

#### REQ-AIPH-002 [Ubiquitous]

**The receiver shall apply the trust check only in `imoogi-agent-receive-file`, to the file whose content would be read after resolving symbolic links, after the existing event-file preconditions and before the size check and any read of the content.**

1. 검사 대상은 이벤트 파일 경로의 심볼릭 링크를 해석한 실제 파일이다. 링크 자체의 소유자·모드는 판정에 쓰지 않는다.
2. 순서는 § 2.3 이다: 기존 파일 조건(`unreadable`)이 먼저이고, 그다음 `untrusted`, 그다음 `too-large`.
3. `imoogi-agent-receive-json` 의 동작은 바뀌지 않는다. 이 진입점에는 파일이 없으므로 어떤 파일 속성도 결과에 영향을 주지 않는다.
4. 두 진입점이 같은 입력에 같은 결과를 낸다는 SPEC-AGENTIPC-001 REQ-AIPC-001.1 은 신뢰 검사를 통과한 이벤트 파일에 대해 그대로 성립한다.

#### REQ-AIPH-003 [When — event-detected]

**When a payload path that passed the path policy of REQ-AIPC-009 is detected to name a file larger than 104,857,600 bytes, the receiver shall reject the whole event with `error:payload-too-large`, visit no file, display nothing, show no notification, and append one rejection line to the log.**

1. 크기는 심볼릭 링크를 해석한 실제 파일의 크기다. 정확히 104,857,600 바이트는 상한을 넘지 않으므로 수락한다.
2. 검사는 검증 단계에서 끝난다. 처리(dispatch) 단계로 넘어가기 전에 거부하므로 `task-finished` 의 `artifact` 가 상한을 넘으면
   완료/실패 알림도 띄우지 않는다(SPEC-AGENTIPC-001 REQ-AIPC-009.3 과 같은 이벤트 전체 거부).
3. 대상 이벤트: `open-file.path`, `goto-location.path`, `artifact-created.path`, `task-finished.artifact`.
4. 이벤트 크기 상한 `too-large`(1 MiB)와는 별개의 reason 이다.

#### REQ-AIPH-004 [Ubiquitous]

**The receiver shall keep opening accepted payload files without any large-file prompt, so that the payload size cap, not the interactive warning, bounds the files it opens.**

1. 상한 이하의 payload 파일은 SPEC-AGENTIPC-001 REQ-AIPC-005 대로 표시한다. 사용자의 전역 `large-file-warning-threshold` 값이
   파일 크기보다 작아도 질문하지 않는다(REQ-AIPC-008.1 유지).
2. 상한 값 104,857,600 바이트는 수신 모듈 안의 한 곳에서 정해지며, 사용자의 `large-file-warning-threshold` 설정을 읽어 쓰지 않는다.

### 3.2 Go CLI (t18b, t19, 종료 코드)

#### REQ-AIPH-005 [Ubiquitous]

**The CLI shall classify the receiver statuses `"error:untrusted"` and `"error:payload-too-large"` as protocol rejections with exit code 3 and report the reason in one stderr line.**

1. 이는 SPEC-AGENTIPC-001 REQ-AIPC-014.2 의 기존 분류로 이미 성립하며, 이 SPEC 은 두 reason 에 대해 그것을 계약으로 고정한다.

#### REQ-AIPH-006 [Ubiquitous]

**The CLI shall write each event file with mode 0600 inside a new directory created for that invocation with mode 0700 under the system temporary directory, and shall remove both the file and the directory on every path, including acceptance, rejection, delivery failure, timeout, and event-file write failure.**

1. 디렉터리는 호출마다 새로 만들며 이름은 CLI 가 정한다(예측 가능한 고정 이름을 쓰지 않는다). 이름은 `imoogi-agent-` 로 시작한다.
2. emacsclient 에 넘기는 경로는 여전히 절대 경로 하나다(SPEC-AGENTIPC-001 REQ-AIPC-012.1).
3. 디렉터리를 만들지 못하면 이벤트 파일도 만들지 않고 emacsclient 도 부르지 않으며 종료 코드는 1 이다.

#### REQ-AIPH-007 [When]

**When `$EMACSCLIENT` is set to a value without a `/`, the CLI shall resolve it only by a `PATH` search in which the first matching entry decides, shall treat a first match in an empty or relative `PATH` entry, or any non-absolute result, as a resolution failure, and on success shall use the absolute path that search returns.**

1. 값에 `/` 가 하나라도 있으면 지금 동작을 유지한다: 실행 가능해야 하고, 절대 경로로 바꿔 쓴다(`./ec`, `bin/ec` 도 여기에 속한다).
2. 빈 값은 설정되지 않은 것으로 보고 SPEC-AGENTIPC-001 REQ-AIPC-013 의 나머지 탐색을 한다(현재 동작).
3. 맨 이름을 현재 작업 디렉터리 기준으로 해석하지 않는다. `PATH` 를 앞에서부터 보며 **처음** 찾아진 항목이 결과를 정한다.
   그 항목이 빈 항목이거나 `.` 같은 상대 항목이면 해석은 실패하며(REQ-AIPH-008), 뒤의 절대 경로 항목으로 넘어가지 않는다.
   이는 Go `exec.LookPath` 의 `exec.ErrDot` 동작과 같다(go1.26.4 `os/exec/lp_unix.go`).
4. 검색이 성공했어도 결과가 절대 경로가 아니면 실패로 본다. 실행 환경의 `GODEBUG=execerrdot=0` 이 `exec.ErrDot` 을 끄더라도
   현재 디렉터리 기준 경로를 쓰지 않기 위한 조건이다.

#### REQ-AIPH-008 [When — event-detected]

**When a bare `$EMACSCLIENT` name is detected not to resolve through that search, the CLI shall fail delivery with exit code 1 and one stderr line naming `EMACSCLIENT` and its value, and shall not create an event file, invoke any program, or fall back to the later discovery candidates.**

1. 나중 후보는 `PATH` 의 `emacsclient` 와 앱 번들 경로들이다(SPEC-AGENTIPC-001 REQ-AIPC-013). 명시적 설정이 틀렸으면 다른 emacsclient 로
   조용히 바꾸지 않는다는 기존 원칙("실행 불가면 전달 실패")을 맨 이름에도 적용한다.

### 3.3 인수 기준 (Acceptance Criteria)

Tier S 이므로 인수 기준을 여기에 둔다. 각 기준은 Given-When-Then 과 결과가 참/거짓으로 갈리는 검증 명령을 가진다.

#### 3.3.0 검증 명령 공통 형식

SPEC-AGENTIPC-001 `acceptance.md` § 0.1·§ 0.2 의 형식을 그대로 쓴다.

```bash
ert_sel() {
  IMOOGI_TEST_USER_DIR="$(mktemp -d)" "${EMACS:-emacs}" --batch -Q \
    --eval "(advice-add 'ert-run-tests-batch-and-exit :filter-args (lambda (_) (list \"$1\")))" \
    -l tests/run.el
}
go test ./internal/agentipc/... -run 'TestAIPHNN' -count=1
```

- ERT 통과 조건: exit 0, 끝줄 `Ran N tests, N results as expected, 0 unexpected`, N ≥ 1. 테스트 이름은 `imoogi-agent-aiphNN-` 로 시작한다.
- Go 통과 조건: exit 0, 출력에 `ok` 이 있고 `no tests to run` 이 없음. 테스트 이름은 `TestAIPHNN` 으로 시작한다.
- 에코 영역·창 상태·로그 관찰 방법은 SPEC-AGENTIPC-001 `acceptance.md` § 0.1 과 같다(`*Messages*` 에 추가된 줄, `delete-other-windows` 로 시작).

#### AC-AIPH-001 — 신뢰할 수 없는 이벤트 파일 거부 (REQ-AIPH-001)

**Given** 유효한 `message` 이벤트(`text` = `"aiph"`)를 담은 이벤트 파일 F 가 모드 0600 으로 있을 때
**When** 다음 상태에서 `imoogi-agent-receive-file` 로 F 를 수신하면
- (a) F 의 모드를 0620 으로 바꿈
- (b) F 의 모드를 0602 로 바꿈
- (c) 모드 0600 그대로, `user-uid` 가 실제 uid 보다 1 큰 값을 돌려주도록 `cl-letf` 로 바꿈(다른 사용자 소유를 흉내)
- (d) 모드 0600 그대로, 아무것도 바꾸지 않음(대조군)

**Then** (a)~(c) 는 `"error:untrusted"` 이고, 그 호출 동안 `insert-file-contents` 계열 호출이 0회이며, `*Messages*` 에 `aiph` 줄이
추가되지 않고, `*imoogi-agent*` 에 줄이 정확히 하나 추가되며 그 줄은 시각 뒤에 ` - project=- session=- untrusted` 를 담는다
(`type`·`project`·`session` 이 모두 `-`, REQ-AIPH-001.2). (d) 는 `"ok"` 이다.

- 검증: `ert_sel "^imoogi-agent-aiph01-"`

#### AC-AIPH-002 — 신뢰 검사의 범위·순서·링크 해석 (REQ-AIPH-002)

**Given** 30-agent 모듈이 로드되어 있을 때
**When** 다음을 수신하면
- (a) AC-AIPH-001 (a) 의 파일 내용과 같은 JSON 문자열을 `imoogi-agent-receive-json` 으로
- (b) 모드 0620 이고 크기가 1,048,577 바이트인 이벤트 파일
- (c) 모드 0620 인 디렉터리 경로
- (d) 모드 0620 인 유효 이벤트 파일을 가리키는 심볼릭 링크(링크는 테스트가 직접 만든다)
- (e) 모드 0600 인 유효 이벤트 파일을 가리키는 심볼릭 링크

**Then** (a) `"ok"`, (b) `"error:untrusted"`(`too-large` 가 아님), (c) `"error:unreadable"`(파일 조건이 먼저),
(d) `"error:untrusted"`, (e) `"ok"` 이다.

- 검증: `ert_sel "^imoogi-agent-aiph02-"`

#### AC-AIPH-003 — payload 크기 상한 초과 시 이벤트 전체 거부 (REQ-AIPH-003)

**Given** 크기가 104,857,601 바이트인 희소 파일 P(`plan.md` § 5.1 의 방법으로 만든다)와 P 를 가리키는 심볼릭 링크 LP 가 있고,
창이 1개이며 P 를 방문 중인 버퍼가 없을 때
**When** 다음 이벤트를 각각 수신하면
- (a) `open-file`, `path` = P
- (b) `goto-location`, `path` = P, `line` = 1
- (c) `artifact-created`, `path` = P, `title` = `"big"`
- (d) `task-finished`, `status` = `"success"`, `artifact` = P
- (e) `open-file`, `path` = LP
- (f) `open-file`, `path` = P 를 가리키는 상대 경로

**Then** (a)~(e) 는 모두 `"error:payload-too-large"` 이고, 각 호출 뒤 `(find-buffer-visiting P)` 가 `nil` 이며, 창 상태(창 수, 선택된 창,
선택된 창의 point)가 호출 전과 같고, `*Messages*` 에 `Agent artifact created` 나 `✓ Agent task finished` 로 시작하는 줄이 추가되지 않으며,
`*imoogi-agent*` 에 `payload-too-large` 를 담은 줄이 호출마다 정확히 하나 추가된다. (f) 는 `"error:bad-path"` 다(경로 정책이 먼저).

- 검증: `ert_sel "^imoogi-agent-aiph03-"`

#### AC-AIPH-004 — 상한 경계와 질문 없는 표시 (REQ-AIPH-004)

**Given** 수신 모듈의 payload 크기 상한 값이 104,857,600 이고(테스트가 값을 확인한다), 전역 `large-file-warning-threshold` 를 1 로 둔 상태에서
**When** 상한을 테스트 안에서 16 으로 동적 바인딩하고 다음을 수신하면
- (a) 정확히 16 바이트 파일 A 로 `open-file`
- (b) 17 바이트 파일 B 로 `open-file`

**Then** (a) 는 `"ok"` 이고 A 가 선택되지 않은 창에 표시되며 질문이 없다(질문이 생기면 `inhibit-interaction` 때문에 `"error:handler"` 가 된다).
(b) 는 `"error:payload-too-large"` 이다. 테스트가 끝나면 `large-file-warning-threshold` 는 원래 값으로 돌아간다.

- 검증: `ert_sel "^imoogi-agent-aiph04-"`

#### AC-AIPH-005 — 새 reason 의 종료 코드 (REQ-AIPH-005)

**Given** emacsclient 결과 분류기가 있을 때
**When** stdout 이 각각 `"error:untrusted"` + `\n`, `"error:payload-too-large"` + `\n` 이고 stderr 가 비어 있으며 exit 0 인 결과를 분류하면
**Then** 둘 다 종료 코드 3 이고, 진단은 한 줄이며 각각 `untrusted`, `payload-too-large` 를 포함한다.

- 검증: `go test ./internal/agentipc/... -run 'TestAIPH05' -count=1`

#### AC-AIPH-006 — 사용자 전용 디렉터리와 정리 (REQ-AIPH-006)

**Given** `TMPDIR` 이 비어 있는 테스트 전용 디렉터리이고 가짜 emacsclient 가 받은 경로의 파일 모드와 그 부모 디렉터리의 모드·이름을 기록할 때
**When** 가짜 emacsclient 가 수락·거부·elisp 오류·연결 실패·30초 멈춤(시간 상한 1초) 중 하나로 응답하는 `Deliver` 를 각각 실행하면
**Then** 매번 기록된 파일 모드는 `600`, 부모 디렉터리 모드는 `700`, 부모 디렉터리 이름은 `imoogi-agent-` 로 시작하고 그 부모는 `TMPDIR` 이며,
`Deliver` 가 돌아온 뒤 `TMPDIR` 에는 아무 항목도 남지 않는다. 연달아 실행한 두 `Deliver` 가 기록한 부모 디렉터리 이름은 서로 다르다.
또한 `TMPDIR` 이 쓰기 불가(0500)이면 종료 코드 1 이고 가짜 emacsclient 는 불리지 않는다.

- 검증: `go test ./internal/agentipc/... -run 'TestAIPH06' -count=1`

#### AC-AIPH-007 — 맨 이름은 PATH 에서만 찾음 (REQ-AIPH-007)

**Given** 실행 가능한 `ec` 가 절대 경로 디렉터리 P 와 현재 작업 디렉터리 D 에 각각 따로 있을 때
**When** 다음 설정으로 emacsclient 를 찾으면
- (a) `EMACSCLIENT=ec`, `PATH=P`
- (b) `EMACSCLIENT=./ec`, `PATH` 는 빈 임시 디렉터리
- (c) `EMACSCLIENT=<P 의 ec 절대 경로>`

**Then** (a) 는 `P/ec`(D 의 `ec` 가 아님), (b) 는 `D/ec` 의 절대 경로, (c) 는 그 절대 경로다.

- 검증: `go test ./internal/agentipc/... -run 'TestAIPH07' -count=1`

#### AC-AIPH-008 — 맨 이름을 찾지 못하면 전달 실패 (REQ-AIPH-008)

**Given** 실행 가능한 `ec` 가 현재 작업 디렉터리 D 에 있고, 절대 경로 디렉터리 E 에는 `emacsclient` 만, 절대 경로 디렉터리 Q 에는 `ec` 만 있으며,
앱 번들 후보 경로에도 `emacsclient` 가 있고, 이들이 모두 실행되면 기록을 남기는 가짜 emacsclient 일 때
**When** 다음 설정으로 `Deliver` 를 실행하면
- (a) `EMACSCLIENT=ec`, `PATH` = E
- (b) `EMACSCLIENT=ec`, `PATH` = `.` 다음 E
- (c) `EMACSCLIENT=ec`, `PATH` = `.` 다음 Q (처음 찾아진 `ec` 가 `.` 에 있음)
- (d) `EMACSCLIENT=ec`, `PATH` = `.`, 테스트 프로세스 환경에 `GODEBUG=execerrdot=0`

**Then** 네 경우 모두 종료 코드 1 이고, 진단은 한 줄이며 `EMACSCLIENT` 와 `ec` 를 포함하고, D·Q 의 `ec` 도 E·앱 번들의 `emacsclient` 도
실행되지 않으며, `TMPDIR` 에 아무 항목도 생기지 않는다. (c) 는 Q 의 `ec` 로 넘어가지 않음을, (d) 는 절대 경로 조건(REQ-AIPH-007.4)을 확인한다.

- 검증: `go test ./internal/agentipc/... -run 'TestAIPH08' -count=1`

#### 3.3.9 Definition of Done

- AC-AIPH-001 ~ 008 이 모두 통과한다.
- SPEC-AGENTIPC-001 의 기존 인수 테스트(`ert_sel "^imoogi-agent-ac"`, `go test ./cmd/imoogi-agent/... ./internal/agentipc/... -count=1`)가
  모두 통과한다. 단, 현재 디렉터리 해석을 고정한 `TestAC14DiscoveryIgnoresDirectoriesAndRelativePath` 의 뒷부분은 REQ-AIPH-007·008 에 맞게 바뀐다.
- `make ci-local` exit 0, `gofmt -l internal/agentipc` 출력 없음.

## 4. Out of Scope

이 SPEC 이 만들지 않는 것.

### Out of Scope — scripts/imoogi-editor

- `scripts/imoogi-editor` 는 수정하지 않는다. 이 스크립트는 `[[ -x "${EMACSCLIENT}" ]]` 로 맨 이름을 현재 디렉터리 기준으로 검사한 뒤
  `exec "${emacsclient}"` 로 PATH 에서 찾아 실행하는 자체 불일치가 있다. 이 SPEC 이후 맨 이름 처리에서 Go CLI 와 스크립트는 의도적으로 다르다.
- 두 구현의 후보 순서 동기화를 지키는 `TestDiscoveryCandidatesMatchImoogiEditor` 는 그대로 둔다.

### Out of Scope — 시그널 처리와 남은 이벤트 파일

- 4차원 리뷰의 Functionality #1·Security #5 지적(CLI 에 시그널 처리가 없어 SIGINT/SIGTERM 때 `defer` 정리가 돌지 않음)은 다루지 않는다.
  이 SPEC 뒤에는 그 경우 사용자 전용 디렉터리와 그 안의 0600 파일이 남을 수 있다. 별도 카드로 다룬다.

### Out of Scope — 경로 허용 목록

- payload 경로의 허용 목록(allowlist)이나 특정 디렉터리 가두기. SPEC-AGENTIPC-001 § 4 "경로 허용 목록" 결정을 그대로 유지한다.
- payload 파일의 소유자·권한 검사. 신뢰 검사는 이벤트 파일에만 적용한다.

### Out of Scope — 더 강한 파일 신뢰 모델

- 이벤트 파일이 들어 있는 디렉터리의 소유자·권한 검사, 검사와 읽기 사이의 경쟁(TOCTOU) 제거, `O_NOFOLLOW` 류의 링크 거부.
- `imoogi-agent-receive-json` 을 부르는 발신자에 대한 신뢰 판단.
- emacsclient 서버 소켓 디렉터리의 권한 점검.

### Out of Scope — 문서와 기타 리뷰 지적

- README 의 종료 코드·reason 절 갱신은 sync 단계에서 한다(`plan.md` § 8).
- 4차원 리뷰의 Craft·Consistency 지적(가짜 emacsclient 중복, `parseRequest` 복잡도, `Deps` 주입 방식, `context.Context` 인자 등)과
  Functionality #0(AC-004 테스트 약점)은 다루지 않는다.
- `$EMACSCLIENT` 가 비었을 때의 기존 `PATH` `emacsclient` 탐색에 절대 경로 가드를 더하는 일(`plan.md` R-8). REQ-AIPH-007.4 는 맨 이름 분기에만 적용한다.

## 5. Constraints

- **망분리**: 새 elpa 패키지와 새 서드파티 Go 모듈을 추가하지 않는다. Go 는 표준 라이브러리만 쓴다.
- **Emacs 버전**: 30.1 이상(SPEC-AGENTIPC-001 과 같음).
- **사용자 상호작용 금지**: 새 검사도 어떤 경로로든 사용자 입력을 기다리지 않는다(REQ-AIPC-008).
- **작업 트리**: `CLAUDE.md` 의 커밋되지 않은 수정분과 다른 SPEC 디렉터리를 건드리지 않는다(`plan.md` § 7 PRESERVE).

## 6. 변경 파일 (Brownfield Delta)

| 표시 | 경로 | 내용 |
|------|------|------|
| [MODIFY] | `modules/development/30-agent.el` | 이벤트 파일 신뢰 검사(`imoogi-agent--read-file`, 47-67행 부근), payload 크기 상한(`imoogi-agent--resolve-path`, 146-157행 부근) |
| [MODIFY] | `tests/agent-test.el` | `imoogi-agent-aiph01-` ~ `aiph04-` 테스트 |
| [MODIFY] | `internal/agentipc/emacsclient.go` | `Deliver`(49-86행)·`writeEventFile`(91-115행)의 사용자 전용 디렉터리, `findEmacsclient`(120-149행, `$EMACSCLIENT` 분기 120-127행)의 맨 이름 탐색 |
| [MODIFY] | `internal/agentipc/emacsclient_test.go` | `TestAIPH05` ~ `TestAIPH08`, 가짜 emacsclient 기록에 부모 디렉터리 정보 추가, 355-362행의 현재 디렉터리 해석 고정 부분 교체 |
| [UNCHANGED] | `scripts/imoogi-editor`, `cmd/imoogi-agent/*`, `internal/agentipc/request.go`, `internal/agentipc/agentipc.go` | 변경 없음 |
| [UNCHANGED] | `go.mod`, `go.sum`, `go.work`, `packages.el`, `packages.lock`, `vendor/` | 망분리 제약 |

## 7. Traceability

| REQ | AC | 카드 |
|-----|----|------|
| REQ-AIPH-001 | AC-AIPH-001 | t18 |
| REQ-AIPH-002 | AC-AIPH-002 | t18 |
| REQ-AIPH-003 | AC-AIPH-003 | t20 |
| REQ-AIPH-004 | AC-AIPH-004 | t20 |
| REQ-AIPH-005 | AC-AIPH-005 | t18, t20 |
| REQ-AIPH-006 | AC-AIPH-006 | t18 |
| REQ-AIPH-007 | AC-AIPH-007 | t19 |
| REQ-AIPH-008 | AC-AIPH-008 | t19 |
