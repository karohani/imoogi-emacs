Obsidian Web Clipper helps you highlight and capture the web in your favorite browser. Anything you save is stored as durable Markdown files that you can read offline, and preserve for the long term.

- **[Download Web Clipper](https://obsidian.md/clipper)**
- **[Documentation](https://help.obsidian.md/web-clipper)**
- **[Troubleshooting](https://help.obsidian.md/web-clipper/troubleshoot)**

## Get started

Install the extension by downloading it from the official directory for your browser:

- **[Chrome Web Store](https://chromewebstore.google.com/detail/obsidian-web-clipper/cnjifjpddelmedmihgijeibhnjfabmlf)** for Chrome, Brave, Arc, Orion, and other Chromium-based browsers.
- **[Firefox Add-Ons](https://addons.mozilla.org/en-US/firefox/addon/web-clipper-obsidian/)** for Firefox and Firefox Mobile.
- **[Safari Extensions](https://apps.apple.com/us/app/obsidian-web-clipper/id6720708363)** for macOS, iOS, and iPadOS.
- **[Edge Add-Ons](https://microsoftedge.microsoft.com/addons/detail/obsidian-web-clipper/eigdjhmgnaaeaonimdklocfekkaanfme)** for Microsoft Edge.

## Use the extension

Documentation is available on the [Obsidian Help site](https://help.obsidian.md/web-clipper), which covers how to use [highlighting](https://help.obsidian.md/web-clipper/highlight), [templates](https://help.obsidian.md/web-clipper/templates), [variables](https://help.obsidian.md/web-clipper/variables), [filters](https://help.obsidian.md/web-clipper/filters), and more.

## Contribute

### Documentation

User documentation is maintained in the [`en/Obsidian Web Clipper` directory of obsidian-help](https://github.com/obsidianmd/obsidian-help/tree/master/en/Obsidian%20Web%20Clipper).

### Translations

You can help translate Web Clipper into your language. Submit your translation via pull request using the format found in the [/_locales](/src/_locales) folder.

### Features and bug fixes

See the [help wanted](https://github.com/obsidianmd/obsidian-clipper/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22) tag for issues where contributions are welcome.

## Roadmap

In no particular order:

- [ ] Annotate highlights
- [ ] Template directory
- [ ] Sync settings across browsers
- [x] A separate icon for Web Clipper (1.6.3)
- [x] Template validation (1.1.0)
- [x] Template logic (if/for)  (1.1.0)
- [x] Save images locally ([Obsidian 1.8.0](https://obsidian.md/changelog/2024-12-18-desktop-v1.8.0/))
- [x] Translate UI into more languages — help is welcomed

## Developers

To build the extension:

```
npm run build
```

This will create three directories:
- `dist/` for the Chromium version
- `dist_firefox/` for the Firefox version
- `dist_safari/` for the Safari version

### Install the extension locally

For Chromium browsers, such as Chrome, Brave, Edge, and Arc:

1. Open your browser and navigate to `chrome://extensions`
2. Enable **Developer mode**
3. Click **Load unpacked** and select the `dist` directory

#### Chrome에서 개발하며 사용하기

개발용 확장은 폴더를 한 번 등록하면 Chrome에 등록 상태가 유지됩니다. 코드를 수정할 때마다 확장을 삭제하거나 다시 등록할 필요 없이, 빌드 후 확장을 새로고침하면 됩니다.

1. 처음에는 프로젝트 폴더에서 의존성을 설치합니다.
   ```bash
   npm install
   ```
2. 개발용 빌드와 파일 변경 감시를 실행합니다.
   ```bash
   npm run dev:chrome
   ```
   첫 빌드가 완료되면 `dev/` 폴더가 생성됩니다. 명령을 실행해 둔 동안에는 소스 파일을 저장할 때마다 자동으로 다시 빌드됩니다.
3. Chrome에서 `chrome://extensions`를 열고 **개발자 모드**를 켭니다.
4. **압축해제된 확장 프로그램을 로드합니다**를 눌러 이 프로젝트의 `dev/` 폴더를 선택합니다. 이 등록은 처음 한 번만 하면 됩니다.
5. 이후 코드를 수정하면 빌드가 완료된 것을 확인한 뒤, `chrome://extensions`에서 해당 확장의 **새로고침** 버튼을 누릅니다. 자동 빌드가 확장 새로고침까지 수행하지는 않습니다.
6. 클리핑할 웹페이지도 새로고침한 뒤 확장을 다시 열어 변경 사항을 확인합니다.

개발을 종료할 때는 터미널에서 `Ctrl+C`로 변경 감시를 중단합니다. 다음에 개발할 때는 `npm run dev:chrome`을 다시 실행하면 됩니다. 등록한 `dev/` 폴더의 위치를 옮기거나 삭제했다면 다시 등록해야 합니다.

위의 배포용 빌드 방법으로 `dist/` 폴더를 등록한 경우에는 `npm run build:chrome`으로 다시 빌드한 뒤 확장을 새로고침하세요. 개발용 빌드의 출력 폴더는 `dev/`이므로, 변경 사항을 확인하려면 Chrome에 등록한 폴더와 빌드 출력 폴더가 일치해야 합니다.

#### Org 문서로 저장하기 (Chrome, macOS)

이 포크에는 웹페이지를 로컬 폴더에 Org 문서로 저장하는 **Org로 저장** 기능이 추가되어 있습니다. 로컬 도우미(`native-host/`)를 설치하고 등록해야 사용할 수 있습니다. 설치·등록·저장 루트 설정·업데이트·제거 방법은 [native-host/README.md](native-host/README.md)를 참고하세요.

For Firefox:

1. Open Firefox and navigate to `about:debugging#/runtime/this-firefox`
2. Click **Load Temporary Add-on**
3. Navigate to the `dist_firefox` directory and select the `manifest.json` file

If you want to run the extension permanently you can do so with the Nightly or Developer versions of Firefox.

1. Type `about:config` in the URL bar
2. In the Search box type `xpinstall.signatures.required`
3. Double-click the preference, or right-click and select "Toggle", to set it to `false`.
4. Go to `about:addons` > gear icon > **Install Add-on From File…**

For iOS Simulator testing on macOS:

1. Run `npm run build` to build the extension
2. Open `xcode/Obsidian Web Clipper/Obsidian Web Clipper.xcodeproj` in Xcode
3. Select the **Obsidian Web Clipper (iOS)** scheme from the scheme selector
4. Choose an iOS Simulator device and click **Run** to build and launch the app
5. Once the app is running on the simulator, open **Safari**
6. Navigate to a webpage and tap the **Extensions** button in Safari to access the Web Clipper extension

### Run tests

```
npm test
```

Or run in watch mode during development:

```
npm run test:watch
```

## Third-party libraries

- [webextension-polyfill](https://github.com/mozilla/webextension-polyfill) for browser compatibility
- [defuddle](https://github.com/kepano/defuddle) for content extraction and Markdown conversion
- [dayjs](https://github.com/iamkun/dayjs) for date parsing and formatting
- [lz-string](https://github.com/pieroxy/lz-string) to compress templates to reduce storage space
- [lucide](https://github.com/lucide-icons/lucide) for icons
- [dompurify](https://github.com/cure53/DOMPurify) for sanitizing HTML

## License

Obsidian Web Clipper source code is open source under the MIT License. All trademarks, icons, marketing copy, and other marketing assets are excluded from that license.
