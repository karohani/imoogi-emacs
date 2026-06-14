# CLAUDE.md

**먼저 [`AGENTS.md`](AGENTS.md) 를 읽으세요.** 이 저장소의 에이전트 작업 규칙·제약·워크플로는 모두 거기에 단일 관리됩니다.

특히 다음을 반드시 숙지할 것:

- **망분리(air-gap) 제약** — 부팅 경로에 네트워크 의존을 추가하지 말 것. 모든 패키지는 `vendor/elpa/` 에 동봉.
- **모듈 작성 규칙** — 최상단 `(imoogi-require ...)`, `use-package`, `(provide ...)`, `boot.el` 등록.
- **패키지 추가/업데이트** — 온라인 머신에서 `scripts/vendor.el` 실행 후 커밋(폐쇄망 내 금지).
- **검증** — 문법 + 오프라인 부팅 테스트.

세부 사항·예시는 `AGENTS.md`, 사람용 개요는 `README.md`, 구조는 `ARCHITECTURE.md` 를 참고하세요.
