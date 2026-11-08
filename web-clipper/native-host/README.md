# Org 저장 도우미 (org-clipper-host)

Chrome에서 보고 있는 웹페이지의 본문과 이미지를 로컬 폴더에 Org 문서로 저장하는 기능입니다. 확장 프로그램이 정리한 본문을 Native Messaging으로 로컬 도우미에 넘기고, 도우미가 파일을 씁니다. Emacs 명령이나 `org-protocol`이 없어도 저장할 수 있고, 저장된 `.org` 파일은 Emacs에서 열어 사용합니다.

기존 "Add to Obsidian" 등의 저장 방식은 그대로 유지되며, 저장 대상에 **Org로 저장**이 추가됩니다.

현재 지원 환경은 macOS의 Chrome입니다.

## 구성

```
Chrome 확장 (팝업/사이드 패널)
   │  본문 HTML → 검증된 블록 구조 + 이미지 목록
   ▼
서비스 워커 ── 송신자·요청 검증
   │  runtime.connectNative("com.karohani.org_clipper")
   ▼
org-clipper-host (Go) ── 요청 재검증 → 이미지 다운로드 → Org 변환 → 저장
   │
   ▼
<저장 루트>/00. Inbox/<분류>/<날짜>-<제목>.org (+ 이미지는 assets/<날짜>-<제목>/)
```

- 확장은 전체 파일 경로가 아니라 **분류 ID**만 보냅니다. 저장 루트는 도우미 설정 파일에만 있으며, 웹페이지나 저장 요청으로는 바꿀 수 없습니다.
- 도우미는 로컬 HTTP 서버를 띄우지 않으며, 셸 명령 실행·Lisp 평가·임의 파일 읽기 기능을 제공하지 않습니다.
- 메시지 형식은 [PROTOCOL.md](PROTOCOL.md)에 정의되어 있습니다.

## 준비물

- Google Chrome
- Node.js와 npm (확장 빌드)
- Go 1.26 이상 (도우미 빌드)

## 1. 확장 빌드와 등록

`web-clipper/` 폴더에서 실행합니다.

```bash
npm install
npm run build:chrome      # 배포용 빌드 → dist/
# 또는 개발 중이라면
npm run dev:chrome        # 변경 감시 빌드 → dev/
```

1. Chrome에서 `chrome://extensions`를 열고 **개발자 모드**를 켭니다.
2. **압축해제된 확장 프로그램을 로드합니다**를 눌러 `dist/`(또는 `dev/`) 폴더를 선택합니다.
3. 확장 카드에 표시된 ID가 `ppclibiagilamlklkoapodlammmkdale`인지 확인합니다.

`src/manifest.chrome.json`에 고정 공개키(`key`)가 들어 있어서, 폴더 위치를 옮기거나 다시 로드해도 확장 ID가 바뀌지 않습니다. 도우미는 이 ID만 허용합니다.

### 다른 확장 ID를 쓰는 경우

`key`를 지웠거나 다른 키로 바꾼 개발용 빌드는 ID가 달라집니다. 이때는 `chrome://extensions`에서 실제 ID를 확인한 뒤, 아래 3단계의 `install`을 그 ID로 다시 실행합니다. 여러 ID를 함께 허용하려면 `--extension-id`를 반복해서 적습니다.

```bash
bin/org-clipper-host install \
  --extension-id ppclibiagilamlklkoapodlammmkdale \
  --extension-id <개발용-확장-ID>
```

## 2. 도우미 빌드

`web-clipper/native-host/` 폴더에서 실행합니다.

```bash
make build        # bin/org-clipper-host 생성
make test         # 테스트 실행 (선택)
```

## 3. 저장 루트 설정과 도우미 등록

저장 루트는 실제 절대경로로 지정합니다. 폴더 이름만으로 경로를 추측하지 않으며, 폴더는 미리 만들어 두어야 합니다.

```bash
mkdir -p "$HOME/satelite"
bin/org-clipper-host setup --root "$HOME/satelite"
bin/org-clipper-host install --extension-id ppclibiagilamlklkoapodlammmkdale
bin/org-clipper-host status
```

`install`은 다음 두 가지를 합니다.

| 생성 파일 | 용도 |
|-----------|------|
| `~/Library/Application Support/org-clipper/bin/org-clipper-host` | Chrome이 실행하는 도우미 사본 |
| `~/Library/Application Support/Google/Chrome/NativeMessagingHosts/com.karohani.org_clipper.json` | Chrome에 도우미를 알리는 등록 파일. `allowed_origins`에는 지정한 확장 ID만 들어갑니다. |

설정은 `~/Library/Application Support/org-clipper/config.json`(권한 `0600`)에 저장됩니다. 관리자 권한은 필요 없습니다.

`status` 출력에서 저장 루트, 등록 파일, 허용된 확장 ID를 확인할 수 있습니다. 등록 후에는 Chrome에서 확장을 한 번 새로고침합니다.

## 4. 사용 방법

1. 확장 **설정 → 일반 → 저장 동작(Save behavior)**에서 **Org로 저장**을 고르면 기본 버튼이 Org 저장으로 바뀝니다. 다른 저장 동작을 쓰는 중이라면 버튼 옆 ▾ 메뉴에서 **Org로 저장**을 고릅니다. 그러면 분류 드롭다운이 나타나고 기본 버튼이 **Org로 저장**으로 바뀝니다. 분류를 고른 뒤 버튼을 누르면 저장됩니다.
2. 버튼 위에 나타나는 드롭다운에서 분류를 고릅니다. 처음에는 **04. Clippings**가 선택되며, 마지막으로 고른 분류를 기억합니다.
3. 이미지 없이 본문만 저장하려면 **이미지 저장**의 체크를 해제합니다. 체크하면 도우미가 이미지를 내려받기 위해 추가 네트워크 요청을 보냅니다.
4. 저장 결과는 팝업에 표시되며 팝업은 자동으로 닫히지 않습니다.

| 분류 | 저장 폴더 |
|------|-----------|
| articles | `00. Inbox/01. Articles` |
| papers | `00. Inbox/02. Papers` |
| transcripts | `00. Inbox/03. Transcripts` |
| clippings | `00. Inbox/04. Clippings` |
| ai-research | `00. Inbox/05. AI Research` |

분류 폴더가 없으면 저장 루트 안에 자동으로 만듭니다.

### 저장 결과 구조

```
~/satelite/00. Inbox/04. Clippings/
  2026-11-08-원문-제목.org
  2026-11-08-다른-글.org
  assets/
    2026-11-08-원문-제목/
      908fc41c8132064983e5f14b071d882e.png
```

- 파일 이름은 날짜와 정리된 제목(최대 60자, 한글 유지)으로 만듭니다. 같은 이름이 이미 있으면 `-2`, `-3`을 붙입니다. 원래 제목은 문서 안 `#+TITLE:`에 그대로 남습니다.
- 이미지가 있는 클립만 `assets/<파일 이름>/` 폴더가 생깁니다. 이미지는 내용 해시로 이름을 붙이고 `[[file:assets/<파일 이름>/…]]` 상대경로로 연결하므로, 분류 폴더를 통째로 옮겨도 연결이 유지됩니다.
- 기존 파일은 덮어쓰지 않습니다.

### 결과 메시지

| 표시 | 의미 |
|------|------|
| Org로 저장했습니다 | 문서와 요청한 이미지를 모두 저장함 |
| 이미지 N개를 저장하지 못했습니다 | 문서는 저장됨. 실패한 이미지는 문서에 링크 없이 `(image not saved)`로 표시됨 |
| 이미 Org로 저장된 클립입니다 | 같은 요청을 다시 보냄. 새 폴더를 만들지 않고 이전 결과를 보여줌 |
| 도우미가 설치되지 않았거나 … 등록되지 않았습니다 | 3단계를 확인하세요 (`status`) |
| 요청은 도우미에 전달됐지만 완료 응답을 받지 못했습니다 | 도우미가 저장을 마쳤을 수 있습니다. 버튼을 다시 누르면 같은 요청 ID로 결과를 확인합니다 |
| 저장 폴더가 설정되지 않았습니다 / 없거나 사용할 수 없습니다 | `setup --root`를 다시 실행하세요 |
| 쓸 권한이 없습니다 / 디스크 공간이 부족해 … | 저장 루트의 권한이나 남은 공간을 확인하세요 |

실패해도 작성한 내용은 팝업에 남아 있으며, 버튼을 다시 눌러 재시도할 수 있습니다. 같은 내용으로 재시도하면 같은 요청 ID를 쓰므로 클립이 중복으로 생기지 않습니다.

## 5. 업데이트

```bash
# 확장
cd web-clipper && npm run build:chrome
# chrome://extensions 에서 확장 새로고침

# 도우미
cd web-clipper/native-host && make build
bin/org-clipper-host install --extension-id ppclibiagilamlklkoapodlammmkdale
```

`install`을 다시 실행해야 새 도우미가 설치 위치로 복사됩니다. 저장 루트 설정과 기존 클립은 그대로 유지됩니다.

## 6. 제거

```bash
bin/org-clipper-host uninstall                 # 등록 파일과 설치된 도우미 삭제
bin/org-clipper-host uninstall --purge-config  # 설정 파일(config.json)까지 삭제
```

저장된 클립과 로그는 삭제하지 않습니다. 필요하면 직접 지웁니다.

| 남는 항목 | 위치 |
|-----------|------|
| 클립 | 저장 루트 (`~/satelite`) |
| 도우미 내부 기록 | `<저장 루트>/.org-clipper/` (요청 기록, 임시 폴더) |
| 로그 | `~/Library/Logs/org-clipper/host.log` |

마지막으로 `chrome://extensions`에서 확장을 제거합니다.

## 보안 동작 요약

- **요청 검증**: 확장과 도우미 양쪽에서 필드의 타입·크기·필수값을 검사합니다. 도우미는 메시지 본문을 읽기 전에 길이(최대 8MiB)부터 확인합니다. 서비스 워커는 확장 자신의 페이지(팝업·사이드 패널)에서 온 요청만 받고, 웹페이지의 content script에서 온 요청은 거절합니다.
- **경로**: 분류 ID 허용 목록만 받습니다. 모든 파일 작업은 Go의 `os.Root`로 저장 루트 안에서만 수행하므로, 상위 경로 이동이나 심볼릭 링크로 루트 밖에 쓸 수 없습니다. 원문 파일명이나 `Content-Disposition`은 쓰지 않습니다.
- **저장 순서**: `<루트>/.org-clipper/tmp/`에 먼저 쓰고, 완성되면 최종 위치로 옮긴 뒤에 완료 응답을 보냅니다. 요청 ID 기록으로 재전송을 중복 없이 처리하고, 중단 후 다시 실행해도 결과를 확인할 수 있습니다. 하루 이상 지난 임시 폴더는 도우미가 만든 것만 정리합니다.
- **Org 변환**: 웹페이지 문자열은 문맥별로 이스케이프합니다. `#+` 지시문, 블록 종료, 헤드라인, 표 수식(`#+TBLFM`), 매크로, 인라인 Babel(`src_`, `call_`), 파일 지역 변수(`Local Variables:`, `-*-`)는 실행 가능한 문법으로 남지 않습니다. 코드 예제는 `src` 블록이 아니라 `example` 블록으로 저장합니다. 활성 링크는 `http(s)` 링크와 `assets/` 상대경로뿐이며, `shell:`·`elisp:`·`javascript:` 링크는 글자로만 남습니다.
- **이미지 다운로드**: PNG·JPEG·WebP만, HTTPS(443번 포트)만 받습니다. 형식은 확장자나 Content-Type이 아니라 실제 파일 시그니처로 확인합니다. 접속하는 순간의 IP를 검사해 localhost·사설망·링크 로컬·예약 주소(IPv4/IPv6)를 차단하므로 DNS 재바인딩과 리디렉션 우회도 막습니다. 리디렉션은 최대 3번이며 매번 다시 검사합니다. 쿠키와 인증 헤더는 보내지 않습니다. 한도는 이미지당 10MiB, 클립당 합계 50MiB, 최대 30개, 5천만 픽셀이며, 실제로 받은 바이트를 기준으로 셉니다. `data:`·`blob:`·`file:` 이미지는 내려받지 않습니다.
- **로그**: 요청 ID, 분류, 결과, 원문 URL의 호스트만 기록합니다. 제목, 본문, 전체 URL·쿼리, 헤더는 기록하지 않습니다.
- **권한**: 도우미가 만드는 폴더는 `0700`, 파일은 `0600`입니다.

### 알려진 한계

- Org 이스케이프에는 보이지 않는 문자(U+200B, U+2060)를 씁니다. 문서에서 글을 복사하면 이 문자가 함께 복사됩니다.
- 파일 지역 변수 방어는 현재 Emacs의 문자열 일치 방식을 전제로 합니다. Emacs의 `enable-local-variables`는 기본값(실행 전에 묻기)으로 두는 것을 권장합니다.
- 최종 위치로 옮기기 직전의 존재 확인과 이동 사이에는 아주 짧은 경쟁 구간이 있습니다. 요청 ID별 잠금이 같은 요청의 동시 처리를 막습니다.
- 파일 시그니처 검사는 이미지 파일의 안전을 완전히 보증하지는 않습니다.
