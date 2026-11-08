import { describe, test, expect, beforeEach, afterEach } from 'vitest';
import browser from 'webextension-polyfill';
import { handleOrgSave, isTrustedOrgSender } from './org-native';
import { buildOrgSaveRequest } from './org-clip';

const runtime = browser.runtime as any;
const ID = '8c3f0f8e-1f8a-4c55-9a59-0e5a3f8f9d21';

function validPayload() {
	// No DOM in this environment: an empty body still yields a valid request.
	return {
		schemaVersion: 1, type: 'saveClip', requestId: ID, category: 'clippings',
		title: 'T', sourceUrl: 'https://example.com/', clippedAt: '2026-11-08T10:00:00+09:00',
		downloadAssets: false, blocks: [{ t: 'para', inl: [{ t: 'text', v: 'hi' }] }], assets: [],
	};
}

const popupSender = { id: 'mock-id', url: 'chrome-extension://mock-id/popup.html' };

class FakePort {
	posted: unknown[] = [];
	private messageListeners: ((m: unknown) => void)[] = [];
	private disconnectListeners: ((p: unknown) => void)[] = [];
	error?: { message: string };
	onMessage = { addListener: (fn: (m: unknown) => void) => this.messageListeners.push(fn) };
	onDisconnect = { addListener: (fn: (p: unknown) => void) => this.disconnectListeners.push(fn) };
	constructor(private script: (port: FakePort, msg: unknown) => void) {}
	postMessage(msg: unknown) { this.posted.push(msg); this.script(this, msg); }
	disconnect() {}
	emit(m: unknown) { this.messageListeners.forEach((fn) => fn(m)); }
	drop(message: string) { this.error = { message }; this.disconnectListeners.forEach((fn) => fn(this)); }
}

describe('isTrustedOrgSender', () => {
	beforeEach(() => { runtime.id = 'mock-id'; });
	test('accepts the popup page', () => {
		expect(isTrustedOrgSender(popupSender as any)).toBe(true);
	});
	test('accepts the clipper iframe embedded in a page (reader mode)', () => {
		expect(isTrustedOrgSender({ id: 'mock-id', url: 'chrome-extension://mock-id/side-panel.html?context=iframe', tab: { id: 1 }, frameId: 3 } as any)).toBe(true);
	});
	test('rejects content scripts (page url, tab present)', () => {
		expect(isTrustedOrgSender({ id: 'mock-id', url: 'https://example.com/article', tab: { id: 1 }, frameId: 0 } as any)).toBe(false);
	});
	test('rejects a sender without a url', () => {
		expect(isTrustedOrgSender({ id: 'mock-id' } as any)).toBe(false);
	});
	test('rejects a lookalike origin prefix', () => {
		expect(isTrustedOrgSender({ id: 'mock-id', url: 'chrome-extension://mock-idx/popup.html' } as any)).toBe(false);
	});
	test('rejects other extensions', () => {
		expect(isTrustedOrgSender({ id: 'other', url: 'chrome-extension://other/x.html' } as any)).toBe(false);
	});
	test('rejects a web page url even with our id', () => {
		expect(isTrustedOrgSender({ id: 'mock-id', url: 'https://evil.example/' } as any)).toBe(false);
	});
});

describe('handleOrgSave', () => {
	beforeEach(() => { runtime.id = 'mock-id'; });
	afterEach(() => { delete runtime.connectNative; });

	test('refuses an untrusted sender without touching the native port', async () => {
		let connected = false;
		runtime.connectNative = () => { connected = true; };
		const res = await handleOrgSave({ action: 'saveOrg', payload: validPayload() }, { id: 'mock-id', url: 'https://evil.example/', tab: { id: 3 } } as any);
		expect(res).toMatchObject({ type: 'clientError', code: 'forbidden_sender' });
		expect(connected).toBe(false);
	});

	test('refuses an invalid payload without touching the native port', async () => {
		let connected = false;
		runtime.connectNative = () => { connected = true; };
		const payload = { ...validPayload(), category: '../../x' };
		const res = await handleOrgSave({ action: 'saveOrg', payload }, popupSender as any);
		expect(res).toMatchObject({ type: 'clientError', code: 'invalid_request' });
		expect(connected).toBe(false);
	});

	test('forwards a valid request and returns the host result after ack', async () => {
		const port = new FakePort((p, msg) => {
			p.emit({ type: 'ack', requestId: ID });
			p.emit({ type: 'result', requestId: 'someone-else', status: 'failed' }); // ignored
			p.emit({ type: 'result', requestId: ID, status: 'saved', path: '00. Inbox/04. Clippings/x/index.org' });
		});
		runtime.connectNative = (name: string) => { expect(name).toBe('com.karohani.org_clipper'); return port; };
		const res = await handleOrgSave({ action: 'saveOrg', payload: validPayload() }, popupSender as any);
		expect(res).toMatchObject({ type: 'result', status: 'saved' });
		expect(port.posted).toHaveLength(1);
		expect(port.posted[0]).toEqual(validPayload());
	});

	test('reports a missing host as not installed', async () => {
		const port = new FakePort((p) => p.drop('Specified native messaging host not found.'));
		runtime.connectNative = () => port;
		const res = await handleOrgSave({ action: 'saveOrg', payload: validPayload() }, popupSender as any);
		expect(res).toMatchObject({ type: 'clientError', code: 'host_not_installed', delivered: false });
	});

	test('distinguishes a disconnect after delivery', async () => {
		const port = new FakePort((p) => { p.emit({ type: 'ack', requestId: ID }); p.drop('Native host has exited.'); });
		runtime.connectNative = () => port;
		const res = await handleOrgSave({ action: 'saveOrg', payload: validPayload() }, popupSender as any);
		expect(res).toMatchObject({ type: 'clientError', code: 'host_disconnected', delivered: true });
	});

	test('reports unsupported browsers', async () => {
		const res = await handleOrgSave({ action: 'saveOrg', payload: validPayload() }, popupSender as any);
		expect(res).toMatchObject({ type: 'clientError', code: 'unsupported_browser' });
	});
});

// Keeps the builder import exercised in a non-DOM environment for type coverage.
void buildOrgSaveRequest;
