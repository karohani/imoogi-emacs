# Sync 보고서 — SPEC-ANKICARD-005 (2026-09-28)

- 커밋: `0d6771f` (main, origin/main에 push 완료, 차이 `0 0`)
- README.md: Anki 여러 줄 카드 절에 보조 블록 비중첩, `extra_block_unbalanced` 건너뛰기, 표식 글자 표시 방법(verbatim·줄 중간·쉼표 이스케이프)을 추가.
- `.moai/project/domains.md`: 해소된 Known gaps `t16` 줄 삭제.
- SPEC 종료: spec.md `status: completed`, progress.md §E.4 기록(미추적 파일).
- 계획 보고서 정정: `anki-card-types-plan-20260920.md`의 "중첩된 `#+BEGIN_EXTRA`(정상 Org)" 전제에 정정 주석(Org 9.7.11은 같은 이름 블록을 중첩하지 않음).
- 검증: 오케스트레이터 재실행(6678118) — elisp 496개 중 예상 외 0, orgdoc 100.0%, planner 92.8%, go vet·gofmt 통과. Tier S라 4차원 감사는 하지 않음.
- 후속: SPEC-ANKICARD-004 재개 시 manager-spec이 REQ-SW-001.7, spec.md:972-980, acceptance.md:497·118을 고친다.
