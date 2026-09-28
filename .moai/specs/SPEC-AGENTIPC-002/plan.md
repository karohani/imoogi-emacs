# SPEC-AGENTIPC-002 — Implementation Plan

Tier S. 요구사항 8개, 인수 기준 8개(둘 다 `spec.md` § 3). 이 문서는 **어떻게**를 다룬다.
결정은 바뀔 가능성이 큰 것부터 적고, 기계적인 작업은 뒤에 둔다.

## 1. 맥락

- 기준 커밋: `main` @ `ecf68a9` (= `origin/main`). 작업 트리에는 이 SPEC 과 무관한 수정분(`CLAUDE.md`)과 추적되지 않는 `.moai/` 파일들이 있다.
- 선행 SPEC: SPEC-AGENTIPC-001(`completed`). 이 SPEC 은 그 계약(§ 2.3 reason 표, § 2.4 종료 코드, REQ-AIPC-003.5 순서)을 넓힌다.
- 근거: `.moai/reports/sync-audit/SPEC-AGENTIPC-001-4dim.json` findings Security #2·#3·#4, 백로그 카드 t18·t19·t20.
- plan 단계 측정(요약, 원문은 `spec.md` HISTORY):

| 항목 | 측정 | 관찰 |
|------|------|------|
| 소유자·모드 원시 함수 | Emacs 30.2 batch, `make-temp-file` 파일 | 모드 `600`, `file-attribute-user-id`(`'integer`) `501` = `(user-uid)` `501` |
| 그룹·기타 쓰기 판정 | `set-file-modes` `#o620` / `#o602` 뒤 `(logand (file-modes f) #o022)` | 둘 다 0 아님 |
| `user-uid` 바꿔치기 | 바이트 컴파일된 호출자 + `cl-letf` `symbol-function` | 바뀐 값 `4242` 반환 |
| Go 툴체인 | `go version`, `go.mod` | `go1.26.4`, `go 1.26` → `exec.ErrDot` 기본 적용 |
| 희소 파일 생성 | `dd … seek=104857601` | plan 작성 단계 셸이 거부. plan-audit 1회차가 Emacs 30.2 batch `call-process` 로 측정: `dd exit 0`, 크기 `104857601`(R-3) |
| `LookPath` 규칙 | go1.26.4 `os/exec/lp_unix.go` 읽기 | 처음 찾아진 항목에서 멈춤, 상대 항목이면 `ErrDot`, `execerrdot=0` 이면 상대 경로를 오류 없이 반환(D-4) |

## 2. 결정 사항 (바뀔 가능성이 큰 순서)

### D-1 신뢰 검사의 대상은 "읽힐 파일"

이벤트 파일 경로가 심볼릭 링크이면 링크를 해석한 실제 파일의 소유자·모드를 본다(`spec.md` REQ-AIPH-002.1).

- 이유: 보호하려는 것은 "읽어서 이벤트로 처리할 내용"이다. 이 검사가 막는 것은 **다른 사용자가 직접 쓸 수 있는 파일**을 읽는 일이다
  (다른 사용자 소유이거나 그룹·기타 쓰기 가능). 링크 자체를 보면, 사용자 소유 링크가 다른 사람이 쓸 수 있는 파일을 가리키는 경우를 놓친다.
- 한계(v0.1.1, plan-audit D2): 사용자 소유이고 다른 사람이 쓸 수 없는 파일이라도 내용은 공격자가 고른 것일 수 있다(예: 신뢰할 수 없는
  저장소를 받아 둔 파일). 공유 `/tmp` 에서 다른 사용자가 지워진 CLI 디렉터리와 같은 이름의 디렉터리를 다시 만들고 그 안의 `event.json` 을
  그런 파일로 가는 심볼릭 링크로 두면, 늦게 읽는 수신기는 신뢰 검사를 통과시킨다. 잔여 위험 R-7 참조.
- 대안(링크를 아예 거부)은 기존 동작을 바꾸고 CLI 가 만들지 않는 경우를 새로 막는 것이라 택하지 않았다.
- 구현: `(file-truename file)` 의 `(file-attributes … 'integer)` 에서 uid, `(file-modes …)`(기본적으로 링크를 따라감)에서 모드.

### D-2 검사 순서: 파일 조건 → untrusted → too-large

사용자 결정은 "기존 절대/로컬/일반/읽기 가능 검사 뒤, 읽기 전"이다. 크기 검사도 읽기 전이므로 둘의 순서를 따로 정했다:
신뢰 검사가 크기 검사보다 먼저다(`spec.md` § 2.3). 신뢰할 수 없는 파일에 대해서는 크기 판정조차 결과로 내보내지 않는다.

### D-3 payload 크기 상한은 검증 단계, 고정 상수

- 위치: `imoogi-agent--resolve-path`(146-157행)에서 `file-regular-p` 검사를 통과한 실제 경로에 대해 `file-attribute-size` 를 본다.
  `imoogi-agent--display`/`--visit` 안에 두면 `task-finished` 가 이미 알림을 낸 뒤에 거부하게 되어 "이벤트 전체 거부"가 깨진다.
- 값: `(defconst imoogi-agent-max-payload-bytes 104857600 …)`. 기존 `imoogi-agent-max-bytes`(19행)와 나란히 둔다.
  `large-file-warning-threshold` 를 읽지 않는다(수신기는 그 변수를 `nil` 로 바인딩하고, 사용자가 값을 바꿀 수 있다).
- 경계: `>` 로 비교한다(정확히 상한이면 수락). 기존 `too-large` 와 같은 규칙이다.
- `imoogi-agent--visit`(242-250행)의 `large-file-warning-threshold nil` 바인딩은 그대로 둔다(REQ-AIPH-004).

### D-4 맨 이름 판정은 `/` 포함 여부, 해석은 `exec.LookPath`

- `strings.ContainsRune(explicit, '/')` 가 거짓이면 맨 이름이다(이 저장소의 대상 플랫폼은 macOS·Linux).
- 맨 이름: `exec.LookPath(explicit)`. 오류(찾지 못함, `exec.ErrDot` 포함)면 `EMACSCLIENT not found in PATH: <값>` 같은 한 줄 진단으로
  즉시 실패한다. 나중 후보로 넘어가지 않는다(REQ-AIPH-008).
- `exec.LookPath` 는 `PATH` 를 앞에서부터 보고 처음 찾아진 항목에서 멈춘다. 그 항목이 상대 경로(빈 항목은 `.`)이면 `ErrDot` 과 함께 돌아오며
  뒤의 절대 경로 항목을 보지 않는다(go1.26.4 `os/exec/lp_unix.go`). 이것이 REQ-AIPH-007.3 의 규칙이다.
- 성공한 결과에 `filepath.IsAbs(found)` 를 한 번 더 요구하고, 아니면 찾지 못한 것과 같이 실패한다(REQ-AIPH-007.4). `GODEBUG=execerrdot=0`
  이면 `LookPath` 가 상대 경로를 오류 없이 돌려주기 때문이다(`lp_unix.go` 의 `execerrdot.Value() != "0"` 분기). 같은 가드를 기존
  `PATH` 의 `emacsclient` 탐색(128-131행)에도 두는 것은 이 SPEC 범위 밖이며 잔여 위험 R-8 로 남긴다.
- `/` 포함: 지금 코드(`isExecutable` → `filepath.Abs`) 그대로.
- 대안(맨 이름 거부)은 `EMACSCLIENT=emacsclient-30` 같은 정상 사용을 막으므로 택하지 않았다.

### D-5 사용자 전용 디렉터리

- `os.MkdirTemp("", "imoogi-agent-*")`(Go 문서상 모드 0700) 안에 `os.OpenFile(filepath.Join(dir, "event.json"), O_WRONLY|O_CREATE|O_EXCL, 0o600)`
  로 파일을 만들고, 기존처럼 `f.Chmod(0o600)` 로 umask 영향을 없앤다.
- 정리: `defer os.RemoveAll(dir)` 하나로 파일과 디렉터리를 함께 지운다. 디렉터리를 만든 뒤의 모든 실패 경로에서도 돌도록,
  `writeEventFile` 은 디렉터리가 만들어졌으면 오류와 함께 디렉터리 경로를 돌려준다(지금 "파일이 만들어졌으면 경로를 돌려준다"와 같은 방식).
- 두 겹 방어의 관계: 시간 초과 뒤 CLI 가 디렉터리를 지웠는데 다른 사용자가 공유 `/tmp` 에 같은 이름의 디렉터리와 파일을 다시 만들면,
  늦게 읽는 수신기는 그 파일의 소유자가 다르므로 `untrusted` 로 거부한다(REQ-AIPH-001). CLI 쪽 디렉터리는 CLI 가 살아 있는 동안의
  바꿔치기를, 수신기 쪽 검사는 CLI 가 끝난 뒤 **다른 사용자가 직접 쓸 수 있는 파일**로의 바꿔치기를 막는다. 사용자 소유 파일로 가는
  심볼릭 링크로의 바꿔치기는 막지 못한다(D-1 한계, R-7).
- 디렉터리 이름은 `os.MkdirTemp` 가 호출마다 무작위로 정하므로 REQ-AIPH-006.1 의 "예측 가능한 고정 이름을 쓰지 않음"을 만족한다.

### D-6 종료 코드는 코드 변경 없음

`classify`(`emacsclient.go` 158-180행)는 `"error:"` 로 시작하는 모든 상태를 3 으로 분류한다. 새 reason 을 위해 Go 분류 코드를 바꾸지 않고,
두 reason 을 고정하는 테스트만 더한다(REQ-AIPH-005).

## 3. 기술 접근

### 3.1 `modules/development/30-agent.el`

- `imoogi-agent--read-file`(47-67행): 기존 `unreadable` 조건 뒤, 크기 검사 앞에 신뢰 검사를 넣는다.

  ```elisp
  (let* ((true (file-truename file))
         (uid (file-attribute-user-id (file-attributes true 'integer)))
         (modes (file-modes true)))
    (unless (and (eql uid (user-uid)) modes (zerop (logand modes #o022)))
      (imoogi-agent--reject "untrusted")))
  ```

  docstring 에 순서(파일 조건 → 신뢰 → 크기 → 읽기)와 D-1 의 링크 해석을 적는다. 크기 검사는 기존대로 `file` 에 대해 해도 되지만,
  같은 `true` 를 쓰면 판정 대상이 하나로 모인다.
- `imoogi-agent--resolve-path`(146-157행): `file-regular-p` 통과 뒤
  `(when (> (file-attribute-size (file-attributes true)) imoogi-agent-max-payload-bytes) (imoogi-agent--reject "payload-too-large"))`.
  docstring 에 이벤트 전체 거부와 검사 위치(dispatch 전)를 적는다.
- 파일 머리 주석(3-11행)에는 reason 목록이 없으므로 고치지 않는다. 신뢰 검사 앞에 `@MX:NOTE` 로 위협 모델(공유 TMPDIR, SPEC-AGENTIPC-001 D-10)을 남긴다.

### 3.2 `internal/agentipc/emacsclient.go`

- `writeEventFile`(91-115행) → 디렉터리 생성 + 파일 쓰기. 반환값을 `(dir, path string, err error)` 로 바꾸거나 디렉터리만 돌려받아
  `Deliver` 에서 `path = filepath.Join(dir, "event.json")` 로 만든다. 어느 쪽이든 절대 경로를 보장한다(`filepath.Abs`).
- `Deliver`(49-86행): `defer os.Remove(path)` → `defer os.RemoveAll(dir)`. 주석과 `@MX:ANCHOR` 설명의 "event file lifetime" 을
  "event directory lifetime" 으로 고친다.
- `findEmacsclient`(120-149행, 바꾸는 곳은 `$EMACSCLIENT` 분기 120-127행): D-4 분기. 함수 주석의 "$EMACSCLIENT (must be executable, no fallback)" 을
  "a bare name is looked up in PATH only; a name with a slash must be executable" 로 고친다.

### 3.3 테스트

- ERT(`tests/agent-test.el`): 기존 도우미(`imoogi-agent-test--write`, `--receive-file`, `--receive-json`, `--window-state`,
  `--kill-visiting`, `--with-temp-files`)를 재사용한다. 새 테스트는 파일 끝 `ac10` 뒤에 `;;; SPEC-AGENTIPC-002` 절로 모은다.
- Go(`internal/agentipc/emacsclient_test.go`): `fakeRecord` 에 `DirPerm`, `DirName`, `DirParent` 필드를 더하고 `fakeEmacsclient` 가
  `filepath.Dir(path)` 를 `os.Stat` 해 채운다. `Deliver` 가 돌아온 뒤에는 디렉터리가 이미 없으므로 0700 은 가짜만 관찰할 수 있다.

## 4. 마일스톤 (우선순위 순)

### M1 — Emacs 수신기: 신뢰 검사(t18a) + payload 크기 상한(t20) · Priority High

1. RED: `imoogi-agent-aiph01-untrusted-event-file`, `imoogi-agent-aiph02-trust-scope-order-and-links`,
   `imoogi-agent-aiph03-payload-too-large-rejects-whole-event`, `imoogi-agent-aiph04-payload-cap-boundary-without-prompt` 를 쓰고 실패를 확인한다.
2. GREEN: § 3.1 의 두 곳을 고친다.
3. 회귀: `ert_sel "^imoogi-agent-"`(SPEC-AGENTIPC-001 의 `ac01`~`ac10` 포함) 전부 통과.

### M2 — Go CLI: 사용자 전용 디렉터리(t18b) + 맨 이름 탐색(t19) · Priority High

1. RED: `TestAIPH05ClassifiesNewReasons`, `TestAIPH06PrivateDirectoryAndCleanup`, `TestAIPH07BareNameUsesPathOnly`,
   `TestAIPH08BareNameNotFoundFailsDelivery` 를 쓰고 실패를 확인한다(`TestAIPH05` 는 현재 코드로도 통과할 수 있다 — 계약 고정용이며 E8 에 그렇게 적는다).
2. `TestAC14DiscoveryIgnoresDirectoriesAndRelativePath` 355-362행(맨 이름 `ec` 가 현재 디렉터리 기준 절대 경로로 풀리기를 기대)을
   "`ec` 가 현재 디렉터리에만 있으면 오류"로 바꾼다. 앞부분(디렉터리 이름 `emacsclient` 거부)은 그대로 둔다.
3. GREEN: § 3.2.
4. 회귀: `go test ./cmd/imoogi-agent/... ./internal/agentipc/... -count=1`, `gofmt -l internal/agentipc`, `go vet ./internal/agentipc/...`.
   `TestAC14UnwritableTempDirIsNotDelivered` 는 이제 `os.MkdirTemp` 단계에서 실패하므로 그대로 통과해야 한다.
   `TestAC13DeliveryArgumentsPermissionsAndCleanup` 의 "파일이 남지 않음" 판정도 그대로 성립해야 한다.
   `cmd/imoogi-agent/main_test.go` 는 고치지 않는다: plan 단계 grep 결과 이 파일은 `EMACSCLIENT` 에 절대 경로만 넣고(91·364행),
   CLI 가 끝난 뒤 `TMPDIR` 이 비었는지 본다(159-160·267-268행). 디렉터리째 지우면 이 판정은 그대로 성립하므로 회귀 확인용으로 함께 돌린다.

M1 과 M2 는 서로 다른 파일만 고치므로 순서 의존이 없다. 마지막에 `make ci-local` 을 한 번 돌린다.

## 5. 테스트 방법 세부

### 5.1 100 MiB 초과 파일을 매번 100 MiB 쓰지 않고 만들기

- 1순위: 희소 파일. `(call-process "dd" nil nil nil "if=/dev/zero" (concat "of=" p) "bs=1" "count=0" "seek=104857601")`.
  `file-attribute-size` 는 논리 크기를 보고하므로 디스크를 거의 쓰지 않고 104,857,601 바이트 파일이 된다. 테스트는 만든 뒤
  `(should (= 104857601 (file-attribute-size (file-attributes p))))` 로 전제를 확인한다. 거부 경로에서는 파일을 읽지 않으므로 빠르다.
- 경계(AC-AIPH-004)는 실제 100 MiB 파일을 열지 않도록 상한 상수를 16 으로 동적 바인딩해 16/17 바이트 파일로 확인한다.
  `defconst` 로 정의한 특수 변수는 `let` 으로 동적 바인딩된다. 상수 값 자체는 `(should (= imoogi-agent-max-payload-bytes 104857600))` 로 고정한다.
- 대안(`dd` 를 쓸 수 없을 때): AC-AIPH-003 도 상한을 작게 바인딩한 작은 파일로 시험하고, 실제 값은 상수 비교로만 고정한다.
  이 경우 run 단계 보고서에 대안을 썼다고 적는다.

### 5.2 "다른 사용자 소유"를 root 없이 시험

- `(cl-letf (((symbol-function 'user-uid) (lambda () (1+ real-uid)))) …)` 로 모든 파일이 남의 것으로 보이게 한다(plan 단계 측정으로
  바이트 컴파일된 호출자에서도 동작 확인). `real-uid` 는 `cl-letf` 밖에서 미리 구한다.
- 그룹·기타 쓰기 가능은 진짜 `set-file-modes` `#o620`·`#o602` 로 만든다.
- 파일 읽기가 없었음은 SPEC-AGENTIPC-001 `ac08` 처럼 `insert-file-contents`·`insert-file-contents-literally` 호출 횟수를 세어 확인한다.

### 5.3 Go 맨 이름 시험

- `installExecutable` 로 `P/ec`, `D/ec` 를 만든다. `t.Chdir(D)`, `t.Setenv("PATH", P)`.
- `ErrDot` 경우(AC-AIPH-008 (b)(c)): `t.Setenv("PATH", "."+string(os.PathListSeparator)+E)`(E 에는 `emacsclient` 만) 또는
  `"."+sep+Q`(Q 에는 `ec`) → `findEmacsclient` 오류, 진단에 `EMACSCLIENT` 와 `ec` 포함, E 의 `emacsclient` 나 Q 의 `ec` 로 넘어가지 않음.
- 절대 경로 가드(AC-AIPH-008 (d)): `t.Setenv("GODEBUG", "execerrdot=0")`, `PATH` = `.`. go1.26.4 `runtime/runtime.go` 의
  `syscall_runtimeSetenv`(202-209행)가 `GODEBUG` 변경을 곧바로 반영하므로 `t.Setenv` 로 충분하다(plan-audit 2회차가 go1.26.4 에서 실행해 확인).
  이때 `LookPath` 는 `ec`(`./` 없이)를 오류 없이 돌려주고, `filepath.IsAbs` 가드가 실패로 바꿔야 한다.
- "어떤 프로그램도 실행되지 않음"은 `useFake` 의 기록 파일이 생기지 않았음으로 확인한다. 이를 위해 `D/ec` 와 `PATH` 의 `emacsclient`
  (앱 번들 후보도 마찬가지)를 빈 스크립트가 아니라 테스트 바이너리를 가리키는 심볼릭 링크로 만든다. 그래야 기록 파일이 없다는 사실 하나로
  어느 쪽도 실행되지 않았음이 관찰된다.

## 6. 위험

| # | 위험 | 대응 |
|---|------|------|
| R-1 | umask 002 같은 환경에서 다른 방법으로 만든 이벤트 파일이 그룹 쓰기 가능이 되어 `untrusted` 로 새로 거부됨 | CLI 는 0600 을 강제하므로 정상 경로는 영향 없음. 기존 ERT 픽스처는 `make-temp-file`(측정: 0600) + `write-region`(기존 모드 유지)이라 통과. 직접 `imoogi-agent-receive-file` 을 부르는 다른 발신자에게는 동작 변경이므로 sync 단계 README 에 적는다 |
| R-2 | `EMACSCLIENT` 에 맨 이름을 두고 현재 디렉터리의 실행 파일에 기대던 사용 | 의도된 동작 변경. 진단 한 줄이 변수 이름과 값을 알려 준다. `./ec` 처럼 `/` 를 넣으면 지금처럼 쓸 수 있다 |
| R-3 | 희소 파일 생성(`dd`)을 plan 작성 단계에서 측정하지 못함 | **해소**(v0.1.1): plan-audit 1회차가 Emacs 30.2 batch 에서 `(call-process "dd" … "seek=104857601")` → `dd exit 0`, `size 104857601` 을 측정했다. § 5.1 의 1순위 방법을 쓰고, 대안은 비상용으로만 남긴다 |
| R-4 | 디렉터리 생성 뒤 파일 쓰기 실패 경로의 정리는 테스트로 일으키기 어려움 | 정리를 `defer os.RemoveAll(dir)` 하나로 모아 경로별 분기를 없앤다. 코드 리뷰로 확인하고 잔여 위험으로 남긴다 |
| R-5 | 시그널로 CLI 가 죽으면 디렉터리가 남음 | `spec.md` § 4 Out of Scope. 남아도 0700 디렉터리 안의 0600 파일이며, 늦은 읽기는 수신기 신뢰 검사가 막는다 |
| R-6 | 신뢰 검사와 실제 읽기 사이의 경쟁(TOCTOU) | `spec.md` § 4 Out of Scope. CLI 쪽 사용자 전용 디렉터리가 CLI 수명 동안의 바꿔치기를 막는다 |
| R-7 | CLI 가 디렉터리를 지운 뒤 다른 사용자가 같은 이름의 디렉터리를 공유 `/tmp` 에 다시 만들고 `event.json` 을 사용자 소유이지만 공격자가 내용을 고른 파일(예: 신뢰할 수 없는 저장소의 파일)로 가는 심볼릭 링크로 두면, 늦은 읽기가 신뢰 검사를 통과한다 | 영향은 표시·알림·로그뿐이다(이벤트 값은 코드로 평가되지 않음, SPEC-AGENTIPC-001 REQ-AIPC-008). 완화책(디렉터리 소유자 검사, 링크 거부)은 `spec.md` § 4 "더 강한 파일 신뢰 모델" Out of Scope 에 둔다. per-user TMPDIR(macOS 기본 `drwx------`)에서는 성립하지 않는다 |
| R-8 | 기존 `PATH` 의 `emacsclient` 탐색(128-131행)은 `GODEBUG=execerrdot=0` 에서 상대 경로를 받아들일 수 있음 | 이 SPEC 은 맨 이름 분기에만 절대 경로 가드를 둔다(D-4). 기존 탐색 가드는 별도 카드로 다룬다 |

## 7. PRESERVE (건드리지 않음)

- `scripts/imoogi-editor`
- `CLAUDE.md`(커밋되지 않은 수정분 포함)
- `go.mod`, `go.sum`, `go.work`, `packages.el`, `packages.lock`, `vendor/`
- `.moai/specs/` 아래 다른 SPEC 디렉터리(SPEC-AGENTIPC-001 포함)
- `modules/project/*`
- `cmd/imoogi-agent/*`, `internal/agentipc/request.go`, `internal/agentipc/agentipc.go`(바꿀 필요가 없음; 필요해지면 blocker 로 보고)
- 런타임 관리 파일(`.moai/state/`, `.moai/cache/`, `.moai/logs/`)

## 8. sync 단계 후속 작업

- README 의 `imoogi-agent` 종료 코드·reason 절에 `untrusted`(이벤트 파일 소유자·모드)와 `payload-too-large`(payload 파일 100 MiB 초과)를
  더하고, 둘 다 종료 코드 3 임을 적는다.
- README 의 `EMACSCLIENT` 설명에 "경로 구분자가 없는 이름은 PATH 에서만 찾는다"를 더한다.
- R-1 의 동작 변경(직접 `imoogi-agent-receive-file` 을 부르는 경우)을 README 에 한 줄 적는다.
- 카드 t18·t19·t20 을 완료로 옮긴다.
