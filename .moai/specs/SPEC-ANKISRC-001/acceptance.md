# 인수 기준 — SPEC-ANKISRC-001

모든 자동 기준은 `org-id-locations-file` 을 임시 경로로, 메모리의 `org-id-locations` ·
`org-id-files` 를 테스트마다 초기화된 값으로 격리한다. 경로 비교는 `file-truename`
(`expand-file-name` 포함) 동치로 한다. 실행 명령:

- Elisp: `IMOOGI_ERT_SELECTOR=anki emacs --batch -Q -l tests/run.el`
- Go: `go test ./internal/anki/... ./cmd/imoogi-anki/...`

## § D AC 매트릭스

#### AC-ASRC-001 — 새 heading 의 첫 동기화, 제외 파일 불변 (REQ 001, 001.5, 003)
Given ID 가 없는 `ANKI_NOTE_TYPE` heading 과, ID 를 가진 부모 아래의 자식 heading 이 저장된 파일에 있고, ID 없는 `ANKI_NOTE_TYPE` heading 을 가진 다른 파일이 제외 패턴에 걸려 있고
When `imoogi-sync` 가 (stub 바이너리로) 실행되면
Then 앞의 두 heading 모두 자기 drawer 에 서로 다르고 부모와도 다른 `:ID:` 를 디스크에 가지며, 요청 entry 의 `org_id` 가 각 heading 의 ID 와 같다. 제외된 파일은 `:ID:` 를 얻지 않고 내용과 수정 시각이 바뀌지 않는다.

#### AC-ASRC-002 — 기존 ID·노트 ID 보존 (REQ 001.2, 002)
Given `:ID:` X 를 가진 heading 과, `ANKI_NOTE_ID` 는 있고 `:ID:` 는 없는 heading 이 있고
When 동기화와 `added` writeback 이 실행되면
Then X 는 바이트 단위로 그대로이고, 두 번째 heading 은 새 `:ID:` 만 얻으며 `ANKI_NOTE_ID` 값은 바뀌지 않는다. writeback 은 `:ID:` 를 바꾸지 않는다.

#### AC-ASRC-003 — 수정된 버퍼와 ID 확보 실패 (REQ 001.3, 001.6, 003.2)
Given (a) 대상 파일을 방문 중인 버퍼에 저장되지 않은 사용자 편집과 ID 없는 heading 이 있고, (b) ID 없는 heading 을 가진 다른 대상 파일이 읽기 전용이며, (c) 문제없는 세 번째 대상 파일이 있고
When 동기화가 실행되면
Then (a) 그 버퍼는 저장되지 않고 사용자 편집이 남아 있으며 needs-save 로 보고되고 그 heading 의 entry 에 `org_id` 가 없다. (b) 동기화는 중단되지 않고 사용자 질문(`yes-or-no-p` 등, stub 으로 계수)이 0회이며, 그 파일은 ID 확보 실패로 보고되고, ID 없던 heading 은 `org_id` 없이, 디스크에 `:ID:` 가 있던 heading 은 그 `org_id` 와 함께 요청에 들어 있으며 파일 내용은 그대로다. (c) 그 파일의 entry 는 `org_id` 를 가진다. 이 단계가 새로 연 버퍼는 남아 있지 않고, (a) 의 버퍼는 여전히 열려 있다.

#### AC-ASRC-004 — 동기화 시 중복 ID (REQ 004)
Given 같은 `:ID:` 를 가진 heading 두 개(같은 파일 한 경우, 다른 파일 한 경우)가 있고
When 동기화가 실행되면
Then 두 entry 모두 요청에 들어 있고 `org_id` 가 없으며 그 밖의 필드는 기준선과 바이트 단위로 같고, 결과는 `failed` 가 아니다. 보고문에 그 ID 와 두 경로가 있으며, 디스크의 두 `:ID:` 는 바뀌지 않는다.

#### AC-ASRC-005 — 위치 등록과 저장 시점 (REQ 005)
Given 임시 `org-id-locations-file` 이 지정되어 있고
When `org_id` 를 실은 동기화가 끝나면
Then 그 파일에 이번에 만든 ID 와 디스크에 이미 있던 ID 가 모두 기록되어 있고, 기록된 경로를 `file-truename` 으로 펼친 값이 원본 파일의 `file-truename` 과 같으며, 저장은 동기화당 한 번, 모든 등록 뒤에 일어난다.

#### AC-ASRC-006 — 뒷면 링크 삽입 (REQ 005.2, 006)
Given `org_id` 를 가진 imoogi-Basic, imoogi-Cloze, stock `Basic` entry, `org_id` 없는 entry, 문자 집합 밖 `org_id`(예: `a b`) entry 가 있고
When planner 가 실행되면
Then Basic 형태의 `Back` 과 Cloze 형태의 `Back Extra` 끝에 DD-2 앵커가 정확히 붙고(`href` 의 ID 는 퍼센트 인코딩, 파일 경로 없음), `Front`/`Text` 는 그대로이며, `org_id` 없는 entry 의 필드는 기준선과 같고, 문자 집합 밖 entry 는 링크 없이 추가·갱신되며 그 key 의 `org_id_invalid` 진단이 있다. 프런트엔드 메시지 표에 `org_id_invalid` 항목이 있다.

#### AC-ASRC-007 — 해시 일관성과 1회 갱신 (REQ 007, 008)
Given 링크 이전 해시로 등록된 기존 노트가 있고
When 같은 entry 가 `org_id` 와 함께 두 번 동기화되면
Then 첫 실행은 결과가 같은 note ID 의 `updated` 이고 `updateNoteFields` 가 호출되며 `addNote`·`deleteNotes` 는 없고, 두 번째 실행은 변경 요청이 0건이고, fake `notesInfo` 재해시 값이 등록 해시와 같다.

#### AC-ASRC-008 — 프로토콜 (REQ 003.1)
Given `org_id` 가 있는 JSON 과 없는 JSON 이 있고
When decode/encode 하면
Then 있으면 값이 보존되고, 없으면 부재로 읽히며, 부재 시 encode 결과에 `org_id` 키가 없고, `Version` 은 2 이다.

#### AC-ASRC-009 — 캐시 적중 열기 (REQ 011.2, 011.5, 011.8)
Given (a) 동기화된 heading 의 ID 가 임시 캐시에 등록되어 있고 다른 대상에는 그 ID 가 없거나, (b) 캐시가 등록 대상 밖에 있는 파일을 가리키고 그 파일에 heading-local 일치가 정확히 1개이며 대상에는 그 ID 가 없고
When 수신 함수가 그 ID 로 호출되면
Then 두 경우 모두 현재 버퍼가 그 파일이고 point 가 heading-local `:ID:` 가 그 ID 인 heading 에 있으며, 결과는 `"ok"` 이고, 프레임 활성화 함수(stub)가 1회 호출된다.

#### AC-ASRC-010 — 파일 이동 후 재스캔 (REQ 011.3, 011.5)
Given 캐시가 옛 경로를 가리키고 파일은 등록 대상 안의 다른 경로로 옮겨졌고
When 수신 함수가 호출되면
Then 새 경로의 heading 이 열리고, 임시 캐시 파일의 경로가 `file-truename` 동치로 새 경로다.

#### AC-ASRC-011 — 파일 삭제 / ID 없음 (REQ 011.4, 011.8)
Given 캐시의 파일이 삭제되었거나, 어느 대상에도 그 ID 가 없고
When 수신 함수가 호출되면
Then ID 와 `imoogi-sync` 재동기화 안내가 담긴 메시지가 나오고, 존재하지 않는 파일을 방문하는 버퍼도 디스크의 새 파일도 생기지 않으며, 프레임 활성화 함수(stub)가 1회 호출된다.

#### AC-ASRC-012 — 열기 시 중복 (REQ 011.2, 011.6, 011.8)
Given 같은 ID 를 가진 heading 두 개가 서로 다른 대상 파일에 있고, (a) 임시 캐시에 그 ID 항목이 없거나 (b) 임시 캐시가 그중 한 파일을 가리키고
When 수신 함수가 호출되면
Then 두 경우 모두 현재 버퍼와 point 는 바뀌지 않고, 두 경로가 모두 담긴 후보 목록이 보이며, 프레임 활성화 함수(stub)가 1회 호출된다.

#### AC-ASRC-013 — 열기 시 사용자 편집 보호 (REQ 011.2, 012)
Given 대상 파일을 방문 중인 버퍼에 저장되지 않은 편집이 있고
When 수신 함수가 그 heading 의 ID 로 호출되면
Then 그 버퍼에서 heading 으로 이동하고, 버퍼는 여전히 수정 상태이며 내용과 디스크 파일이 바뀌지 않는다.

#### AC-ASRC-014 — 문자 집합과 주입 거부 (REQ 009, 011.1)
Given URL `imoogi-anki://open?id=x%22)(shell-command%20%22touch%20pwn%22`, 공백 포함, 빈 ID, 129자 ID, `id` 두 개, 다른 scheme/host 가 있고
When `imoogi-anki open` 이 각 URL 로 (fake runner) 실행되면
Then 모두 0 이 아닌 코드로 끝나고 emacsclient 호출은 0건이며 알림 호출이 1건 있다. 그리고 uuid / `prefix:random` / `ts` / `…@host.example` 형식 ID(퍼센트 인코딩된 링크로 전달)는 받아들여져 eval 인자가 디코딩된 ID 의 `(imoogi-anki-open-id "<ID>")` 와 정확히 같다. Elisp 수신 함수도 문자 집합 밖 문자열을 거부하고 파일을 열지 않는다.

#### AC-ASRC-015 — Emacs 미실행, 호출 실패, 시간 제한 (REQ 010)
Given 짧게 주입된 상한·호출별 제한과 fake runner 가 있고
When 처리기가 유효한 URL 로 실행되면
Then (a) 탐침이 계속 실패하면 Emacs 실행 1회, 상한까지 탐침, 알림 1회, 0 이 아닌 코드 / (b) 실행 뒤 두 번째 탐침이 성공하면 eval 1회, 코드 0 / (c) emacsclient 탐색 자체가 실패하면 emacsclient 호출 0건, 실행 0건, 알림 1회, 0 이 아닌 코드 / (d) eval 이 0 이 아닌 코드로 끝나거나 `*ERROR*`(예: void-function)를 내거나 세 정상 값 밖의 결과(예: `"weird"`)를 내면 그 오류 문자열을 담은 알림 1회, 0 이 아닌 코드 / (e) 탐침 또는 eval 이 호출별 제한을 넘기면 그 호출이 종료되고 알림 1회, 0 이 아닌 코드 / (f) eval 결과가 `"not-found"` 또는 `"duplicate"` 이면 알림 0회, 코드 0 (`"ok"` 는 (b)).

#### AC-ASRC-016 — `.app` 생성을 등록 없이 검증 (REQ 013)
Given 공백이 들어간 임시 디렉터리와, 공백이 들어간 바이너리 경로가 있고
When 설치 명령이 `--dest <임시> --no-register` (fake runner)로 실행되거나 생성 함수가 직접 호출되면
Then 생성된 `Info.plist` 에 `CFBundleURLTypes` → `CFBundleURLSchemes` 의 `imoogi-anki` 가 있고, AppleScript 원본에 `on open location` 이 있으며, 바이너리 절대 경로와 URL 이 각각 `quoted form of` 로 인용되어 있고(PATH 조회 없음), `lsregister` 호출은 0건이다.

## § D.1 Definition of Done

- AC 001–016 자동 테스트 통과(출력 원문을 `progress.md` §E.2 에 기록)
- 기준선 비회귀: 위 두 실행 명령이 모두 통과하고, ERT anki 실행 수는 33 이상이며 예상 밖 결과는 0
- 운영자 확인(자동화 불가): macOS 에 처리기를 설치하고 노트를 동기화한 뒤, Anki 카드 뒷면의 "Emacs에서 원본 열기" 를 Emacs 실행 중 / 미실행 두 경우로 누르면 Emacs 가 앞으로 나오며 그 heading 으로 이동한다. Anki 가 사용자 정의 scheme 링크를 OS 로 넘기는지(R-4)도 이 확인으로만 검증되므로, 결과를 `progress.md` 에 운영자 확인으로 기록한다.
- README 갱신(M5)
