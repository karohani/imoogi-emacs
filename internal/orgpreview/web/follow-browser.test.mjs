// Real-browser check for tabs that follow the selected Emacs buffer.
// Run by hand: BINARY=<built imoogi-org-preview> node follow-browser.test.mjs
const {chromium} = await import(process.env.PLAYWRIGHT_MODULE || '/Users/jay/.npm/_npx/9833c18b2d85bc59/node_modules/playwright/index.mjs');
import {spawn} from 'node:child_process';
import {mkdtemp, rm} from 'node:fs/promises';
import {once} from 'node:events';
import assert from 'node:assert/strict';

const fixture = await mkdtemp('/private/tmp/follow-fixture-');
const server = spawn(process.env.BINARY || '/private/tmp/imoogi-reader-check', ['-token', 'follow-test-token', '-print-bootstrap-json'], {stdio: ['ignore', 'pipe', 'pipe']});
let browser;
try {
  const [first] = await once(server.stdout, 'data');
  const base = 'http://' + JSON.parse(first.toString().trim()).addr;
  const headers = {'Content-Type': 'application/json', 'X-Org-Preview-Token': 'follow-test-token'};
  const envelope = (buffer, id) => ({version: 'org-preview/v1', session_id: 'emacs', buffer_id: buffer, revision: 1, event_id: id, origin: 'emacs'});
  const publish = async (buffer, title) => {
    const r = await fetch(base + '/api/emacs/revisions', {method: 'POST', headers, body: JSON.stringify({
      ...envelope(buffer, 'rev-' + buffer), generation: 1, path: fixture + '/' + buffer + '.org', syntax: 'org',
      text: '* ' + title + '\n\n' + Array.from({length: 40}, (_, i) => '문단 ' + i + ' ' + '내용 '.repeat(30)).join('\n\n'),
      allowed_roots: [fixture]})});
    assert.equal(r.status, 202);
    return (await r.json()).browser_view;
  };
  const current = async buffer => {
    const r = await fetch(base + '/api/emacs/current', {method: 'POST', headers, body: JSON.stringify(envelope(buffer, 'current-' + buffer + Date.now()))});
    assert.equal(r.status, 200);
  };
  const viewA = await publish('a', 'Alpha');
  await publish('b', 'Beta');
  const url = (view, follow) => base + '/preview?session=emacs&buffer=a&generation=1&view=' + view + (follow ? '&follow=1' : '');

  browser = await chromium.launch({headless: true, executablePath: process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  const errors = [];
  const followTab = await browser.newPage();
  const plainTab = await browser.newPage();
  for (const page of [followTab, plainTab]) page.on('pageerror', e => errors.push(e.message));
  await followTab.goto(url(viewA, true));
  await plainTab.goto(url(viewA, false));
  for (const page of [followTab, plainTab]) await page.getByText('Alpha').first().waitFor();

  // Document mode: the follow tab moves to b and keeps following.
  await current('b');
  await followTab.waitForURL(u => new URL(u).searchParams.get('buffer') === 'b');
  assert.equal(new URL(followTab.url()).searchParams.get('follow'), '1');
  await followTab.getByText('Beta').first().waitFor();
  await plainTab.waitForTimeout(400);
  assert.equal(new URL(plainTab.url()).searchParams.get('buffer'), 'a', 'a plain tab must stay on its file');
  console.log('DOCUMENT follow ->', new URL(followTab.url()).searchParams.get('buffer'));

  // Book mode: the page stays and offers the move instead.
  await followTab.getByRole('button', {name: '책자', exact: true}).click();
  await followTab.waitForTimeout(200);
  await current('a');
  const offer = followTab.getByRole('button', {name: 'Emacs 문서로 이동', exact: true});
  await offer.waitFor({state: 'visible'});
  assert.equal(new URL(followTab.url()).searchParams.get('buffer'), 'b', 'reading must not be pulled away');
  assert.match(await followTab.locator('.reader-status').textContent(), /Emacs에서 다른 문서를 보고 있음 · a\.org/);
  console.log('BOOK offer shown, still on', new URL(followTab.url()).searchParams.get('buffer'));
  await offer.click();
  await followTab.waitForURL(u => new URL(u).searchParams.get('buffer') === 'a');
  await followTab.getByText('Alpha').first().waitFor();
  assert.equal(new URL(followTab.url()).searchParams.get('follow'), '1');
  console.log('OFFER click ->', new URL(followTab.url()).searchParams.get('buffer'));

  // A tab opened on an older buffer learns the current one as it connects.
  await current('b');
  await followTab.waitForURL(u => new URL(u).searchParams.get('buffer') === 'b');
  const late = await browser.newPage();
  late.on('pageerror', e => errors.push(e.message));
  await late.goto(url(viewA, true));
  await late.waitForURL(u => new URL(u).searchParams.get('buffer') === 'b');
  console.log('LATE tab ->', new URL(late.url()).searchParams.get('buffer'));
  assert.deepEqual(errors, []);
  console.log('errors', errors);
} finally {
  await browser?.close();
  server.kill('SIGINT');
  await once(server, 'exit');
  await rm(fixture, {recursive: true, force: true});
}
