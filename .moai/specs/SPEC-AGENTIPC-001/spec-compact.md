# SPEC-AGENTIPC-001 — Compact

run 단계 적재용 요약. 요구사항 + 인수 기준 + 변경 파일 + 제외 범위만 담는다.
전체 맥락: `spec.md`(계약 § 2), `plan.md`(결정 D-1~D-12, 마일스톤), `acceptance.md`(검증 명령), `research.md`.

- id: `SPEC-AGENTIPC-001` · tier: M · status: draft · phase: `v0.x agent-ipc MVP` · version: `0.1.2`
- 계약 요약: 봉투 `{version:"1", type, timestamp, project?, session?, payload}`, 유형 5개
  (`message`/`open-file`/`goto-location`/`artifact-created`/`task-finished`), 수신기 반환 `"ok"` | `"error:<reason>"`,
  CLI 종료 코드 0 수락 / 1 전달 실패 / 2 잘못된 요청 / 3 거부. 상세 표는 `spec.md` § 2.
- v0.1.1: plan-audit 1회차 지적(D1-D4, O1·O4·O6) 반영. 상태 문자열은 정확히 `"ok"`|`"error:<reason>"`(뒤에 설명 없음),
  Emacs 파싱 옵션 `:object-type 'alist :array-type 'array :null-object :null :false-object :false`.
- v0.1.2: plan-audit 2회차 N1 반영. 알림은 "호출 동안 `*Messages*` 에 추가된 줄 중 정확히 하나"로 판정하고, 파일을 보여 주는 처리기는 표시 뒤에 알림을 낸다.

## Requirements (GEARS, 15)

### 프로토콜과 검증
- **REQ-AIPC-001** [Ubiquitous] 진입점은 `imoogi-agent-receive-file`(경로)과 `imoogi-agent-receive-json`(문자열) 둘뿐이며, 같은 검증·처리 경로를 공유하고 항상 상태 문자열을 돌려준다. 오류를 밖으로 내보내지 않고, 이벤트 파일을 수정·삭제하지 않는다.
- **REQ-AIPC-002** [Ubiquitous] `spec.md` § 2.1–2.2 스키마를 따르는 이벤트만 받는다. 필수 필드의 `null` 은 `bad-field`, `false` 는 참이 아니며, 정의되지 않은 필드는 무시한다. 선택 필드는 키 없음·`null` 만 없음이고, `[]`·`{}`·숫자·불리언 등 형식이 다른 값은 `bad-field`.
- **REQ-AIPC-003** [When — event-detected] 크기(1 MiB)·이벤트 파일 조건·문법·후행 내용·키 중복·스키마 위반은 해당 `error:<reason>` 으로 거부하고, 처리·표시 없이 로그에 거부 줄을 남긴다. 크기는 해석 전에 보고, 위반이 겹치면 파일→크기→문법→후행→중복→버전→유형→필드→경로 순의 첫 위반을 보고한다(원격 경로는 크기를 묻기 전에 거부).

### 처리와 UI
- **REQ-AIPC-004** [When] `message`·`task-finished` 는 에코 영역 한 줄 + 로그 한 줄. 텍스트는 글자 그대로(`%` 해석 없음). 성공 `✓ Agent task finished`, 실패 `✗ Agent task failed`, 뒤에 `summary`. `artifact` 가 있으면 REQ-005 방식으로 표시. 알림·로그 값(`text`·`summary`·`title`·`artifactType`·`project`·`session`·파일 이름)의 줄바꿈(CRLF·LF·CR 각각)은 `⏎`(U+23CE) 하나로 바꿔 항상 한 줄.
- **REQ-AIPC-005** [When] `open-file`·`goto-location` 은 선택되지 않은 다른 창에 표시하고 선택하지 않는다. 선택 창·프레임·사용자 point 불변, 선택되지 않은 창에 이미 보이면 그 창 재사용, `goto` 는 그 창의 point 만 1부터 센 줄·열로 옮기며 범위를 넘으면 `point-max` 줄 / 줄 끝으로 맞춘다. 실제 경로로 연다.
- **REQ-AIPC-006** [When] `artifact-created` 는 `Agent artifact created: <title>`(없으면 파일 이름) 알림 + 로그 + REQ-005 표시. `artifactType` 은 로그에만. 줄바꿈은 REQ-004 규칙.
- **REQ-AIPC-007** [Ubiquitous] 수락·거부 모두 `*imoogi-agent*` 에 한 줄 추가(수신 시각, type, project, session, 결과; 없거나 문자열이 아니면 `-`, 줄바꿈은 `⏎`). 추가 전용, 읽기 전용, 세션 한정.
- **REQ-AIPC-008** [Ubiquitous — negated] 사용자 입력을 기다리지 않고(큰 파일 경고·지역 변수 확인·바뀐 파일 질문 같은 간접 질문 포함), 선택 창·프레임·사용자 point 를 바꾸지 않으며, payload 파일을 쓰거나 실행하지 않고, 이벤트 값을 코드로 평가하지 않으며, `project`/`session` 을 표시 외에 쓰지 않는다(26-project-notes 의존 없음).

### 경로 안전
- **REQ-AIPC-009** [When — event-detected] payload 경로가 상대·원격·없음·(링크 해석 뒤) 일반 파일 아님(디렉터리·장치·FIFO·소켓·끊긴 링크)이면 이벤트 전체를 `error:bad-path` 로 거부하고 아무것도 표시하지 않는다. 원격 판정은 연결을 열지 않는다.

### CLI 와 전송
- **REQ-AIPC-010** [Ubiquitous] § 2.5 문법(`message`/`open-file`/`goto`/`artifact`/`finish`, `--project`/`--session`/`--timeout`, `--`, `--version`)으로 호출당 이벤트 하나를 만든다. `version` "1", RFC 3339 `timestamp`, 상대 경로는 cwd 기준 절대 경로(존재 판정은 Emacs 가 단독), 주어진 선택 필드만, 중복 키 없음, 성공 시 stdout 비움.
- **REQ-AIPC-011** [When — event-detected] 전송 전 잘못된 호출은 exit 2 + stderr 진단 한 줄, 이벤트 파일 생성 없음, emacsclient 호출 없음.
- **REQ-AIPC-012** [Ubiquitous] 0600·CLI 생성 이름의 임시 파일에 쓰고, 셸 없이 emacsclient 를 고정 평가식 + 별도 경로 인자(`server-eval-args-left`)로 부른다. 평가식은 경로를 가장 먼저 꺼내므로 수신 함수가 없어도 경로가 평가되지 않고 `error:not-loaded`(exit 3). CLI 는 모든 경로에서 끝난 뒤 이벤트 파일을 지우고, Emacs 는 지우지 않는다.
- **REQ-AIPC-013** [Ubiquitous] emacsclient 탐색 순서: `$EMACSCLIENT`(실행 불가면 실패) → PATH → `/Applications/Emacs-${EMACS_VERSION:-31.1}.app/…` → `/Applications/Emacs.app/…` → `/Applications/Emacs-*.app/…`. `scripts/imoogi-editor` 와 같다.
- **REQ-AIPC-014** [Ubiquitous] 결과 → 종료 코드: `"ok"` 0 / `"error:…"`·`*ERROR*:`·해석 불가 출력 3 / 찾기 실패·실행 불가·연결 실패·시간 초과·이벤트 파일 I/O 실패 1. 시간 상한 기본 5초, `--timeout` > `IMOOGI_AGENT_TIMEOUT`, CLI 가 직접 강제하고 상한 + 1초 안에 끝난다.

### 빌드·부팅·설치
- **REQ-AIPC-015** [Ubiquitous] `make build-agent`(+ `build-all`·`.PHONY`·`help`), `boot.el` 과 로드 순서 테스트에 `29-org-roam` 다음 `30-agent`, 오프라인 부팅 `:error` 없음, `install.sh` 가 `~/.local/bin/imoogi-agent` 링크 + 셸 테스트, `go.mod`/`go.sum`/`go.work`/`packages.el`/`packages.lock`/`vendor/` 불변.

## Acceptance Criteria (15)

검증 형식: ERT `ert_sel "^imoogi-agent-acNN-"`(`acceptance.md` § 0.1; 에코 영역 알림은 호출 동안 `*Messages*` 에 추가된 줄 중 정확히 하나로 관찰, 다른 줄 추가 허용, 호출 전 `*Messages*` 비움), Go `go test ./cmd/imoogi-agent/... ./internal/agentipc/... -run 'TestACNN' -count=1`.

- **AC-AIPC-001** (001, 004, 007) message: `"ok"`, `%` 글자 그대로, 로그 한 줄, 읽기 전용, 파일/문자열 진입점 동등, 이벤트 파일 불변, `text`·`project` 의 LF/CRLF/CR → `⏎`(알림 한 줄, 로그 정확히 1줄 증가).
- **AC-AIPC-002** (005, 008) open-file: 다른 창 표시, 선택 창·프레임·point 불변, 이미 보이는 창 재사용.
- **AC-AIPC-003** (005, 008) goto-location: (3,4)/(3,-)/(99)/(2,999) 위치 맞춤, 사용자 point 불변.
- **AC-AIPC-004** (006) artifact-created: 제목/파일 이름 알림, 표시, 로그에 artifactType.
- **AC-AIPC-005** (004) task-finished: 성공/실패 문구 + summary, artifact 표시.
- **AC-AIPC-006** (001, 003, 007) 잘림·깊이 초과 `parse`, 후행 `trailing-content`, 봉투·payload 중복 `duplicate-key`, 오류 누출 없음.
- **AC-AIPC-007** (002, 003) 스키마 표 15행(버전·유형·필수 필드·null·false·정수·payload 형식·추가 필드 무시·위반 순서·봉투/payload 선택 필드의 `[]`·`{}`·`false`·숫자 → `bad-field`).
- **AC-AIPC-008** (003) 1 MiB 경계(파일·문자열), 초과 파일은 읽지 않음, 이벤트 파일 없음·상대·디렉터리·TRAMP → `unreadable`.
- **AC-AIPC-009** (009) 경로 정책 9행(일반·링크→실제 경로·상대·없음·디렉터리·디렉터리 링크·끊긴 링크·`/dev/null`·TRAMP) + 4개 경로 필드 공통.
- **AC-AIPC-010** (001, 008) 입력 함수 차단 상태에서 큰 파일·지역 변수·바뀐 파일 → `"ok"`, 질문 0회, eval 미실행, project-notes 호출·문자열 0.
- **AC-AIPC-011** (010) 하위 명령 8종의 이벤트 모양, 절대 경로화, 선택 필드, `--`, `--version`, 중복 키 없음.
- **AC-AIPC-012** (011) 잘못된 호출 16종 → exit 2, stderr 진단 정확히 한 줄, emacsclient·임시 파일 없음.
- **AC-AIPC-013** (012) 인자 정확히 세 개 `--eval EXPR PATH`(`--timeout` 등 추가 인자 없음), 평가식 불변·내용 미포함, 0600, 5가지 결과 모두 파일 삭제, `pop` 이 먼저.
- **AC-AIPC-014** (013, 014) 탐색 7경우, 출력 모양별 종료 코드, 시간 상한(옵션 우선, 기본 5초, 상한 + 1초 안 종료), TMPDIR 쓰기 불가 → 1.
- **AC-AIPC-015** (012, 014, 015) build-agent·build-all·로드 순서 테스트·설치 링크 셸 테스트·의존 불변·`make ci-local` + 실제 30.2 데몬 스모크(0 / 3 bad-path / 3 not-loaded / 1, 파일 잔존 없음).

## 변경 파일

- [NEW] `modules/development/30-agent.el`, `tests/agent-test.el`, `cmd/imoogi-agent/main.go`, `cmd/imoogi-agent/main_test.go`, `internal/agentipc/`
- [MODIFY] `boot.el`, `tests/module-layout-test.el`, `Makefile`, `scripts/install.sh`, `tests/setup-toolchain-test.sh`
- [UNCHANGED] `go.mod`, `go.sum`, `go.work`, `packages.el`, `packages.lock`, `vendor/`, `modules/project/26-project-notes.el`, `modules/general/13-system.el`, `scripts/imoogi-editor`

## 제외 범위

- `progress`·`show-diff` 및 5개 외 이벤트 유형.
- 상주 TCP/Unix 소켓, JSON-RPC, ACP, 토큰 스트리밍, 서버 소켓 설정 변경, 재시도·대기열.
- Emacs → 에이전트 요청, 세션 수명 주기, 에이전트 프로세스 관리, `project`/`session` 기반 프로젝트 조회, `26-project-notes.el` 변경.
- 원격 호스트·TRAMP·SSH 전달, Emacs 30.1 미만, 목록 밖 플랫폼별 탐색 경로.
- Emacs 의 산출물 작성·`artifacts/agent/` 생성·이름 관례 강제, 데스크톱 알림·소리·영구 로그 파일.
- payload 경로의 허용 목록·디렉터리 가두기(containment) 제한 — 사용자 결정. 같은 사용자 권한의 에이전트, 사용자 전용 소켓, 표시 전용 수신기, 허용 목록은 26-project-notes 의존을 부름.
- 프로젝트 문서("여섯 개의 CLI" 등) 갱신 — sync 단계(`plan.md` § 7).
