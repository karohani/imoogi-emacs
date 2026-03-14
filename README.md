# imoogi-emacs

개인 Emacs 설정. 모듈별로 분리하여 관리한다.

## 설치

```bash
# 1. 클론
git clone <repo-url> ~/workspace/imoogi-emacs

# 2. 심볼릭 링크
ln -s ~/workspace/imoogi-emacs ~/.config/imoogi-emacs

# 3. init.el 설정 (~/.emacs.d/init.el)
```

```elisp
(load-file (expand-file-name "boot.el" "~/.config/imoogi-emacs"))
```

첫 실행 시 패키지가 자동 설치된다.

## 주요 키바인딩

| 키 | 기능 |
|----|------|
| `C-c h` | 마스터 hydra (창, 프로젝트, Git, 줌, treemacs) |
| `C-c p` | projectile 커맨드 맵 |
| `M-o` | ace-window (창 점프) |
| `S-SPC` | 한영 전환 |
| `C-s` | swiper 검색 |
| `M-x` | counsel-M-x |

## 패키지 관리

- **package.el + use-package** — MELPA/GNU ELPA/NonGNU ELPA 패키지
- **straight.el** — GitHub 직접 설치가 필요한 패키지 (obsidian.el 등)
