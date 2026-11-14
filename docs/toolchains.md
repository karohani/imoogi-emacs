# Language Toolchain 운영 가이드

이 저장소는 Go/TypeScript LSP 실행 파일까지 폐쇄망에 반입한다. 온라인 빌드
머신만 `fetch`를 실행하고, 폐쇄망 타겟은 커밋된 bootstrap으로 `setup`만 실행한다.
Emacs 부팅 경로에서는 두 명령 모두 자동 실행하지 않는다.

## 현재 고정 버전

| 구분 | 버전 |
|---|---|
| CLI | `1.0.0` |
| bundle | `2026.11.14.2` |
| target | `darwin/arm64` |
| gopls | `v0.23.0` (`014f87ff5c01915bc90f4f11a6bb8aea3e0edbd7`) |
| Node.js | `v24.19.0` LTS |
| TypeScript | `6.0.3` |
| typescript-language-server | `6.0.0` (`1c0f224eb44c626d96dae07aaf5d78654de0e1f2`) |
| claude-agent-acp | `0.99.1` (ACP 어댑터, 아티팩트는 로컬 빌드) |
| codex-acp | `2.5.0` (ACP 어댑터) |

CLI는 SemVer(`MAJOR.MINOR.PATCH`), 설치 bundle은 CalVer
(`YYYY.MM.DD.N`)를 쓴다. 예를 들어 같은 날 두 번째 bundle은
`2026.08.22.2`다. `v.260101.1`처럼 SemVer와 CalVer가 섞인 표기는 허용하지
않는다.

TypeScript 7은 네이티브 플랫폼 패키지를 사용하고 기존 `tsserver.js`를 제공하지
않는다. 현재 typescript-language-server 6의 SDK 탐색 방식과 맞지 않으므로 이
bundle은 TypeScript 6.0.3을 의도적으로 고정한다.

## 명령 경계

일반 설치와 재설치는 플랫폼이나 CLI 버전 경로를 직접 입력하지 않는다. wrapper가
`uname`과 `toolchains.lock.json`을 읽어 현재 머신에 맞는 동봉 bootstrap을 고른다.

```sh
./scripts/install.sh       # Emacs 연결 + 호환되는 언어 서버 자동 setup
make toolchain-setup       # 언어 서버만 별도로 setup/재검증
```

호환 bootstrap이 동봉되지 않은 플랫폼에서 `install.sh`은 Emacs 연결을 완료하고
언어 서버만 건너뛴다. `make toolchain-setup`은 이 경우 실패하므로 반입 누락을
명확하게 확인할 수 있다. 언어 서버가 필요 없으면
`./scripts/install.sh --without-toolchain`을 사용한다.

낮은 수준의 CLI 명령은 저장소 루트 또는 그 하위 디렉터리에서 실행할 수 있다.

```sh
TOOL=./scripts/setup-toolchain.sh
$TOOL --help
$TOOL fetch --help
$TOOL setup --help
$TOOL version
```

- `fetch`: 온라인 전용. `toolchains.json`의 정확한 핀만 내려받고, Node 공식
  `SHASUMS256.txt`를 확인하며, gopls와 bootstrap을 재현 가능한 Go 빌드 옵션으로
  만든다. 결과는 `vendor/toolchains/`와 `toolchains.lock.json`에 기록한다.
- `setup`: 오프라인 전용(예외: 아래 ACP 어댑터의 로컬 빌드 — 실패하면 그 어댑터만
  건너뛴다). 네트워크나 시스템
  Go/Node를 사용하지 않는다. 모든
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
./scripts/setup-toolchain.sh fetch
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
make toolchain-setup
./scripts/setup-toolchain.sh version
.local/bin/gopls version
.local/bin/node --version
.local/bin/tsc --version
.local/bin/typescript-language-server --version
```

setup은 반복 실행해도 같은 lock이면 `setup reused <bundle>`로 끝난다. `17-lsp`는
`.local/bin`이 저장소 내부의 유효한 상대 symlink일 때만 `exec-path`와 `PATH`의
맨 앞에 한 번 추가한다. 설치가 없거나 링크가 잘못되면 시스템 PATH를 변경하지
않고 해당 언어 설정만 기존 fallback 동작을 유지한다.

## ACP 어댑터 (claude-agent-acp, codex-acp)

agent-shell(`modules/development/31-agent.el`)이 Claude Code와 Codex에 붙을 때 쓰는
ACP 어댑터도 툴체인 구성요소다. kind는 `acp-adapter`이고, 동봉 Node로 실행된다.

- 입력: `toolchains/acp/<name>/package.json`과 `package-lock.json`을 커밋한다.
  lockfile은 사용자 `~/.npmrc`(`min-release-age` 포함)를 그대로 둔 채
  `npm install --package-lock-only --save-exact --omit=optional --ignore-scripts`로
  만든다. `toolchains.json`의 `revision`은 이 lockfile의 SHA-256이며, fetch와
  setup은 둘이 다르면 npm을 실행하기 전에 실패한다.
- 빌드: 잠긴 Node 아티팩트 안의 npm으로 `npm ci --omit=optional --ignore-scripts
  --no-audit --no-fund --no-bin-links --install-strategy=hoisted`를 실행한다.
  플랫폼별 네이티브 바이너리(optional 의존성)는 받지 않는다. npm 캐시만 임시
  디렉터리로 분리하고 사용자 npmrc는 그대로 적용한다. 설치된 `node_modules`는
  정렬된 항목, mtime 0, uid/gid 0, 소유자 이름 없음, 0644/0755 모드, 이름·시각 없는
  gzip 헤더로 묶으므로 같은 입력이면 바이트가 같다(두 번 빌드해 SHA-256이 같음을
  확인했다).
- 설치: `lib/acp/<name>/`에 풀고 `bin/claude-agent-acp`, `bin/codex-acp` 런처를
  만든다. 런처는 `CLAUDE_CODE_EXECUTABLE` / `CODEX_PATH`가 비어 있으면 PATH에서
  `claude` / `codex`를 찾아 넣고, 없으면 비워 둔다(어댑터가 첫 세션에서 알려 준다).
- 점검: `make doctor`(또는 `M-x imoogi-doctor`)가 활성 bundle이 현재 lock으로 설치됐는지,
  런처가 있는지, ACP `initialize`에 5초 안에 답하는지 확인한다. `claude` / `codex`
  CLI가 없으면 실패가 아니라 `warn`으로 표시한다.

### claude-agent-acp는 아티팩트를 커밋하지 않는다

claude-agent-acp는 Anthropic의 `@anthropic-ai/claude-agent-sdk`("All rights
reserved")를 포함하므로 공개 저장소에 재배포하지 않는다. 이 구성요소는
`"artifact_policy": "local-build"`이며 `vendor/toolchains/acp/claude-agent-acp/`는
`.gitignore` 대상이다. 커밋되는 것은 `package.json`, `package-lock.json`, 어댑터
자체의 Apache-2.0 라이선스, 그리고 lock의 SHA-256뿐이다.

`setup`은 이 아티팩트가 없으면 커밋된 lockfile로 위와 같은 빌드를 실행한 뒤 lock의
크기·SHA-256과 비교한다. 이 빌드만 npm 레지스트리에 접속한다.

- **npm 설치가 끝나지 못하면(오프라인, 레지스트리 오류, npm 비정상 종료) 이 구성요소만
  건너뛴다.** setup은 구성요소 이름·원인·해결 방법을 담은 `warning:`을 출력하고,
  나머지 언어 서버와 다른 어댑터는 그대로 설치·활성화한 뒤 종료 코드 `0`으로 끝난다.
  건너뛴 구성요소에는 런처가 생기지 않고, `install.json`의 `skipped`에 기록된다.
  `make doctor`는 이 어댑터를 조치가 필요한 항목(`missing`)으로 보고하며 해결 방법을
  함께 보여 준다.
- **해결 방법:** 네트워크가 되는 곳에서 `make toolchain-setup`을 다시 실행하거나, 온라인
  머신에서 만든 `vendor/toolchains/acp/claude-agent-acp/0.99.1/darwin-arm64/bundle.tgz`를
  같은 경로에 복사한 뒤 `make toolchain-setup`을 실행한다. 파일이 있으면 setup은
  빌드하지 않고 SHA-256만 검증한다.
- 구성요소를 건너뛴 bundle은 그 구성요소를 여전히 빌드할 수 없는 동안에만 재사용한다.
  빌드가 성공하면 setup은 완전한 bundle을 `.local/toolchains/<bundle>-<트리 SHA-256 앞
  12자리>/`에 따로 만든 뒤, 임시 symlink를 `.local/bin` 위로 rename해 한 번에 전환하고
  그다음에 이전 디렉터리를 지운다. 그래서 `.local/bin`은 어느 순간에도 이전 bundle이나
  새 bundle 중 하나를 온전히 가리키며, 중간에 끊긴 setup은 다시 실행하면 이어서
  마무리된다. `make doctor`는 `.local/bin`이 가리키는 디렉터리의 `install.json`을 읽는다.
- **다음은 여전히 종료 코드 `10`인 실패다:** 빌드했거나 미리 놓아 둔 아티팩트의 크기·
  SHA-256이 lock과 다를 때, 커밋된 `package-lock.json`의 SHA-256이 `revision`과 다를 때,
  그리고 커밋된(local-build가 아닌) 구성요소의 모든 실패. 이때는 아무것도 활성화하지
  않으며 lock을 고치지 않는다.

codex-acp와 그 의존성은 모두 허용 라이선스(Apache-2.0, MIT, ISC, BSD-3-Clause)라
아티팩트를 그대로 커밋한다. 그중 `@openai/codex` 0.165.0은 package.json에
Apache-2.0을 선언하지만 npm 배포물에 라이선스 파일이 없어,
`vendor/toolchains/licenses/codex-acp-NOTICE`에 그 사실을 적고 lock의
`license.notice`로 연결한다(전문은 `codex-acp-LICENSE`).

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
