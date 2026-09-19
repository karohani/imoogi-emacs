# 모듈 서브패키지 리팩토링 시나리오

## 범위와 보존 계약

일반 설정은 general, 프로젝트/작업공간/프로젝트 노트는 project,
Org·Markdown·미리보기·첨부·학습카드는 org, 개발 도구는 development,
언어별 LSP 설정은 development/lang으로 분류한다.
기존 명령명, feature, 단축키, 모듈 실패 식별자와 로딩 순서를 보존한다.
개인 데이터와 레지스트리의 위치·형식은 바꾸지 않는다. 외부 패키지와 Go CLI는
이번 구조 변경 대상이 아니다. 기존 미커밋 변경을 포함한 현재 파일이 기준이다.

## 변경 전 작성한 시나리오와 테스트

| ID | Given / When / Then | 검증 |
|---|---|---|
| S1 | 오프라인 초기 설정으로 부팅하면 모든 모듈이 기존 순서로 로드되고 누락 모듈이 없다 | boot assertion + module-layout-test |
| S2 | 수정 중인 Org 버퍼와 버퍼 로컬 상태가 있을 때 reload하면 내용·수정 플래그·파일 연결을 보존한다 | module-layout-test 신규 |
| S3 | 기존 사용자 저장 경로와 사용자 정의 언어 설정 경로가 있을 때 reload하면 그대로 유지한다 | module-layout-test 신규 + config/project-notes/targets 테스트 |
| S4 | 일반 단축키·포맷터·Git 표시·LSP를 쓰면 기존 명령 및 훅이 연결된다 | intellij-keybindings, lsp-modules, module-layout-test 신규 |
| S5 | Anki/flashcards/GPTel 메뉴를 별도로 바이트 컴파일하고 새 프로세스에서 로드하면 메뉴가 등록된다 | compiled-menu-test 경로 갱신, 기존 단언 유지 |
| S6 | flashcards 의존성이 없으면 해당 모듈만 실패하고 나머지는 정상이다 | flashcards-boot-test, boot-health-test |
| S7 | project/study/외장 노트 및 클립보드의 생성·저장·이동을 하면 기존 데이터 계약을 따른다 | project-notes-test, clipboard-test, Anki 전체 테스트 유지 |
| S8 | 하위 패키지에 문법 오류가 있으면 검사 단계에서 감지한다 | run.sh의 재귀 check-parens |
| S9 | reload 후 언어 훅과 메뉴에 중복 등록이 생기지 않는다 | 기존 transient reload + 신규 hook count 검사 |

## 순서

1. 현재 테스트 결과와 소스 해시를 기록한다.
2. 상태·훅 보존 테스트를 기존 구조에서 실행한다.
3. 계획을 독립 리뷰하고 파일 이동, 참조 경로 및 검사 경로를 갱신한다.
4. 포맷터 구현과 Git 표시의 설정 소유권을 모아 중복 책임을 줄인다.
5. 전체 테스트, 오프라인 부팅, 바이트 컴파일 메뉴 검증 및 diff 검사를 실행한다.

## 검증 한계

기존 GUI calendar 테스트 2개는 batch 환경에서 건너뛴다.
실제 외장 장치 탈착, OS 클립보드, 실제 Anki 서버는 이번 리팩토링 테스트가
직접 조작하지 않는다. 관련 로직은 기존 격리 테스트로 검증한다.

## 변경 전 기준 결과 및 계획 리뷰

- 2026-09-19: `make test` 성공. 신규 보존 테스트 2개를 포함하여 ERT 389개 중
  387개 통과, 기존 GUI 2개 skip. Go·셸·vendor 검증 통과.
- 독립 critic 리뷰: OKAY. 이전 LSP 기본 경로만 이관하고 사용자 경로는 유지하는
  테스트, 재귀 문법 검사, compiled-menu/boot-health/clean-elc 경로 갱신을 필수로 지정.
- 초기 sandbox 실행의 로컬 HTTP 테스트 4개 실패는 제한 해제 후 모두 통과했다.

## 실제 이동 목록

| 패키지 | 기존 번호 모듈 |
|---|---|
| `general/` | `00-defaults`, `01-keys`, `02-completion`, `03-which-key`, `05-transient`, `09-autorevert`, `10-theme`, `11-editing`, `12-navigation`, `13-system`, `21-native-compile` |
| `project/` | `04-projects`, `06-git`, `07-treemacs`, `22-tabs`, `26-project-notes` |
| `org/` | `08-obsidian`, `14-org`, `15-markdown`, `23-org-preview`, `24-anki`, `25-flashcards`, `28-clipboard` |
| `development/` | `16-elisp`, `17-lsp`, `18-languages`, `19-folding`, `20-terminal`, `27-gptel` |

`lsp/*.el` 9개는 `development/lang/`, `anki/*.el` 10개는 `org/anki/`,
`flashcards/*.el` 4개는 `org/flashcards/`로 이동했다.
기존 모듈 소스 52개와 테스트 소스 49개가 모두 남아 있음을 변경 전 해시 목록과
경로 대응표로 확인했다. 기존 테스트는 경로 참조만 갱신했고 단언을 삭제하지 않았다.

추가 보존 검증: 이전 세션의 Anki/flashcards 기본 라이브러리 경로와 load-path를
새 위치로 바꾸되 사용자 경로는 그대로 유지한다. 디스크의 사용자 상태는 변환하지 않는다.

## 최종 검증 (2026-09-19)

- `make test`: 성공. ERT 392개 중 390개 통과, 실패 0개, 기존 GUI 2개 skip.
- 신규 회귀 테스트 5개: 수정 중인 문서·사용자 설정, 개발 단축키·훅 중복,
  기존 모듈 로딩 순서, LSP 기본 경로 이관, Anki/flashcards 경로 이관·사용자 override 보존.
- 재귀 check-parens, 오프라인 boot, 깨끗한 subprocess의 바이트 컴파일 메뉴 테스트 통과.
- Go 전체 테스트, 셸 테스트, vendor provenance 검증 통과.
- `make lint` (go vet), `git diff --check` 통과.
- fmt-check 종료 코드는 0이지만 기존 vendor/go-mode 테스트 fixture 두 개에서
  Go package 선언이 없다는 진단이 발생한다. 이번 변경은 Go 소스를 수정하지 않는다.
- 개인 데이터·사용자 레지스트리 이동 없음. 커밋·푸시는 수행하지 않았다.
