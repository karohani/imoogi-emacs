# Org → Anki 카드 유형 확장 계획 — 최종 (logseq-anki-sync 참고)

작성일: 2026-09-20 · 상태: 계획 확정, 백로그 등록 완료 (코드 변경 없음) · 대상: `internal/anki/**` + `modules/org/anki/**`

## 1. 조사 결과

- 요청한 문서 URL(`…/docs/card-types`)은 404였다. GitHub 프로젝트 위키(클론)와 `main` 브랜치의 `src/notes/*.ts`를 1차 출처로 사용했다.
- **핵심 관찰**: logseq-anki-sync는 카드 유형이 여러 개이지만 Anki에는 **Cloze 노트 타입 하나**만 만든다. Multiline 카드는 자식 블록을 `{{c1::…}}`로, 역방향이면 부모를 `{{c2::…}}`로 감싸고, Swift 카드도 화살표 양쪽을 같은 방식으로 감싼다. 유형별 차이는 **Text 필드를 어떻게 조립하느냐**뿐이다.
- 현재 imoogi-anki 구조 (관찰):
  - `ANKI_NOTE_TYPE` 속성이 있는 헤딩 1개 = Anki 노트 1개 (`ANKI_NOTE_ID` 1개, registry 1건).
  - 노트 타입은 `imoogi-Basic`(Front/Back)과 `imoogi-Cloze`(Text/Back Extra) 둘뿐. Cloze는 `{{cN::…}}`가 없으면 `cloze_marker_missing`.
  - 본문은 헤딩 아래부터 **다음 헤딩 직전까지**만 읽는다 (`imoogi-scan.el`). 하위 헤딩은 포함되지 않는다.
  - `ANKI_DECK`/`ANKI_TAGS`는 nearest-wins 상속 체인(`imoogi-props.el`), `ANKI_NOTE_TYPE`은 자기 drawer만.
  - `orgdoc.Render` 호출부는 `planner.go`와 `migrate.go` 2곳.
- go-org v1.9.1 렌더 프로브 (실측): `#+BEGIN_CLOZE …` → `<div class="cloze-block">`, 목록 → `<ul><li>`, `{{c1::bar}}` 그대로 통과, `:->` → `:-&gt;`로 이스케이프(화살표 감지는 렌더 **전** 텍스트 단계).
- `model.IsOwned`는 접두사만 검사하지만 설치·planner(`planner.go:411`, `:625`)·registry는 선언된 타입명이 실제 Anki 모델명이라고 가정한다 → 가상 노트 타입명 방식은 부적합.

## 2. 확정된 설계

### 2.1 원칙

1. **새 Anki 노트 타입 없음.** 모든 카드 유형은 기존 `imoogi-Cloze`로 만든다.
2. **Cloze 표기는 편집기 명령이 만든다.** Go가 `{{cloze X}}` 같은 대체 문법을 해석하지 않는다. 사용자는 영역을 잡고 `imoogi-anki-cloze`를 부르면 `{{cN::…}}`가 된다.
3. **노트 타입은 편집 시점에 drawer에 기록된다.** cloze 명령이나 방향 지정 명령이 `ANKI_NOTE_TYPE`이 없으면 `imoogi-Cloze`를 써 준다. 동기화 때 Go는 타입을 바꾸지 않고 **검증만** 한다.
4. **카드 옵션 속성 3개**: `ANKI_DIRECTION`(`->`|`<-`|`<->`), `ANKI_INCREMENTAL`(`t`), `ANKI_SWIFT`(`t`). `ANKI_DIRECTION` 또는 `ANKI_INCREMENTAL`이 있으면 Multiline(Q/A) 카드, `ANKI_SWIFT`가 있으면 Swift 카드다. 별도 `ANKI_CARD_KIND`는 두지 않는다.
5. 세 속성 모두 `ANKI_DECK`처럼 nearest-wins 상속(`#+PROPERTY:`로 파일 전체 지정 가능). 카드 여부는 여전히 자기 drawer의 `ANKI_NOTE_TYPE`이 결정한다.

### 2.2 카드 유형 매핑

| logseq 유형 | logseq 문법 | imoogi 방식 | 어디서 | 카드 |
|---|---|---|---|---|
| Cloze 매크로/블록 | `{{cloze X}}`, `{{c1 X}}`, `#+BEGIN_CLOZE` | 영역 선택 → `imoogi-anki-cloze` → `{{cN::X}}` (번호 자동, `C-u N`으로 지정, 안쪽 `}}`→`} }`, 끝이 `}`면 닫기 앞에 공백) | Elisp | t11 |
| Extra | `extra::`, `#+BEGIN_EXTRA` | `#+BEGIN_EXTRA … #+END_EXTRA` → 기존 `Back Extra` 필드 | Go orgdoc | t12 |
| (토대) | — | 속성 3개 → 프로토콜 v2 → Go 검증(`card_option_needs_cloze`) | Elisp + Go | t13 |
| Multiline 카드 | `#card` + 자식, `#forward/#reversed/#bidirectional`, `#incremental` | 헤딩 = 질문, 본문 최상위 목록 = 답. `ANKI_DIRECTION`, `ANKI_INCREMENTAL` | Go orgdoc + Elisp 명령 | t14 |
| Swift 카드 | `A :-> B`, `:<-`, `:<->` | `ANKI_SWIFT: t` 헤딩의 본문 각 줄. 헤딩 1개 = 노트 1개, 줄마다 카드 | Go orgdoc | t15 |
| replacecloze | `replacecloze:: " 'a', /re/ "` | 보류 (정규식 속성, 수요 낮음) | — | — |
| Image Occlusion | 편집기 UI + occlusion 데이터 | 제외 (그래픽 편집기 필요) | — | — |
| deck | `deck::` | 기존 `ANKI_DECK` 그대로 | — | 기존 |

Multiline이 `imoogi-Basic`과 다른 점: 단순 `->` 비-incremental Q/A는 지금도 `imoogi-Basic`으로 가능하다. Multiline은 `<-`, `<->`(역방향·양방향)와 `incremental`(답 항목마다 별도 카드)이 필요할 때만 쓴다.

Multiline 생성 규칙 (logseq와 동일): `->`이면 답 항목을 `{{c1::…}}`, `<-`이면 제목을 `{{c2::…}}`, `<->`이면 둘 다. `incremental`이면 항목마다 c1, c2, … 별도 번호.

Swift 번호 규칙: i번째 화살표 줄은 오른쪽을 c(2i−1), 왼쪽을 c(2i)로 매긴다. `:->`는 오른쪽만, `:<-`는 왼쪽만, `:<->`는 둘 다(카드 2장).

### 2.3 결정 기록

| # | 결정 | 확정안 | 배제한 대안 |
|---|---|---|---|
| A | 카드 옵션 전달 | Org 속성 3개 → `protocol.Entry` 필드 `direction`/`incremental`/`swift` → `Version` 1→2 | 가상 노트 타입명: 설치·registry·planner가 실제 모델명을 가정 |
| B | "답"의 정의 | 본문 최상위 목록 항목 | 하위 헤딩: 스캔 경계 변경, 자체 엔트리와 충돌 |
| C | Swift 카드 개수 | 헤딩당 노트 1개, 줄마다 cloze 번호 | 줄마다 노트: 헤딩↔`ANKI_NOTE_ID` 1:1 불변식 파괴 |
| D | 속성 이름 | `ANKI_DIRECTION`, `ANKI_INCREMENTAL`, `ANKI_SWIFT` (종류는 속성 존재로 결정) | `ANKI_CARD_KIND` 별도 속성: 기억할 이름만 늘어남 |
| E | 상속 | 세 속성 모두 nearest-wins 상속 | 자기 drawer만: 파일 전체가 Q/A일 때 반복 기입 |
| F | 답 없는 Multiline | `multiline_answer_missing`으로 skip | 빈 `{{c1::}}` 생성: 빈 카드가 Anki에 남음 |
| G | Cloze 문법 | Go 대체 문법 없음, Elisp 영역 명령 | `{{cloze X}}`/`#+BEGIN_CLOZE` Go 해석: logseq 흉내일 뿐 편집기에서는 명령이 자연스러움 |
| H | 노트 타입 자동 지정 | 편집 명령이 drawer에 `imoogi-Cloze` 기록, 동기화는 검증만 | 동기화 때 암묵 변경: Org와 registry의 타입이 어긋남, 상속된 방향값이 파일 전체를 카드로 만듦 |

## 3. 작업 분할 (백로그 카드)

| 카드 | 내용 | 변경 범위 | 선행 |
|---|---|---|---|
| **t11** | `imoogi-anki-cloze` 영역 명령: `{{cN::…}}` 감싸기, 번호 자동/`C-u N`, 안쪽 `}}`→`} }`·끝이 `}`면 ` }}`로 닫기, `ANKI_NOTE_TYPE` 없으면 `imoogi-Cloze` 기록, transient 바인딩. `imoogi-anki-uncloze` 선택 | Elisp만 | 없음 |
| **t12** | `#+BEGIN_EXTRA` → `Back Extra` 분리. 기존 카드 렌더 바이트 불변 회귀 테스트 | `internal/anki/orgdoc` | SPEC-ANKICARD-001 종료 |
| **t13** | 속성 3개 상속 읽기 → `protocol.Entry` 필드 → `Version` 2 → `planner.go`/`migrate.go` Render 옵션 전달 → 검증 진단 `card_option_needs_cloze` → 계약 테스트 ~10개 갱신. 렌더 변경 없음 | Elisp + Go 프로토콜/planner | SPEC-ANKICARD-001 종료 |
| **t14** | Multiline 렌더(`orgdoc/multiline.go`), `multiline_answer_missing`, `.children-list` CSS, 방향/incremental 지정 Elisp 명령, README | Go orgdoc + Elisp | t13 |
| **t15** | Swift 렌더(`orgdoc/swift.go`), `swift_arrow_missing`, README | Go orgdoc + Elisp 문구 | t13 |

t11과 t12는 서로 독립이고 프로토콜을 건드리지 않아 단독 배포 가능. t13이 t14·t15의 토대.

## 4. 변경 파일 (예상)

| 영역 | 파일 | 카드 |
|---|---|---|
| Elisp 명령 | `modules/org/anki/imoogi.el`(transient), 신규 `imoogi-cloze.el` | t11, t14 |
| Elisp 속성/전송 | `modules/org/anki/{imoogi-props,imoogi-scan,imoogi-process,imoogi-error}.el` | t13, t14, t15 |
| Go 렌더 | `internal/anki/orgdoc/{extra,multiline,swift}.go` + 테스트 | t12, t14, t15 |
| Go 프로토콜/planner | `internal/anki/protocol/protocol.go`, `planner/planner.go`, `planner/migrate.go` | t13 |
| 카드 스타일 | `internal/anki/model/assets/base.css` | t14 |
| 테스트 | `tests/anki-{process,sync-oneway,sync-error,props,scan}-test.el`, `cmd/imoogi-anki/*_test.go`, `internal/anki/protocol/*_test.go` | t13 |
| 문서 | `README.md` Anki 절 | t11–t15 |

## 5. 위험

| 위험 | 정도 | 대응 |
|---|---|---|
| SPEC-ANKICARD-001이 `in-progress`(sync-audit FAIL 재감사 중) | 높음 | t12~t15는 001 종료 후 착수. t11은 Elisp만이라 독립 |
| 프로토콜 v2로 Emacs↔바이너리 버전 불일치 | 중간 | 기존 `binary_incompatible` 진단이 잡음; `make build-anki` 재설치 안내 |
| 기존 Cloze 카드 렌더 결과 변화 → 불필요한 update | 중간 | 바이트 불변 회귀 테스트 (t12, t13) |
| cloze 안 `}}`(LaTeX) | 낮음 | t11 명령이 안쪽 `}}`는 `} }`로, 선택이 `}`로 끝나면 닫는 `}}` 앞에 공백을 넣음 (Anki cloze 정규식은 첫 `}}`에서 닫힘) |
| 여러 줄 HTML이 든 cloze의 Anki 표시 | 중간 | 실제 Anki 렌더 확인 (AnkiConnect 연결 필요) |
| 워킹트리 미커밋 변경(`CLAUDE.md`, `04-projects.el`, `workspace-bridge-test.el`) | 낮음 | 별도 worktree에서 작업 |

## 6. 다음 단계 (계획 시점)

1. 백로그에서 카드를 골라 `/moai plan`으로 SPEC 작성 (t11부터 가능).
2. t12~t15는 SPEC-ANKICARD-001 종료 확인 후.

## 7. 실행 기록 (2026-09-20)

### t11 — 계획 정정: 대부분 이미 구현돼 있었다

착수 시점에 `modules/org/24-anki.el`을 읽어 보니 `imoogi-anki-cloze-region`이 **이미 존재**했다. 계획서는 이를 신규 작업으로 적었는데, 이는 사전 확인 없이 세운 가정이었다. 실제 상태:

| t11 항목 | 착수 전 상태 |
|---|---|
| 영역을 `{{cN::…}}`로 감싸기 | 구현됨 (`24-anki.el:201`) |
| 번호 자동 = subtree 최대 + 1 | 구현됨 (`imoogi-anki--next-cloze-number`) |
| `C-u N`으로 번호 지정 | 구현됨 (prefix arg) |
| `ANKI_NOTE_TYPE` 없으면 `imoogi-Cloze` 기록, 다른 값이면 알림만 | 구현됨 (스톡 `Cloze`도 인정) |
| `C-c a z` + transient 바인딩 | 구현됨 |
| 테스트 | 3개 존재 (`tests/anki-commands-test.el`) |
| **`}}` 이스케이프** | **없음 → 이번에 추가** |
| **영역 없을 때 커서 낱말** | **없음(`user-error`였음) → 이번에 추가** |
| `imoogi-anki-uncloze` | 없음 (카드에서 '선택' 항목, 구현 안 함) |

실제 변경 (재현-우선, RED → GREEN):
- `imoogi-anki--cloze-safe-text` 신규: 연속한 `}` 를 한 칸씩 띄운다(`}}`→`} }`, `}}}}`→`} } } }`). Anki cloze 정규식이 비탐욕이라 첫 `}}`에서 닫히는 것을 막는다.
- 영역이 `}`로 끝나면 닫는 `}}` 앞에 공백 하나(`pad`). `\sqrt{a^{2}}` → `{{c1::\sqrt{a^{2} } }}`.
- `interactive` 사양이 경계를 직접 고른다: 영역이 있으면 영역, 없으면 `bounds-of-thing-at-point 'word`, 낱말도 없으면 안내 후 중단.
- 테스트 4개 추가 (`separates-inner-double-brace`, `separates-trailing-brace-from-marker`, `leaves-brace-free-text-untouched`, `falls-back-to-word-at-point`).

검증 (실측):
- RED: `Ran 406 tests, 402 results as expected, 2 unexpected` — 새 테스트 2개만 실패, 회귀 방지 테스트는 통과. 로그 `.moai/state/verify/297a7e36/t11/red.log`
- GREEN: `Ran 407 tests, 405 results as expected, 0 unexpected, 2 skipped`, `make test-elisp` exit 0. 로그 `.moai/state/verify/297a7e36/t11/green2.log`

미검증: Anki 실기기에서의 렌더는 확인하지 않았다(AnkiConnect 미연결). "Anki cloze 정규식이 첫 `}}`에서 닫힌다"는 Anki의 문서화된 동작을 근거로 한 것이며, 실제 컬렉션에서 대조하지 않았다.

관찰 (수정하지 않음): 영역 안에 `::` 가 있으면 Anki는 그 뒤를 힌트로 해석한다(`{{c1::답::힌트}}`). 카드 t11의 범위 밖이라 손대지 않았다.

### t12 — 구현 완료

죽어 있던 `Back Extra` 필드를 살렸다. `Render` 시그니처는 바꾸지 않았으므로 프로토콜·planner·registry는 그대로다.

- `internal/anki/orgdoc/extra.go` 신규: `splitExtraBlocks`가 본문에서 `#+BEGIN_EXTRA … #+END_EXTRA`를 떼어낸다. 대소문자 무시(Org 키워드 규칙), 줄 시작 고정(문장 속 단어가 블록을 열지 않게), 여러 블록은 빈 줄로 이어 붙임.
- **블록이 없는 본문은 바이트 그대로 반환**한다(트림조차 하지 않음). planner가 렌더 결과를 해시하므로, 여기서 정규화하면 기존 카드 전부가 내용 변화 없이 `updated`로 뜬다.
- 마커 검사는 EXTRA를 뺀 나머지에만 건다. Anki의 cloze 템플릿은 `{{cloze:Text}}`만 읽으므로, 마커가 EXTRA 안에만 있으면 빈칸 없는 노트가 되어 Anki가 거부한다. 그 경우를 `cloze_marker_missing`으로 먼저 잡는다.
- **`Back Extra`는 비어 있어도 항상 내보낸다.** 없을 때 키를 빼면 더 깔끔해 보이지만, AnkiConnect는 주지 않은 필드를 그대로 둔다 — EXTRA를 지운 카드의 update 요청에 그 필드가 없으면 예전 내용이 컬렉션에 영원히 남는다.
- `imoogi-Basic`은 건드리지 않는다. Basic에는 `Back Extra` 필드가 없어 키를 내보내면 `note_field_missing`이 난다. EXTRA는 지금처럼 Back 안에 `<div class="extra-block">`로 남는다.

테스트 7개 추가 (`internal/anki/orgdoc/extra_test.go`): 바이트 불변 골든, 빈 필드 방출, 본문→Back Extra 이동, 소문자 키워드, 다중 블록, EXTRA 안 마커 진단, Basic 인라인 유지.

검증 (실측):
- RED: 새 테스트 5개 실패, 골든 2개(Cloze Text 바이트 불변·Basic 인라인) 통과
- GREEN: `go test ./... -count=1` exit 0 · orgdoc **커버리지 100.0%** 유지 · `make lint` exit 0 · `make fmt-check` exit 0 · `go build ./...` 0 · `GOOS=windows go build ./cmd/imoogi-anki` 0. 로그 `.moai/state/verify/297a7e36/t12/`

알려진 일회성 영향: `hashing.Hash`는 필드 맵의 모든 키를 해시에 넣는다(`hashing.go:47-57` 확인). `Back Extra`가 새로 들어가므로 **기존 Cloze 노트 전부가 다음 동기화에서 `updated` 한 번씩** 뜬다. 내용은 그대로이고 복습 이력도 보존되지만, 첫 동기화 결과 개수가 평소보다 많게 보인다.

미검증: Anki 실기기에서 Back Extra가 카드 뒷면에 나오는지는 확인하지 않았다(AnkiConnect 미연결).

### t12 — 경계에서 잡은 회귀 하나

`Back Extra`를 내보내기 시작하자 **스톡 `Cloze` 노트 타입이 깨졌다.** `imoogi-Cloze`의 필드는 `Text`/`Back Extra`지만 스톡 `Cloze`는 `Text`/`Extra`이고(`fake_client_test.go:183`), `resolveFields`는 이름을 정확히 맞춰 찾으므로 손으로 쓴 스톡 Cloze heading이 `note_field_missing`으로 실패했다. 이 프로젝트는 스톡 타입 동기화를 명시적으로 지원하므로(AC-C-022a) 회귀다.

Go 전체 테스트는 통과하고 있었다 — 스톡 Cloze를 add 경로로 통과시키는 테스트가 없었기 때문이다. 재현 테스트를 먼저 써서 실패를 확인한 뒤 고쳤다.

- `planner.go`에 `clozeExtraAliases` 추가: `back extra` ↔ `extra`. 정확한 이름 매칭이 **실패한 뒤에만** 참조하고, 두 이름 다 없는 모델은 여전히 `note_field_missing`으로 실패한다(두 번째 테스트가 이를 고정).
- `fake_client_test.go`의 "the renderer emits neither" 주석이 내 변경으로 거짓이 되어 함께 고쳤다.
- 테스트 2개 추가 (`internal/anki/planner/extra_alias_test.go`).
- 재검증: `go test ./... -count=1` exit 0 · orgdoc 100.0% · planner 91.6% · lint 0 · fmt 0. 로그 `.moai/state/verify/297a7e36/t12/gotest2.log`

### 관찰 — 고아 삭제 경로 (가설, 미검증)

`planner.go:624`의 고아 확인 로직은 AnkiConnect `notesInfo`가 돌려준 **모든** 필드로 해시를 다시 계산해 registry 값과 비교한다. 실제 Anki가 notesInfo에서 빈 필드까지 포함해 돌려준다면, t12 이전에는 registry 해시가 `Text` 하나로 계산돼 있었으므로 Cloze 노트의 고아 삭제는 **항상 `delete_candidate_unowned`로 보류**되어 왔을 것이고, t12 이후 첫 동기화가 registry를 갱신하면 비로소 맞아떨어진다.

이는 **가설이다.** 테스트의 fake client는 notesInfo가 seed한 필드만 돌려주도록 모델링돼 있어(`fake_client_test.go:274-278`) 테스트 스위트로는 판별할 수 없고, 실제 컬렉션으로 확인하지 않았다. 안전한 방향(삭제 안 함)으로 치우쳐 있어 급하지 않다.

### 카드별 변경 파일

| 카드 | 파일 |
|---|---|
| t11 | `modules/org/24-anki.el`, `tests/anki-commands-test.el` |
| t12 | `internal/anki/orgdoc/orgdoc.go`, `orgdoc/extra.go`(신규), `orgdoc/extra_test.go`(신규), `planner/planner.go`, `planner/extra_alias_test.go`(신규), `planner/fake_client_test.go`(주석) |

두 카드의 파일 집합이 겹치지 않으므로 pathspec으로 따로 스테이징할 수 있다. 아직 커밋하지 않았다.

### 순서에 대한 기록

사용자가 `SPEC-ANKICARD-001`이 아직 `in-progress`인 상태를 알고도 t11~t15를 순서대로 진행하도록 지시했다. 따라서 001의 sync-audit는 나중에 움직인 HEAD를 대상으로 돌게 된다.

### t13~t15 — 사전 확인 결과 (미착수)

같은 실수를 반복하지 않도록 나머지 카드도 착수 전에 확인했다.

- **t12**: 미구현 확인. `Back Extra` 필드는 노트 타입에 선언돼 있고(`internal/anki/model/model.go:99`) 카드 뒷면 템플릿도 그 필드를 렌더하지만(`assets/cloze-back.html:4`), `orgdoc.Render`는 Cloze에 `Text`만 채운다 — 즉 이 필드는 **현재 영구히 비어 있다**. t12는 신규 작업이자 이 죽은 필드를 살리는 작업이다.
- **t13 / t14 / t15**: `ANKI_DIRECTION`·`ANKI_INCREMENTAL`·`ANKI_SWIFT`·multiline·화살표 관련 코드가 Go·Elisp 어디에도 없음. 전부 신규 작업.

### t13 — 구현 완료 (커밋 6f3ae6c)

SPEC-ANKICARD-002로 진행. plan-audit PASS 0.91, 막는 지적 4건 수정 후 착수 승인.

Org 속성 `ANKI_DIRECTION`·`ANKI_INCREMENTAL`·`ANKI_SWIFT`가 `ANKI_DECK`과 같은 nearest-wins 체인으로 해석돼 `protocol.Entry`로 전달되고, 프로토콜이 v1→v2로 올라갔다. 렌더 앞에 검증 게이트가 서서 `card_option_invalid`·`card_option_conflict`·`card_option_needs_cloze` 중 하나를 항목당 하나만 내보낸다. 카드는 진단 코드 1개를 요구했으나 3개로 늘렸다 — 세 실패의 해결 방법이 각각 다르고, `card_option_needs_cloze`라는 이름이 값 오류에 붙으면 이름이 거짓이 되기 때문(작성자와 감사자가 독립적으로 같은 결론).

렌더는 하나도 바뀌지 않는다. 변경 전 기록한 골든 2개로 기계적으로 증명했고, `orgdoc`·`hashing`·바이너리 진입점의 제품 코드 diff는 0이다.

**경계에서 잡은 회귀**: `Back Extra`를 내보내기 시작하자 스톡 `Cloze`(필드가 `Text`/`Extra`)가 `note_field_missing`으로 깨졌다. Go 전체 테스트는 그 경로를 지나는 테스트가 없어 통과하고 있었다. 재현 테스트를 먼저 쓴 뒤 `clozeExtraAliases`(`back extra` ↔ `extra`)로 해결 — 정확한 이름 매칭이 실패한 뒤에만 참조하고, 두 이름 다 없는 모델은 여전히 실패한다.

검증: `go test ./... -count=1` 0 · planner 91.5→91.6% · lint 0 · fmt 0 · host·windows 빌드 0 · elisp 0.

### t14 — 구현 완료 (커밋 08f7504)

SPEC-ANKICARD-003으로 진행. plan-audit 3회(FAIL 0.63 → FAIL 0.706 → **PASS 0.923**), 착수 승인 후 구현.

헤딩 제목이 질문, 본문 최상위 목록 항목이 답. 방향에 따라 답 또는 제목을 감싸고, incremental이면 항목마다 별도 번호.

**렌더 전 Org 원문에서 감싼다.** 렌더 후 방식이 단순히 불편한 게 아니라 틀렸다 — 중첩 목록은 부모 `<li>` 안에 렌더돼 깊이 추적기가 필요하고, **설명 목록은 `<li>`를 아예 안 만들어서 HTML 스캐너가 조용히 놓친다.**

**감싸기 시작 위치는 파서의 바이트 연산을 그대로 따른다.** 토큰을 검색해서 정하면 안 된다(요구사항이 명시적으로 금지). go-org의 세 소비 중 둘은 위치를 보지 않고 0번 위치부터 고정 바이트를 잘라내서, `- Tokyo [ ] is big`은 지금도 앞 네 글자를 잃는다. **한국어에서는 더 나쁘다**: 4바이트 절단이 글자 중간에 떨어져 유효하지 않은 UTF-8이 나온다(`- 서울특별시 [ ] 큼` → `��특별시 [ ] 큼`). 이 저장소의 노트가 한국어라 다국어 예제가 수용 기준에 들어갔다.

이미 직접 쓴 표식이 있는 항목은 감싸지 않는다(Anki 표식은 중첩 불가). 답 목록이 없으면 `multiline_answer_missing`으로 건너뛴다.

검증: `go test ./... -count=1` 0 · orgdoc 100.0% 유지 · planner 92.5→92.8% · lint 0 · fmt 0 · `make ci-local` 0 · elisp 435개 실패 0 · 렌더 골든 `6f3ae6c` 대비 불변.

**알려진 미보고 동작 → 카드 t16으로 등록**: 중첩된 `#+BEGIN_EXTRA`(정상 Org)가 `splitExtraBlocks`의 비탐욕 매칭 탓에 떠돌이 `#+END_EXTRA`를 본문에 남기고, 그게 답 목록을 쪼개 뒤쪽 답을 떨어뜨린다. 표식이 카드에 글자로 보이므로 조용한 실패는 아니다. t12(`5d3868f`)에서 온 기존 동작이며 t14는 테스트로 고정만 했다.

> **정정(2026-09-28)**: "중첩된 `#+BEGIN_EXTRA`(정상 Org)"는 틀린 전제다. Org 9.7.11은 같은 이름의 특수 블록을 중첩하지 않는다(2026-09-28 측정). SPEC-ANKICARD-005가 짝 없는 표식을 `extra_block_unbalanced`로 보고하도록 해소했다.

### 카드별 커밋

| 카드 | 커밋 | 내용 |
|---|---|---|
| t11 | `d053c67` | cloze 표식 중괄호 이스케이프 + 커서 낱말 폴백 |
| t12 | `5d3868f` | `#+BEGIN_EXTRA` → Back Extra, 스톡 Cloze 필드 별칭 |
| t13 | `6f3ae6c` | 카드 옵션 속성 + 프로토콜 v2 + 검증 게이트 |
| t14 | `08f7504` | Multiline Q/A 카드 |
| t15 | 진행 중 | Swift 화살표 카드 (SPEC-ANKICARD-004) |
| t16 | 백로그 | 보조 블록 중첩/비대칭 처리 |

모두 `main`에 커밋, push는 하지 않음.

## 출처

- https://github.com/debanjandhar12/logseq-anki-sync/wiki (How-to-make-simple-cloze-cards, How-to-make-multiline-cards, How-to-make-incremental-multiline-cards, Making-Swift-cards, How-to-add-extra-details-to-card, How-to-make-cloze-cards-inside-math-and-code, How-to-set-or-change-the-deck-for-cards)
- https://github.com/debanjandhar12/logseq-anki-sync/tree/main/src/notes (ClozeNote.ts, MultilineCardNote.ts, SwiftArrowNote.ts, ImageOcclusionNote.ts)
- https://debanjandhar12.github.io/logseq-anki-sync/docs/card-types — 404 (2026-09-20)
- 로컬: `internal/anki/{orgdoc,protocol,planner,model,registry}`, `modules/org/anki/{imoogi-scan,imoogi-props,imoogi-process}.el`, `.moai/specs/SPEC-ANKICARD-001/progress.md`
