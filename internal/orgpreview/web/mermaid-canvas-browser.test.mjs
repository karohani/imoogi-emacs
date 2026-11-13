// Real-browser check for the zoom and pan canvas around Mermaid diagrams.
// Run by hand: BINARY=<built imoogi-org-preview> node mermaid-canvas-browser.test.mjs
const {chromium} = await import(process.env.PLAYWRIGHT_MODULE || '/Users/jay/.npm/_npx/9833c18b2d85bc59/node_modules/playwright/index.mjs');
import {spawn} from 'node:child_process';
import {mkdtemp, rm} from 'node:fs/promises';
import {once} from 'node:events';
import assert from 'node:assert/strict';

const fixture = await mkdtemp('/private/tmp/canvas-fixture-');
const server = spawn(process.env.BINARY || '/private/tmp/imoogi-canvas-check', ['-token', 'canvas-test-token', '-print-bootstrap-json'], {stdio: ['ignore', 'pipe', 'pipe']});
let browser;
try {
  const [first] = await once(server.stdout, 'data');
  const base = 'http://' + JSON.parse(first.toString().trim()).addr;
  const headers = {'Content-Type': 'application/json', 'X-Org-Preview-Token': 'canvas-test-token'};
  // Six rows of twelve steps: wide and tall, like a real process diagram.
  const chain = Array.from({length: 6}, (_, r) => Array.from({length: 12}, (_, i) => 'R' + r + 'N' + i + '[단계 ' + r + '-' + i + ' 처리]').join(' --> ')).join('\n');
  const publish = async (rev, intro, extra = '') => {
    const r = await fetch(base + '/api/emacs/revisions', {method: 'POST', headers, body: JSON.stringify({
      version: 'org-preview/v1', session_id: 'emacs', buffer_id: 'a', revision: rev, event_id: 'rev-' + rev, origin: 'emacs',
      generation: 1, path: fixture + '/a.org', syntax: 'org', allowed_roots: [fixture],
      text: '* 큰 다이어그램\n\n' + intro + '\n\n#+begin_src mermaid\nflowchart LR\n' + chain + extra + '\n#+end_src\n\n' + after})});
    assert.equal(r.status, 202);
    return (await r.json()).browser_view;
  };
  // Headings and text after the diagram must not show through the large view.
  const after = Array.from({length: 8}, (_, i) => '* 뒤 제목 ' + i + '\n\n뒤 문단 ' + i + ' ' + '내용 '.repeat(40)).join('\n\n');
  const view = await publish(1, '처음 내용');
  browser = await chromium.launch({headless: true, executablePath: process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  const page = await browser.newPage({viewport: {width: 1100, height: 900}});
  const errors = [];
  page.on('pageerror', e => errors.push(e.message));
  // Count navigation messages the tab would send to Emacs.
  await page.addInitScript(() => {
    window.__nav = 0;
    const send = WebSocket.prototype.send;
    WebSocket.prototype.send = function (data) { if (String(data).includes('"element_id"')) window.__nav++; return send.call(this, data); };
  });
  await page.goto(base + '/preview?session=emacs&buffer=a&generation=1&view=' + view);
  const viewport = page.locator('.mermaid-viewport');
  await viewport.waitFor();
  const zoom = page.locator('.mermaid-zoom');
  const percent = async () => Number((await zoom.textContent()).replace('%', ''));
  const transform = () => page.locator('.mermaid-stage').evaluate(el => el.style.transform);

  // Fit to width: a long LR chart is shrunk, and the whole chart fits the box.
  const fitted = await percent();
  assert.ok(fitted < 100, 'wide chart starts fitted, got ' + fitted + '%');
  const box = await viewport.boundingBox();
  const stage = await page.locator('.mermaid-stage svg').boundingBox();
  assert.ok(stage.width <= box.width + 1, 'fitted chart is not wider than its box');
  console.log('FIT', fitted + '%', 'box', Math.round(box.width) + 'x' + Math.round(box.height));

  // Buttons zoom in around the center.
  await page.getByRole('button', {name: '확대', exact: true}).click();
  await page.getByRole('button', {name: '확대', exact: true}).click();
  const zoomed = await percent();
  assert.ok(zoomed > fitted * 1.4, 'zoomed in to ' + zoomed + '%');
  await page.getByRole('button', {name: '원래 크기', exact: true}).click();
  assert.equal(await percent(), 100, '1:1 shows the natural size');
  const grown = await viewport.boundingBox();
  assert.ok(grown.height > box.height * 1.5, 'zooming in grows the box: ' + box.height + ' -> ' + grown.height);
  assert.ok(grown.height <= 900 * 0.7 + 1, 'the box stays within 70% of the screen');

  // Dragging pans and does not move Emacs.
  const navBefore = await page.evaluate(() => window.__nav);
  const before = await transform();
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width / 2 - 200, box.y + box.height / 2, {steps: 8});
  await page.mouse.up();
  assert.notEqual(await transform(), before, 'drag pans the chart');
  await page.waitForTimeout(100);
  assert.equal(await page.evaluate(() => window.__nav), navBefore, 'a drag must not send navigation');
  console.log('DRAG', before, '->', await transform());

  // Ctrl+wheel zooms; a plain wheel scrolls the page instead.
  const atWheel = await percent();
  await page.mouse.move(box.x + 100, box.y + box.height / 2);
  await page.keyboard.down('Control');
  await page.mouse.wheel(0, -200);
  await page.keyboard.up('Control');
  await page.waitForTimeout(100);
  assert.ok(await percent() > atWheel, 'ctrl+wheel zooms in');
  const zoomAfterWheel = await percent();
  await page.mouse.wheel(0, 200);
  await page.waitForTimeout(100);
  assert.equal(await percent(), zoomAfterWheel, 'plain wheel does not zoom');

  // An Emacs edit re-renders the chart but keeps the zoom.
  const kept = await percent();
  // The diagram text itself changes too, which changes its block id.
  await publish(2, '고친 내용', '\nR0N11 --> N99[새 단계]');
  await page.locator('#root p', {hasText: '고친 내용'}).waitFor();
  await page.waitForFunction(() => document.querySelector('.mermaid-stage svg')?.textContent.includes('새 단계'));
  assert.equal(await percent(), kept, 'zoom survives a revision');
  console.log('REVISION kept', kept + '%');

  // Fit returns to the column width.
  await page.getByRole('button', {name: '폭에 맞춤', exact: true}).click();
  // The edited chart is a little wider, so compare shapes rather than the old percentage.
  const refit = await viewport.boundingBox(), chart = await page.locator('.mermaid-stage svg').boundingBox();
  assert.ok(Math.abs(chart.width - refit.width) <= 2, 'fit matches the column width');
  assert.ok(Math.abs(Math.max(60, chart.height) - refit.height) <= 2, 'fit restores the compact box');

  // A plain click still moves Emacs to the diagram source.
  await viewport.click({position: {x: 20, y: 20}});
  await page.waitForTimeout(100);
  assert.ok(await page.evaluate(() => window.__nav) > navBefore, 'a click still navigates');

  const shots = process.env.SHOTS;
  const shot = async name => { if (shots) await page.screenshot({path: shots + '/' + name + '.png'}); };
  await shot('1-document-fit');

  // Book view hides the canvas controls, shows the whole chart, and ignores canvas gestures.
  const docPercent = await percent();
  await page.getByRole('button', {name: '책자', exact: true}).click();
  await page.waitForTimeout(300);
  assert.equal(await page.locator('.mermaid-toolbar').first().isVisible(), false, 'book view hides the toolbar');
  await shot('2-book');
  const bookChart = page.locator('.mermaid-stage svg').first();
  if (await bookChart.isVisible()) {
    const b = await bookChart.boundingBox();
    await page.mouse.move(b.x + b.width / 2, b.y + b.height / 2);
    await page.keyboard.down('Control');
    await page.mouse.wheel(0, -200);
    await page.keyboard.up('Control');
  }
  await page.getByRole('button', {name: '발표', exact: true}).click();
  await page.waitForTimeout(300);
  await shot('3-present');
  await page.getByRole('button', {name: '문서', exact: true}).click();
  await page.locator('.mermaid-toolbar').first().waitFor({state: 'visible'});
  assert.equal(await percent(), docPercent, 'gestures in book view do not change the document zoom');

  // The large view fills the window and survives an Emacs edit. Real Chrome can paint a
  // z-index overlay under the page; boxing the content in its own stacking layer forces
  // that here, so only a top-layer large view passes.
  await page.addStyleTag({content: '#root{position:relative;z-index:0}'});
  await page.getByRole('button', {name: '크게 보기', exact: true}).first().click();
  await page.locator('.mermaid.mermaid-expanded').waitFor();
  assert.ok(await page.locator('.mermaid-expanded').evaluate(n => n.matches(':popover-open')), 'the large view is in the top layer');
  await page.waitForTimeout(200);
  const large = await viewport.boundingBox();
  // Emacs may mark the diagram block as active; its highlight must not see through the large view.
  await page.locator('.mermaid-expanded').evaluate(n => n.classList.add('active-block'));
  assert.equal(await page.locator('.mermaid-expanded').evaluate(n => getComputedStyle(n).backgroundColor), 'rgb(17, 21, 27)', 'the large view stays opaque');
  assert.ok(large.height > 600 && large.width > 1000, 'large view fills the window: ' + large.width + 'x' + large.height);
  const covered = await page.evaluate(() => [[10, 10], [150, 300], [550, 450], [700, 30], [900, 120], [1050, 850]].map(([x, y]) => {
    const el = document.elementFromPoint(x, y);
    return !!(el && el.closest('.mermaid-expanded'));
  }));
  assert.deepEqual(covered, [true, true, true, true, true, true], 'nothing on the page draws over the large view');
  console.log('LARGE', await percent() + '%', Math.round(large.width) + 'x' + Math.round(large.height));
  await shot('4-large');
  await publish(3, '크게 보는 중 고친 내용', '\nR5N11 --> N100[또 새 단계]');
  await page.waitForFunction(() => document.querySelector('.mermaid-stage svg')?.textContent.includes('또 새 단계'));
  assert.equal(await page.locator('.mermaid.mermaid-expanded').count(), 1, 'an Emacs edit keeps the large view open');
  assert.ok(await page.locator('.mermaid-expanded').evaluate(n => n.matches(':popover-open')), 'the re-rendered large view is back in the top layer');
  await page.keyboard.press('Escape');
  await page.waitForFunction(() => !document.querySelector('.mermaid-expanded'));
  await page.getByRole('button', {name: '크게 보기', exact: true}).first().click();
  await page.locator('.mermaid.mermaid-expanded').waitFor();
  await page.getByRole('button', {name: '크게 보기', exact: true}).first().click();
  await page.waitForFunction(() => !document.querySelector('.mermaid-expanded'));
  assert.ok((await viewport.boundingBox()).height < 600, 'closing returns to the column');
  assert.deepEqual(errors, []);
  console.log('errors', errors);
} finally {
  await browser?.close();
  server.kill('SIGINT');
  await once(server, 'exit');
  await rm(fixture, {recursive: true, force: true});
}
