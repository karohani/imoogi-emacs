# Language Toolchain 운영 가이드

이 저장소는 Go/TypeScript LSP 실행 파일까지 폐쇄망에 반입한다. 온라인 빌드
머신만 `fetch`를 실행하고, 폐쇄망 타겟은 커밋된 bootstrap으로 `setup`만 실행한다.
Emacs 부팅 경로에서는 두 명령 모두 자동 실행하지 않는다.

## 현재 고정 버전

| 구분 | 버전 |
|---|---|
| CLI | `1.0.0` |
| bundle | `2026.08.22.1` |
| target | `darwin/arm64` |
| gopls | `v0.23.0` (`014f87ff5c01915bc90f4f11a6bb8aea3e0edbd7`) |
| Node.js | `v24.19.0` LTS |
| TypeScript | `6.0.3` |
| typescript-language-server | `6.0.0` (`1c0f224eb44c626d96dae07aaf5d78654de0e1f2`) |

CLI는 SemVer(`MAJOR.MINOR.PATCH`), 설치 bundle은 CalVer
(`YYYY.MM.DD.N`)를 쓴다. 예를 들어 같은 날 두 번째 bundle은
`2026.08.22.2`다. `v.260101.1`처럼 SemVer와 CalVer가 섞인 표기는 허용하지
않는다.

TypeScript 7은 네이티브 플랫폼 패키지를 사용하고 기존 `tsserver.js`를 제공하지
않는다. 현재 typescript-language-server 6의 SDK 탐색 방식과 맞지 않으므로 이
bundle은 TypeScript 6.0.3을 의도적으로 고정한다.

## 명령 경계

모든 명령은 저장소 루트 또는 그 하위 디렉터리에서 실행할 수 있다.

```sh
TOOL=vendor/toolchains/cli/1.0.0/darwin-arm64/imoogi-toolchain
$TOOL --help
$TOOL fetch --help
$TOOL setup --help
$TOOL version
```

- `fetch`: 온라인 전용. `toolchains.json`의 정확한 핀만 내려받고, Node 공식
  `SHASUMS256.txt`를 확인하며, gopls와 bootstrap을 재현 가능한 Go 빌드 옵션으로
  만든다. 결과는 `vendor/toolchains/`와 `toolchains.lock.json`에 기록한다.
- `setup`: 오프라인 전용. 네트워크나 시스템 Go/Node를 사용하지 않는다. 모든
  아티팩트의 크기와 SHA-256을 먼저 확인한 뒤 같은 파일시스템의 staging에서
  materialize/probe하고, `.local/toolchains/<bundle>/`을 불변 설치한 후 상대
  symlink `.local/bin`을 원자적으로 활성화한다. 설치 트리의 결정적 SHA-256은
  `install.json` v2에 기록하며, 재사용할 때 내용·권한·심볼릭 링크를 다시 검증한다.
- `version`: CLI/desired/available/active bundle과 구성요소 버전을 읽기 전용으로
  보고한다.
- 모든 `--help` 경로는 manifest, 설치 상태, 네트워크를 조회하지 않는다.

## 온라인 갱신

일반적인 upstream 갱신은 다음 순서를 따른다.

1. `toolchains.json`의 upstream 버전과 revision, runtime 제약을 수정한다.
2. bundle CalVer를 증가시킨다. CLI 호환 계약이나 구현을 배포할 때만 CLI
   SemVer도 증가시킨다.
3. 온라인 `darwin/arm64` 빌드 머신에서 fetch를 실행한다.

```sh
vendor/toolchains/cli/1.0.0/darwin-arm64/imoogi-toolchain fetch
```

CLI 자체를 변경해 새 bootstrap이 필요하면 온라인 머신의 Go로 현재 소스를 먼저
실행한다.

```sh
go run ./cmd/imoogi-toolchain fetch
```

4. `toolchains.lock.json`, 새 아티팩트, `vendor/toolchains/licenses/`, bootstrap
   provenance를 함께 검토한다. 같은 identity에서 upstream 바이트가 바뀌면 fetch는
   drift로 실패하고 기존 lock을 보존한다.
5. `go test ./...`, `go vet ./...`, 깨끗한 복사본의 `setup`과 도구 probe를
   통과시킨다.
6. 손으로 쓴 코드/manifest와 큰 바이너리 `vendor/toolchains/` 변경은 저장소
   관례에 따라 별도 커밋으로 반입한다.

폐쇄망에서는 `fetch`와 `go run`을 실행하지 않는다.

## 폐쇄망 설치와 확인

```sh
vendor/toolchains/cli/1.0.0/darwin-arm64/imoogi-toolchain setup
vendor/toolchains/cli/1.0.0/darwin-arm64/imoogi-toolchain version
.local/bin/gopls version
.local/bin/node --version
.local/bin/tsc --version
.local/bin/typescript-language-server --version
```

setup은 반복 실행해도 같은 lock이면 `setup reused <bundle>`로 끝난다. `17-lsp`는
`.local/bin`이 저장소 내부의 유효한 상대 symlink일 때만 `exec-path`와 `PATH`의
맨 앞에 한 번 추가한다. 설치가 없거나 링크가 잘못되면 시스템 PATH를 변경하지
않고 해당 언어 설정만 기존 fallback 동작을 유지한다.

## 감사와 무결성

현재 직접 아티팩트의 SHA-256은 다음과 같다. 설치 시에는 CLI가 이 값과 lock의
크기를 모두 다시 계산한다.

```text
a5ea1ffa79f38b8e59a25e06fb3c198defcee6c44e8b757f1a478b05b06d436f  vendor/toolchains/cli/1.0.0/darwin-arm64/imoogi-toolchain
8de8bfd7d4f9e36f2d8fc679061dbd6f982553b1f32e84e7c658f6da4c9c64a8  vendor/toolchains/go/gopls/v0.23.0/darwin-arm64/gopls
8294b7aa9b03997481c06babf1e8b270c859358f27da57a11509afe537ac381d  vendor/toolchains/node/v24.19.0/darwin-arm64/node-v24.19.0-darwin-arm64.tar.gz
33cd0ee1beaa8c9e9d15a9da836c62ddea4c34a42d7c2d349dbc80d94165d22a  vendor/toolchains/typescript/typescript/6.0.3/darwin-arm64/typescript.tgz
6e23b48efc76af4e70928cdfe62ea6e6cfef67ab4c1e7579c4e82dd284fbdfd2  vendor/toolchains/typescript/typescript-language-server/6.0.0/darwin-arm64/server.tgz
```

bootstrap의 빌드 명령, Go 버전, 빌드 시각, 크기와 SHA-256은 같은 디렉터리의
`imoogi-toolchain.provenance.json`에 있다. upstream 라이선스와 제3자 고지는
`vendor/toolchains/licenses/`에 원본 이름을 보존해 동봉한다.

설치 후의 `install.json`은 lock SHA-256과 `install.json` 자체를 제외한 전체 설치
트리 SHA-256을 함께 기록한다. setup 재실행은 probe보다 먼저 이 값을 재계산하므로
실행 파일 내용, 파일 권한, 링크 대상, 하위 디렉터리 파일의 drift를 실패로 처리한다.

## 실패와 rollback

- 잘못된 manifest/lock 또는 아티팩트 손상은 종료 코드 `10`이며 materialize 전에
  중단한다.
- 기존 설치 bundle의 트리 무결성 불일치도 종료 코드 `10`이며, 변조된 bundle을
  probe하거나 다시 활성화하지 않는다. `.local/`을 삭제한 뒤 커밋된 아티팩트로
  재설치할 수 있다.
- setup lock 경합은 `11`, probe 실패는 `12`, 그 밖의 setup 실패는 `13`이다.
- 새 bundle의 검증, probe, publish, 활성화가 끝나기 전까지 `.local/bin`은 이전
  bundle을 계속 가리킨다.
- rollback은 원하는 과거 git revision의 `toolchains.json`, lock,
  `vendor/toolchains/`를 복원한 뒤 그 revision의 committed bootstrap으로 `setup`을
  다시 실행한다. `.local/bin`을 손으로 수정하지 않는다.
- `.local/`은 생성물이며 git에 커밋하지 않는다. 삭제해도 committed artifact로
  다시 setup할 수 있다.
