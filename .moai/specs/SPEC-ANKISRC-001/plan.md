# 구현 계획 — SPEC-ANKISRC-001

설계 결정의 원본은 `.moai/reports/t25/progress.md` D1–D8 이다. 아래 DD 는 그 결정을 구현
수준으로 고정한 것이며, 바뀔 가능성이 큰 것(데이터·인터페이스·UX)부터 적었다. v0.1.1 에서
plan-audit 1회차(`.moai/reports/t25/plan-audit.md`) F1–F16 의 lane 판정을 반영했다.

## § A 맥락 (기준선 `fde2122`)

- 모델: imoogi-Basic `[Front, Back]`, imoogi-Cloze `[Text, Back Extra]` (`internal/anki/model/model.go:87-106`). 필드 추가 경로 없음. stock `Basic`/`Cloze` 도 `renderType`(planner.go:397-406)을 거쳐 같은 `Back`/`Back Extra` 키로 렌더된다.
- planner: `processEntry` 269-372, 렌더 295, 미디어 재작성 316, `hashing.Hash` 324. orphan-delete 재해시 670.
- protocol: `Entry` 149, `Version = 2` (21행), 관대한 decoder. 진단 코드 상수 블록 51-115(설명 주석 44-50), 패키지 주석 4-7("omitempty 없음"), Version 주석 14-20.
- Elisp: 스캔 `imoogi-scan.el:68-115`(디스크를 읽고 `:ID:` 는 읽지 않음), writeback `imoogi-writeback.el:66-138`(7bf2920: `added` 뒤 `org-id-get … 'create`, 버퍼를 닫지 않음), 동기화 `imoogi.el:278-312`, 대상 `imoogi-targets.el`, 메시지 표 `imoogi-error.el:20-88`.
- 모듈 로더: `modules/org/24-anki.el:37` — autoload 쿠키를 수확하지 않으므로 직접 `require` 한다.
- emacsclient 선례: `internal/agentipc/emacsclient.go` (`findEmacsclient` 126, `exec.CommandContext` + 시간 제한 + `WaitDelay` 66-72).
- server-start: `modules/general/13-system.el:112-122` (after-init).
- Org 의 위치 캐시는 `abbreviate-file-name` 형식(`~/…`)으로 경로를 기록한다(org-id.el `org-id-add-location`).

### A.1 PRESERVE

- writeback 의 `added` 뒤 `org-id-get (point) 'create` (멱등 보조 장치, REQ-ASRC-001.4)
- needs-save 정책(수정된 버퍼는 저장하지 않음)
- 관대한 protocol decoder, `Version = 2`
- `hashing.Hash` 입력 순서와 구성(링크는 필드 값의 일부로만 들어감)
- `ANKI_NOTE_ID` 쓰기 경로, 기존 `note_id` 중복 처리 경로(이 SPEC 의 `:ID:` 중복은 그 경로를 쓰지 않는다)

## § B 설계 결정 (가역성이 낮은 순)

### DD-1 — 프로토콜 필드와 예외 기록 (D4)
`Entry` 에 선택 문자열 `org_id`(`omitempty`) 하나. 구 백엔드는 무시하고, 구 프런트엔드는 링크
없이 동작한다. protocol 패키지 주석의 "omitempty 없음"·Version 정책에 이 필드가 의도된 예외임을
근거(관대한 decoder, 선택 필드)와 함께 기록한다(REQ-ASRC-003.1).

### DD-2 — 링크 HTML 형식 (D2, D5)
`<div class="imoogi-src"><a href="imoogi-anki://open?id=<퍼센트 인코딩 ID>">Emacs에서 원본 열기</a></div>`
를 렌더된 뒷면 필드(`Back` / `Back Extra`) 끝에 덧붙인다. ID 는 URL 질의 값 인코딩(`url.QueryEscape`)을
거친다(`:` → `%3A`, `@` → `%40`). 필드가 비어 있으면 앵커만 들어간다. 문자열은 상수 하나로 둔다
(바뀌면 모든 노트가 다시 1회 갱신된다 — R-1).

### DD-3 — 허용 문자 집합
`^[A-Za-z0-9._:@-]{1,128}$`. Go 처리기, planner, Elisp 수신 함수 세 곳이 같은 규칙을 쓴다.
처리기는 질의 값을 퍼센트 디코딩한 뒤 검증한다. `@` 는 `org-id-include-domain` 의 `@FQDN` 을 위한
것이며 Lisp 문자열·HTML 속성 안에서 무해하다. 이 집합 밖의 사용자 `org-id-prefix` 는 R-5.

### DD-4 — 진단 코드 `org_id_invalid` (F3)
`protocol.CodeOrgIDInvalid = "org_id_invalid"` 를 진단 코드 상수 블록에 추가하고(M2), 같은 커밋
범위에서 `imoogi-error-table` 에 메시지 항목을 추가한다(M1). 이름은 기존 `card_option_invalid` 규칙을
따른다. 결과는 `added`/`updated` 그대로, 오류 목록에 key 를 단 이 코드가 함께 실린다.
프런트엔드 전용 보고(중복 `:ID:`, ID 확보 실패)는 Go 코드가 아니며 `binary_not_found` 처럼 Elisp 표
항목 `org_id_duplicated`, `org_id_create_failed` 로 렌더한다.

### DD-5 — 안내 문구와 해석 순서 (D7, UX)
- 0건: `imoogi-anki: Org ID <ID>의 원본 heading을 찾을 수 없습니다. 파일이 옮겨졌거나 삭제·ID 변경되었을 수 있습니다. 확인 후 M-x imoogi-sync 로 다시 동기화하세요.`
- 2건 이상: `*imoogi-anki-open*` 버퍼에 후보(경로, heading 제목)를 나열하고 이동하지 않는다.
- 유일성 확인(F8): 캐시 적중을 파일 안에서 검증한 뒤에도 등록 대상 전체의 동기화 대상 파일을 한 번 스캔해 ID 가 전체에서 유일한지 확인한다. 동기화 시 중복 집합을 따로 저장하는 방식은 새 영속 상태와 무효화 규칙이 필요해 택하지 않았다. 비용은 클릭당 대상 파일 한 번 읽기(텍스트 검색으로 후보 파일을 좁힌 뒤 Org 로 heading-local 여부 확인).
- 모든 결과(열기·0건·2건 이상)에서 수신 함수가 `select-frame-set-input-focus` 로 프레임에 초점을 주고, 처리기도 eval 이 정상 반환하면 Emacs 번들을 활성화한다(F4 b).
- 동기화 보고: 중복 ID, ID 확보 실패, ID 확보 needs-save 를 기존 보고 문자열에 덧붙인다.

### DD-6 — 처리기 실행 경로와 시간 제한 (D6, F4 a, F7)
- 탐색: `findEmacsclient` 규칙. 실패하면 실행 시도 없이 알림.
- 탐침: `emacsclient --eval t`, 호출별 제한 2초. 실패 시 Emacs 실행(exec), 0.5초 간격으로 최대 30초 탐침.
- eval: `(imoogi-anki-open-id "<ID>")`, 호출별 제한 10초. 수신 함수 결과 `"ok"`·`"not-found"`·`"duplicate"` 는 정상 → 코드 0, 알림 없음(정상 반환 뒤 Emacs 번들 활성화). 0 이 아닌 종료, 출력의 `*ERROR*`, 세 값 밖의 결과만 그 내용을 담아 알림 + 0 이 아닌 코드.
- 분류기: M4 는 이 세 정상 값을 아는 자체 분류기를 쓴다. `internal/agentipc` 의 `classify` 는 `"ok"` 와 `error:*` 만 받아 `"not-found"` 를 "unrecognized response"(ExitRejected)로 분류하므로 그대로 재사용하지 않는다(문자열 파싱 보조 함수는 재사용 가능).
- 모든 호출은 `exec.CommandContext` + `WaitDelay`(agentipc 선례). 상한·간격·호출별 제한은 테스트에서 주입한다.
- 알림: `osascript -e 'display notification …'`(인자는 exec 배열, 메시지는 AppleScript 문자열 인용).
- 실행·활성화 대상: `findEmacsclient` 가 앱 번들 안에서 emacsclient 를 찾았으면 그 번들, 아니면 `open -a Emacs`(Emacs Plus 등 여러 빌드가 있을 때 다른 Emacs 를 띄우지 않기 위함).

### DD-7 — 수신 함수 (F4 a)
고정 함수 `imoogi-anki-open-id` 하나. 별도 파일에 두고 `modules/org/24-anki.el` 에서 직접 `require`
해 Anki 모듈 적재 때 함께 올라오게 한다(autoload 쿠키 비의존). 성공·0건·2건 이상 모두 문자열 결과
(`"ok"`, `"not-found"`, `"duplicate"`)를 돌려준다. 처리기는 세 값 모두 정상 종료(코드 0, 알림 없음)로
본다 — `"not-found"`/`"duplicate"` 의 안내는 초점을 받은 Emacs 프레임에 이미 보인다 — 그 밖의 값만 실패로 본다(DD-6).
Go 쪽은 ID 를 Lisp 문자열 인용 규칙(`\` 와 `"` 이스케이프)으로 감싼다.

### DD-8 — `.app` 생성기와 설치 명령 (F15)
`imoogi-anki install-url-handler [--dest DIR] [--no-register]`. 기본 위치 `~/Applications/ImoogiAnki.app`.
AppleScript 원본 → `osacompile` → `Info.plist` 에 `CFBundleIdentifier`, `CFBundleURLTypes` 기록 →
(`--no-register` 가 아니면) `lsregister -f`. shim 은
`do shell script (quoted form of "<절대 경로>") & " open " & (quoted form of theURL)` 형태로, 생성 시점의
`os.Executable` 절대 경로를 고정하며 PATH 를 조회하지 않는다. `Info.plist` 와 AppleScript 원본 생성은
순수 함수로 분리해 macOS 도구 없이 테스트한다. 외부 명령 실행은 주입 가능한 runner 를 거친다.

### DD-9 — ID 확보 단계 (D1, F6, F13)
`imoogi-sync` 에서 대상 해석 뒤, 스캔 전에 실행한다. legacy 경로와 등록 대상 경로 모두.
heading-local 조회는 `(org-entry-get (point) "ID" nil)` 형태(상속 끔). 탐지는 스캔처럼 임시
버퍼에서 하고, ID 없는 heading 이 1개 이상인 파일만 방문 버퍼(기존 것 재사용, 없으면 새로 염)로
연다. 이 단계가 새로 연 버퍼는 저장 후 닫고, 사용자가 이미 방문 중이던 버퍼는 닫지 않는다.
파일 단위로 `condition-case` 를 걸어 실패를 그 파일에 가두고 `org_id_create_failed` 로 보고한다.
실패 경로에서 이 단계가 연 버퍼는 `set-buffer-modified-p nil` 로 이 단계의 변경을 버린 뒤 닫아,
"modified; kill anyway?" 같은 질문이 동기화 도중 뜨지 않게 한다. 사용자가 열어 둔 버퍼는 실패해도
닫지 않는다. 이 단계는 위치를 등록만 하고 저장하지 않는다.

### DD-10 — 위치 캐시 저장 시점 (R3)
`org-id-locations-save` 는 `imoogi-sync` 끝에서 한 번 부른다. 그 전에 보낸 모든 entry 의 ID(이미
디스크에 있던 ID 포함)를 REQ-005 대로 `org-id-add-location` 으로 등록한다.

## § C 사전 점검

- `IMOOGI_ERT_SELECTOR=anki emacs --batch -Q -l tests/run.el` → 33 기대대로(기준선)
- `go test ./internal/anki/... ./cmd/imoogi-anki/...` → ok(기준선)
- 새 ERT 파일은 `anki-` 접두사를 써서 같은 선택자로 실행되게 한다.

## § D 제약

- shell 경유 실행 금지(AppleScript shim 의 `do shell script` + `quoted form of` 는 예외로 한정).
- 테스트는 `org-id-locations-file` 을 임시 경로로 묶고 끝나면 정리한다.
- 실제 `lsregister`, `osacompile`, Emacs.app 실행, `osascript` 는 자동 테스트에서 부르지 않는다(fake runner).

## § E 자기 검증 (run 단계 산출물)

AC 매트릭스(`acceptance.md` AC 001–016)의 각 행에 실행 명령과 출력 원문을 `progress.md` §E.2 에 남긴다.
Definition of Done 의 운영자 확인은 따로 표시한다.

## § F 마일스톤

### M1 — Elisp ID 확보 + 스캔 + 프로토콜 (우선순위 높음)
REQ 001–005, DD-1, DD-4(Elisp 표 항목), DD-9, DD-10. ID 확보 단계와 실패 격리, 스캔의 `:ID:` 읽기와
`org_id` 직렬화, 중복 판정과 보고, `org-id-locations` 등록·저장, Go `Entry.OrgID` 와 패키지 주석.
AC 001–005, 008.

### M2 — Go planner 링크 삽입 (우선순위 높음)
REQ 006–008, DD-2, DD-3, DD-4(Go 상수). 렌더·미디어 뒤 해시 전 삽입(stock 포함), 문자 집합 검증과
`org_id_invalid`, 해시 일관성, 1회 갱신 테스트. AC 006, 007.

### M3 — 열기 쪽 Elisp 해석기 (우선순위 중간)
REQ 011, 012, DD-5, DD-7. 캐시 검증 → 전체 유일성 확인/재스캔 → 0/1/2+ 처리, 프레임 초점,
`24-anki.el` 에서의 `require`. AC 009–013.

### M4 — Go `imoogi-anki open` + `.app` 생성/설치 (우선순위 중간)
REQ 009, 010, 013, DD-3, DD-6, DD-8. emacsclient 탐색은 `internal/agentipc` 규칙 재사용
(필요하면 내보내기). eval 결과는 DD-6 의 자체 분류기로 판정한다. AC 014–016.

### M5 — 문서 (우선순위 낮음)
README 의 Anki 절: 설치 명령, 1회 전체 갱신 안내, 중복 ID·needs-save·ID 확보 실패 안내,
운영자 확인 절차(Definition of Done).

## § G 위험

- R-1 링크 문자열이 바뀌면 전체 노트가 다시 1회 갱신된다 → DD-2 상수 하나, 테스트로 고정.
- R-2 첫 동기화 때 기존 노트 전체 갱신 부하 → 의도된 귀결(D2), README 에 명시.
- R-3 Emacs 실행 후 init 이 오래 걸리면 30초 상한 초과 → 알림 후 재클릭 안내.
- R-4 Anki 리뷰어가 사용자 정의 scheme 링크를 OS 로 넘기는지는 기능 전체의 전제지만 자동 검증이 불가하다. 이 SPEC 은 계획을 바꾸지 않고(F16 판정) 이를 운영자 확인 공백으로 남기며, Definition of Done 의 운영자 확인에서만 검증된다. 넘기지 않는 것으로 확인되면 M4·M5 산출물의 가치가 사라지므로 그 결과를 먼저 기록한다.
- R-5 다른 도구가 `:ID:` 를 이미 쓰는 heading, 또는 집합 밖 문자를 가진 사용자 `org-id-prefix` → 그대로 재사용(재작성 금지), 문자 집합 밖이면 링크 없이 `org_id_invalid`.
- R-6 열 때의 전체 유일성 스캔 비용이 대상 파일 수에 비례한다 → 텍스트 검색으로 후보를 좁힌다(DD-5).
- R-7 eval 호출별 제한 10초 안에 전체 유일성 스캔이 끝나야 한다. 넘기면 emacsclient 는 종료되지만 Emacs 는 계속 처리해 결국 이동하므로, 사용자는 실패 알림 뒤에 성공한 열기를 보게 된다. 완화: 유일성 스캔은 파일마다 Org 전체 파싱이 아니라 원문 버퍼에서 `re-search-forward` 로 `:ID:[ \t]+<정규식 인용된 ID>$` 를 찾고, 맞은 위치에서만 heading-local 여부를 확인한다. M3 에서 대상 파일 수에 대한 소요 시간을 측정해 `progress.md` 에 남긴다.

## § H 테스트 방법

- Elisp: ERT, 바이너리는 기존 테스트처럼 stub, `org-id-locations-file` 은 `make-temp-file`. 메모리의
  `org-id-locations` · `org-id-files` 도 테스트마다 let 으로 묶어 초기화한다(AC 009·010 상호 오염 방지).
  경로 비교는 `file-truename` 동치. 프레임 초점 함수는 stub 으로 호출 수를 센다.
- Go: planner 는 기존 fake AnkiConnector, 처리기·설치기는 주입 runner 로 호출 인자·시간 제한을 기록.
