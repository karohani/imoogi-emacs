# External ticket task creation plan

Goal: native Org property completion offers EXTERNAL_SYSTEM / EXTERNAL_ID / EXTERNAL_URL, and imoogi-emacsctl task-create creates a new TODO with optional external metadata.

- [x] Add Go parser and real receiver regression tests; observe failures.
- [x] Extend CLI envelope with task-create and validated single-line metadata.
- [x] Add configurable native Org system suggestions, including custom names.
- [x] Add receiver validation and safe Inbox task creation with unique ID, preserving dirty/stale/read-only/locked targets.
- [x] Run scoped Go/ERT checks and an isolated CLI-to-Emacs integration check. Final ERT 40/40; both Go packages and actual CLI/server integration passed.
- [x] Document usage, file/default behavior, customization and retry limitations; record evidence.

The command uses the existing protocol version and ok/error response. Explicit --file targets must already exist and be local .org files. The default agenda file may be created only within an existing directory. Inputs validate before any write. No edit-existing command, automatic retry or external system API calls are introduced.

Review disposition: new tasks use the target's first unfinished Org keyword (TODO by default), rejecting targets with no unfinished keyword before mutation. JSON and CLI validation both reject Unicode controls, Unicode whitespace-only fields and malformed URL authority/percent escapes, while preserving ordinary encoded URLs and bracketed IPv6.

Claude 검토 후 보완: 현재 본문의 `#+TODO` 선언과 전역 키워드를 입력으로 상태를 다시 계산한다. 파일·폴더 로컬 변수는 적용하지 않는다. preflight 중복과 불필요한 잠금 플래그를 제거했다. 저장 이후 오류는 실제 디스크 내용을 비교하여 처리하며, 저장 완료 후 ID 색인과 알림 실패는 성공을 뒤집지 않는다. 관련 ERT 50/50 통과. 상세 재현과 한계는 `.moai/reports/external-tickets-claude-disposition.md`에 기록했다.
계산 중 전역 mode hook 차단과 기존 버퍼 상태 캐시 갱신도 추가 검증했으며 최종 scoped ERT는 51/51이다.

최종 독립 검토 F1은 `local-write-file-hooks`가 t를 반환하면 저장을 우회하는 회귀 테스트로 재현했다. 다른 저장 hook과 함께 nil로 바인딩한 뒤 실제 디스크와 버퍼의 일치를 확인했다. F2는 `org-file-contents`를 가로채는 테스트로 SETUPFILE 외부 입력 의존성을 확인하고 문서화했으며 Org 기본 처리 정책은 유지했다.

최신 검증: `IMOOGI_TEST_USER_DIR=/private/tmp/external-ticket-ert IMOOGI_ERT_SELECTOR='imoogi-external-\|imoogi-emacsctl-\|imoogi-org-setup\|imoogi-org-directory' /Applications/Emacs.app/Contents/MacOS/Emacs --batch -Q -l tests/run.el` → `Ran 53 tests, 53 results as expected, 0 unexpected (2026-11-17 08:03:44+0900, 2.771924 sec)`. RED 로그는 `/private/tmp/external-final-hook-red.log`, GREEN 로그는 `/private/tmp/external-final-hook-green.log`에 기록했다.
