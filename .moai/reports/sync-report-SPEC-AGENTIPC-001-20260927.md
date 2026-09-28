# Sync 보고서 — SPEC-AGENTIPC-001

- 작성일: 2026-09-27
- 경로: Route A (main 직접 커밋·푸시), CHANGELOG 생략(사용자 결정, HUMAN GATE 2)
- sync 커밋: `69ee8c1aca9d520834beb3017ed3fed96465e2dd` (`e1f3fc7..69ee8c1 main -> main`, 푸시 후 차이 `0 0`)
- SPEC 상태: `spec.md` 의 `status` 를 `completed` 로 변경(plan.md·acceptance.md 에는 frontmatter 가 없어 변경 없음)

## 갱신한 문서

| 파일 | 변경 내용 |
|------|-----------|
| `.moai/project/product.md` | CLI 수 6→7, 코딩 에이전트 이벤트 전달 역할 추가 |
| `.moai/project/structure.md` | 개요 문장에 호출 방향 예외 명시, 트리에 `cmd/imoogi-agent`·`internal/agentipc`·`30-agent.el` 추가, 최상위 패키지 12→13 |
| `.moai/project/tech.md` | CLI 수 6→7, CLI 표에 `imoogi-agent` 행 추가, 패키지 34→36(internal 27→28), `build-all`·`build-agent` |
| `.moai/project/codemaps/overview.md` | "모든 Go CLI 는 Emacs 가 하위 프로세스로 호출" 문장에 `imoogi-agent` 예외 추가, 수치 갱신 |
| `.moai/project/codemaps/entry-points.md` | `imoogi-agent` 진입점 행 추가, `build-all` 7개, `build-agent` |
| `.moai/project/codemaps/data-flow.md` | 흐름 6→7, "7. Coding agent → Emacs" 절 신설 |
| `README.md` | 설치 절 아래에 `imoogi-agent` 사용법 절 신설(설치 링크, 예시, 종료 코드, `*imoogi-agent*` 로그, 산출물 파일 관례) |

## 품질 게이트

- make ci-local: 종료 코드 0 — 테스트 490개 중 기대 결과 488, 예상 밖 0, 건너뜀 2. Go 전부 ok, setup-toolchain PASS (로그 `.moai/state/verify/df00ce0d/ci-local.log`). 푸시 시 pre-push 훅에서도 다시 통과.
- 커버리지: `cmd/imoogi-agent` 100.0%, `internal/agentipc` 97.2%
- 린트: go vet ok, golangci-lint 이슈 0
- MX 태그: P1/P2 위반 없음
- 4차원 감사: PASS, 조화평균 0.873 (기능 0.93 / 보안 0.87 / 품질 0.82 / 일관성 0.88), 기준 0.80, 치명·쟁점 없음 — `.moai/reports/sync-audit/SPEC-AGENTIPC-001-4dim.json`

## 감사 지적 사항

모두 minor. 4건은 백로그 t18–t21 로 넘겼다. JSON 에는 16개 항목이 있으며, 마지막 항목은 코드가 아닌 커밋 작성자 기록에 관한 메모다.

1. [Functionality/minor] `tests/agent-test.el` — AC-004 test cannot tell whether artifactType is logged: its check `(string-match-p "report" (car log))` also matches the fixture filename `260927-104700-report.md`, so it passes either way.
2. [Functionality/minor] `internal/agentipc/emacsclient.go` — Event-file cleanup relies only on `defer os.Remove(path)` inside Emacsclient.Deliver, and the CLI installs no signal handling.
3. [Security/minor] `modules/development/30-agent.el` — The receiver does not check who owns the event file or its mode.
4. [Security/minor] `internal/agentipc/emacsclient.go` — When EMACSCLIENT is a bare name (no slash), the CLI resolves it against the current working directory with filepath.Abs and runs ./<name>.
5. [Security/minor] `modules/development/30-agent.el` — Nothing limits the size of the payload file the receiver opens.
6. [Security/minor] `internal/agentipc/emacsclient.go` — Leftover-data hygiene: the event file is removed only by a deferred call in Deliver, and the CLI installs no signal handler.
7. [Craft/minor] `cmd/imoogi-agent/main_test.go` — The fake emacsclient helper (about 45 lines) is copied between the two test packages, and the copies have already drifted apart.
8. [Craft/minor] `internal/agentipc/request.go` — parseRequest is a single function of about 125 lines with cognitive complexity 53, above the gocognit default of 30.
9. [Craft/minor] `cmd/imoogi-agent/main_test.go` — The 'default limit is 5 seconds' subtest spends 5 s of real sleep, and the whole TestAC14CLITimeLimits test takes about 8 s.
10. [Craft/minor] `internal/agentipc/emacsclient.go` — Dependency injection is done two different ways.
11. [Craft/minor] `modules/development/30-agent.el` — The (min column most-positive-fixnum) guard in imoogi-agent--display has no stated reason.
12. [Craft/minor] `tests/agent-test.el` — There is duplication in the ERT test file.
13. [Craft/minor] `modules/development/30-agent.el` — The module header comment says "Only defvar/defconst/defun forms live here".
14. [Consistency/minor] `internal/agentipc/emacsclient_test.go` — The two helper-process fake emacsclients in the same SPEC follow different conventions.
15. [Consistency/minor] `internal/agentipc/agentipc.go` — Transport.Deliver takes `timeout time.Duration` and creates its own context from context.Background().
16. [Consistency/minor] `internal/agentipc/emacsclient.go` — This is history metadata, not code style, and it does not affect the score.

## 미검증 사항과 참고

- AC-015 데몬 스모크 테스트는 sync 단계에서 다시 돌리지 않았다(읽기 전용 리뷰). run 단계 기록에 의존한다.
- 커밋 `4b72a86` 의 작성자가 `t <t@example.invalid>` 로 남아 있다(git 설정 손상, `43f88d1` 에서 수정). 이력은 다시 쓰지 않았다.
- `structure.md` 9행의 "Measured at HEAD `efa3567`" 수치(34 Go packages 등)와 로드 순서 표(30개 항목, `29-org-roam`·`30-agent` 없음)는 과거 시점 스냅숏이라 그대로 두었다. `codemaps/modules.md` 의 "27 packages" 도 이번 범위(plan §7) 밖이라 고치지 않았다.
- 백업: `.moai/backups/sync-20260927-210300`
