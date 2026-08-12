# imoogi-emacs 단축키 워크북

이 문서는 `README.md`와 `modules/*.el`에 정의된 단축키를 액션 중심으로 정리한 사용 워크북이다.

## 표기

| 표기 | 의미 |
|---|---|
| `C-` | Control |
| `M-` | Meta/Option 또는 Alt |
| `s-` | macOS Command/Super |
| `S-SPC` | Shift + Space |
| `C-c h` → `w` | `C-c h`를 누른 뒤 `w` |

## 한글 입력

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| 한/영 전환 | `S-SPC` | Emacs 내장 `korean-hangul` 입력기 토글 |
| ghostel 터미널에서 한글 입력 | `S-SPC` | `ghostel-ime-mode`가 터미널 안 입력을 처리 |
| 한글 입력 중 Control/Meta 키 사용 | 기존 `C-...`, `M-...` | 두벌식 자모를 영문 키로 변환해 주요 키바인딩 유지 |

## 명령 실행과 후보 액션

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| 명령 실행 | `M-x` | Vertico 세로 완성으로 명령 선택 |
| 현재 후보에 대한 액션 실행 | `C-.` | Embark 액션 메뉴 |
| 현재 후보에 대한 기본 액션 실행 | `C-;` | Embark DWIM |
| 현재 키맵/바인딩 보기 | `C-h B` | Embark bindings |
| 모드별 명령 실행 | `C-c M-x` | `consult-mode-command` |
| 키보드 매크로 선택 | `C-c k` | `consult-kmacro` |
| Info 문서 검색 | `C-c i` | `consult-info` |
| 이전 복합 명령 선택 | `C-x M-:` | `consult-complex-command` |
| 미니버퍼 히스토리 검색 | `M-s`, `M-r` | 미니버퍼 안에서 `consult-history` |

## 파일과 버퍼

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| 파일 열기 | `C-x C-f` | 기본 `find-file` |
| 현재 작업공간 버퍼 전환 | `C-x b` | 현재 Perspective + 공유 버퍼만 표시 |
| 전체 버퍼에서 전환/가져오기 | `C-u C-x b` | 전체 `consult-buffer` 소스 사용 |
| 다른 창에서 버퍼 열기 | `C-x 4 b` | `consult-buffer-other-window` |
| 프로젝트 버퍼 전환 | `C-x p b` | `consult-project-buffer` |
| 북마크 열기 | `C-x r b` | `consult-bookmark` |
| kill-ring에서 붙여넣기 선택 | `M-y` | `consult-yank-pop` |

## 레지스터

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| 레지스터 불러오기 | `M-#` | `consult-register-load` |
| 레지스터 저장하기 | `M-'` | `consult-register-store` |
| 레지스터 선택/관리 | `C-M-#` | `consult-register` |

## 검색과 이동

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| 현재 버퍼 검색 | `C-s`, `M-s l` | `consult-line` |
| 여러 버퍼 검색 | `M-s L` | `consult-line-multi` |
| 파일 찾기 | `M-s d` | `consult-find` |
| grep 검색 | `M-s g` | `consult-grep` |
| git grep 검색 | `M-s G` | `consult-git-grep` |
| ripgrep 검색 | `M-s r` | `consult-ripgrep` |
| 검색 결과만 남기기 | `M-s k` | `consult-keep-lines` |
| 검색 결과에 초점 맞추기 | `M-s u` | `consult-focus-lines` |
| isearch 히스토리 | `M-s e` | isearch 중에도 사용 가능 |
| 줄 이동 | `M-g g`, `M-g M-g` | `consult-goto-line` |
| 컴파일 에러 이동 | `M-g e` | `consult-compile-error` |
| Flymake 진단 이동 | `M-g f` | `consult-flymake` |
| Outline 이동 | `M-g o` | `consult-outline` |
| mark 이동 | `M-g m` | `consult-mark` |
| global mark 이동 | `M-g k` | `consult-global-mark` |
| imenu 이동 | `M-g i` | 현재 버퍼 심볼 이동 |
| 여러 버퍼 imenu 이동 | `M-g I` | `consult-imenu-multi` |
| 화면 안 문자 점프 | `C-'` | `avy-goto-char-2` |

## 버퍼 내 자동완성

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| 들여쓰기 또는 완성 | `TAB` | `tab-always-indent`가 `complete`로 설정됨 |
| 보완 소스 선택 | `C-c e` | Cape prefix map |

## 프로젝트

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| 프로젝트 Hydra 열기 | `C-c h` → `p` | 프로젝트 작업 모음 |
| 프로젝트+기본 작업공간 전환 | `C-x p p` | Perspective 전환/생성 후 프로젝트 Dired |
| 현재 작업공간에 프로젝트 추가 | `C-u C-x p p` | Perspective를 유지하여 여러 저장소를 함께 사용 |
| 프로젝트 파일 찾기 | `C-c h` → `p` → `f` | `project-find-file` |
| 프로젝트 검색 | `C-c h` → `p` → `s` | `project-find-regexp` |
| 프로젝트 버퍼 전환 | `C-c h` → `p` → `b` | `project-switch-to-buffer` |
| 프로젝트 Dired 열기 | `C-c h` → `p` → `d` | `project-dired` |
| 프로젝트 버퍼 모두 닫기 | `C-c h` → `p` → `k` | `project-kill-buffers` |
| 프로젝트 컴파일 | `C-c h` → `p` → `c` | `project-compile` |
| 임시 Perspective 생성·전환 | `C-x x s` | 없는 이름이면 생성, 있으면 기존 상태로 전환 |
| 현재 Perspective 종료 | `C-x x c` | 임시 작업을 닫고 제거 |
| 직전 Perspective 복귀 | `C-x x l` | 직전에 사용한 작업공간으로 즉시 전환 |
| 버퍼 추가 / 전역 공유 | `C-x x a` / `C-x x g` | 현재 작업공간에 추가 / 프레임 전역 Perspective에 공유 |

Perspective에는 프로젝트 밖 Org/참고 파일, Dired, shell, REPL, compilation 버퍼도
포함할 수 있다. 파일과 Dired 및 창 배치는 정상 종료 후 복원 대상이지만, 실행 중인
프로세스 자체는 재생성하지 않는다.

## 창 관리

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| 창 선택 | `M-o` | `ace-window` |
| 창 관리 Hydra 열기 | `C-c h` → `w` | 창 이동/분할/삭제/크기 조절 |
| 왼쪽/오른쪽/아래/위 창으로 이동 | `C-c h` → `w` → `h/l/j/k` | `windmove-*` |
| 가로/세로 분할 | `C-c h` → `w` → `s/v` | 아래/오른쪽으로 분할 |
| 현재 창 삭제 | `C-c h` → `w` → `d` | `delete-window` |
| 다른 창 모두 삭제 | `C-c h` → `w` → `D` | `delete-other-windows` |
| 창 안 버퍼 전환 | `C-c h` → `w` → `b` | `consult-buffer` |
| 파일 열기 | `C-c h` → `w` → `f` | `find-file` |
| ace-window 실행 | `C-c h` → `w` → `a` | 창 빠른 선택 |
| 창 스왑 | `C-c h` → `w` → `m` | `ace-swap-window` |
| 창 크기 조절 | `C-c h` → `w` → `H/L/J/K` | 좌우/상하 크기 조절 |

## IntelliJ 스타일 프로젝트 도구 창

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| 파일 트리 포커스/닫기 | `s-1` | 편집 버퍼에서는 Treemacs로 이동; Treemacs 안에서는 닫기 |
| 편집 버퍼로 복귀 | `ESC` | Treemacs는 유지하고 마지막으로 사용한 편집 창으로 이동 |
| 북마크 목록 토글 | `s-2` | Emacs bookmark 목록; 다시 누르면 닫기 |
| 코드 구조 토글 | `s-7` | 현재 버퍼의 Imenu Structure; 다시 누르면 닫기 |
| 도구 창 새로고침 / 닫기 | `g` / `q` | Bookmarks·Structure 창 안에서 사용 |
| Treemacs 토글 | `C-x t t` | 파일 트리 열기/닫기 |
| Treemacs 창 선택 | `M-0` | Treemacs 창으로 이동 |
| Treemacs만 남기기 | `C-x t 1` | 다른 창 정리 |
| 추가 분할 없이 파일 열기 | `RET` / `o o` | 기존 편집 창에 선택한 파일 표시 |
| 열 창 직접 선택 | `o a a` | `ace-window`로 대상 창 선택 |
| 최근 편집 창에 열기 | `o r` | 가장 최근에 사용한 비-Treemacs 창에 표시 |
| 분할해서 열기 | `o v` / `o h` | 세로 / 가로 분할을 만든 뒤 파일 표시 |
| 파일을 열고 트리 닫기 | `o c` | 파일을 표시한 뒤 Treemacs 창 닫기 |
| 디렉터리 선택 | `C-x t d` | `treemacs-select-directory` |
| 북마크 열기 | `C-x t B` | `treemacs-bookmark` |
| 현재 파일 찾기 | `C-x t C-t` | `treemacs-find-file` |
| 태그 찾기 | `C-x t M-t` | `treemacs-find-tag` |
| 마스터 Hydra에서 열기 | `C-c h` → `t` | `s-1`과 같은 파일 트리 토글 |

`s-1`, `s-2`, `s-7`은 동일한 왼쪽 도구 창 슬롯을 공유한다. 다른 보기의
키를 누르면 현재 보기를 닫고 새 보기로 교체한다. Treemacs가 이미 보이면
편집 버퍼의 `s-1`은 파일 트리에 포커스를 주고, 그 안의 `s-1`은 파일 트리를
닫는다. `ESC`는 파일 트리를 열어 둔 채 마지막 편집 창으로 돌아간다.
기본 `RET`은 기존 편집 창을 재사용한다. Treemacs만 남아 있다면 파일을 표시할
편집 창이 필요하므로 그때만 옆에 새 창을 만든다.

## Git

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| Git Hydra 열기 | `C-c h` → `g` | Magit 작업 모음 |
| 상태 보기 | `C-c h` → `g` → `s` | `magit-status` |
| 로그 보기 | `C-c h` → `g` → `l` | `magit-log-current` |
| blame 보기 | `C-c h` → `g` → `b` | `magit-blame` |
| diff 보기 | `C-c h` → `g` → `d` | `magit-diff-dwim` |
| 변경 표시 확인 | 여백(fringe) | `diff-hl`이 Git 변경을 표시 |

## 편집

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| 실행 취소 | `C-z` | `undo-fu-only-undo` |
| 다시 실행 | `C-S-z` | `undo-fu-only-redo` |
| 저장 시 끝 공백 제거 | 저장 시 자동 | `stripspace` |
| 저장 시 포매팅 | 저장 시 자동 | `apheleia`가 major mode별 formatter 실행 |
| 괄호/따옴표 짝 자동 입력 | 입력 시 자동 | `electric-pair-mode` |
| 선택 영역 덮어쓰기 | 입력 시 자동 | `delete-selection-mode` |
| 현재 버퍼 닫기 | `s-w` | IntelliJ의 에디터 탭 닫기와 동일; 수정 사항이 있으면 저장 여부 확인 |

## 코드 폴딩

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| 현재 폴드 토글 | `C-c z a` | 열기/닫기 전환 |
| 현재 폴드 열기 | `C-c z o` | `kirigami-open-fold` |
| 재귀적으로 열기 | `C-c z O` | `kirigami-open-fold-rec` |
| 모든 폴드 열기 | `C-c z r` | `kirigami-open-folds` |
| 현재 폴드 닫기 | `C-c z c` | `kirigami-close-fold` |
| 모든 폴드 닫기 | `C-c z m` | `kirigami-close-folds` |

## 터미널

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| ghostel 터미널 열기 | `C-c t` | libghostty-vt 기반 터미널 |
| 터미널에서 한/영 전환 | `S-SPC` | `ghostel-ime-mode` 사용 |

## Obsidian/Markdown

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| 노트 캡처 | `C-c C-n` | Obsidian 버퍼에서 `obsidian-capture` |
| 링크 삽입 | `C-c C-l` | `obsidian-insert-link` |
| 링크 따라가기 | `C-c C-o` | `obsidian-follow-link-at-point` |
| 노트 점프 | `C-c C-p` | `obsidian-jump` |
| 백링크 점프 | `C-c C-b` | `obsidian-backlink-jump` |
| Markdown TOC 생성/갱신 | `M-x markdown-toc-generate-toc`, `M-x markdown-toc-refresh-toc` | 명령으로 실행 |

## 도움말

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| 함수 도움말 | `C-h f` | Helpful로 remap |
| 변수 도움말 | `C-h v` | Helpful로 remap |
| 키 도움말 | `C-h k` | Helpful로 remap |
| 명령 도움말 | `C-h x` | Helpful로 remap |
| 심볼 도움말 | `C-h o` | Helpful로 remap |

## 텍스트 확대/축소

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| 확대/축소 Hydra 열기 | `C-c h` → `z` | 텍스트 크기 조절 |
| 확대 | `C-c h` → `z` → `i` | `text-scale-increase` |
| 축소 | `C-c h` → `z` → `o` | `text-scale-decrease` |
| 초기화 | `C-c h` → `z` → `0` | `text-scale-set 0` |

## macOS Command 키

| 하고 싶은 일 | 단축키 | 설명 |
|---|---|---|
| 복사 | `s-c` | `kill-ring-save` |
| 붙여넣기 | `s-v` | `yank` |
| 잘라내기 | `s-x` | `kill-region` |
| 실행 취소 | `s-z` | 기본 `undo` |
| 전체 선택 | `s-a` | `mark-whole-buffer` |

## 자주 쓰는 진입점

| 진입점 | 용도 |
|---|---|
| `C-c h` | 마스터 Hydra: 창, 프로젝트, Git, 확대/축소, Treemacs |
| `C-x p` | 내장 `project.el` 명령 (`p`는 Perspective 연동 전환) |
| `C-x x` | Perspective 작업공간 관리 |
| `C-c z` | 코드 폴딩 |
| `C-c e` | Cape 보완 소스 |
| `C-c t` | ghostel 터미널 |
