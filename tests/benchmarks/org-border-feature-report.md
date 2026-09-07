# 실제 Org 테두리 구현 검증

2026-09-07. `modules/org/imoogi-org-border.el`을 기존 색상 표시와 비교했다.
격리 GUI Emacs 31.1에서 1만/100만 줄, 여러 제목/긴 본문, 순방향/역방향의
8개 실행이 모두 정상 종료했다. Org 메뉴는 두 비교군 모두 제외했다.

100만 줄 순방향 입력+갱신+redisplay 중앙값:

| 구조 | 기존 색상 | 실제 테두리 |
|---|---:|---:|
| 여러 제목 | 0.82ms | 1.15ms |
| 긴 본문 | 0.84ms | 2.84ms |

실사용은 0.1초 유휴 타이머로 갱신하지만, 이 비교는 매 작업마다 갱신을
강제로 포함한다. 키보드 입력부터 OS 화면 합성까지의 지연은 아니다.
표시 overlay 최대 200개 제한도 확인했다. 먼 위치 이동 중앙값은 약
19~20ms이므로 모든 동작의 무지연을 주장하지 않는다.

별도 GUI에서 실제 idle timer로 최초 표시, 하위 제목 이동 후 갱신,
모드 해제 후 timer/overlay 정리를 확인했다. 전용 ERT 10개 통과.
오프라인 부팅 통과. 전체 ERT는 256/257 통과했으며 남은 1개는 기존
`imoogi-org-preview-browser-navigation-moves-point-and-suppresses-echo` 실패다.

[원시 데이터](org-border-feature-results.json).

재현:

```sh
python3 tests/benchmarks/run-org-border.py \
  --emacs /Applications/Emacs-31.1.app/Contents/MacOS/Emacs \
  --output /tmp/imoogi-border-feature-new --without-org-menu \
  --variants baseline production --sizes 10000 1000000
```
