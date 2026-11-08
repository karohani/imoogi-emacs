// Service-worker side of the Org save target: checks who is asking, validates
// the request, then talks to the native host over a long-lived port.
import browser from 'webextension-polyfill';
import { ORG_NATIVE_HOST, OrgSaveResult, validateOrgSaveRequest } from './org-clip';

export type OrgClientErrorCode =
	| 'forbidden_sender'
	| 'invalid_request'
	| 'host_not_installed'
	| 'host_disconnected'
	| 'ack_timeout'
	| 'save_timeout'
	| 'unsupported_browser';

export interface OrgClientFailure {
	type: 'clientError';
	code: OrgClientErrorCode;
	detail?: string;
	delivered: boolean;
}

export type OrgSaveResponse = OrgSaveResult | OrgClientFailure;

const ACK_TIMEOUT_MS = 15_000;
// Image downloads run inside the host with their own limits; this is the outer bound.
const RESULT_TIMEOUT_MS = 5 * 60_000;

/**
 * Only extension pages (popup, side panel, and the clipper iframe embedded in
 * reader mode) may trigger a save. A content script shares our extension id
 * but its sender.url is the web page's URL, so the URL must be our own origin.
 * sender.tab alone is not a signal: the embedded clipper iframe also has one.
 */
export function isTrustedOrgSender(sender: browser.Runtime.MessageSender): boolean {
	if (sender.id !== browser.runtime.id) return false;
	if (!sender.url) return false;
	return sender.url.startsWith(browser.runtime.getURL(''));
}

export async function handleOrgSave(request: unknown, sender: browser.Runtime.MessageSender): Promise<OrgSaveResponse> {
	if (!isTrustedOrgSender(sender)) {
		return { type: 'clientError', code: 'forbidden_sender', delivered: false };
	}
	const payload = (request as { payload?: unknown })?.payload;
	const problem = validateOrgSaveRequest(payload);
	if (problem) {
		return { type: 'clientError', code: 'invalid_request', detail: problem, delivered: false };
	}
	if (typeof browser.runtime.connectNative !== 'function') {
		return { type: 'clientError', code: 'unsupported_browser', delivered: false };
	}
	return sendToHost(payload as { requestId: string });
}

function sendToHost(payload: { requestId: string }): Promise<OrgSaveResponse> {
	return new Promise((resolve) => {
		let port: browser.Runtime.Port;
		let delivered = false;
		let settled = false;
		let timer: ReturnType<typeof setTimeout> | undefined;

		const finish = (response: OrgSaveResponse) => {
			if (settled) return;
			settled = true;
			if (timer) clearTimeout(timer);
			try { port.disconnect(); } catch { /* already closed */ }
			resolve(response);
		};
		const arm = (ms: number, code: OrgClientErrorCode) => {
			if (timer) clearTimeout(timer);
			timer = setTimeout(() => finish({ type: 'clientError', code, delivered }), ms);
		};

		try {
			port = browser.runtime.connectNative(ORG_NATIVE_HOST);
		} catch (error) {
			resolve({ type: 'clientError', code: 'host_not_installed', detail: String(error), delivered: false });
			return;
		}

		port.onMessage.addListener((message: unknown) => {
			if (typeof message !== 'object' || message === null) return;
			const m = message as { type?: string; requestId?: string };
			if (m.requestId !== undefined && m.requestId !== payload.requestId) return;
			if (m.type === 'ack') {
				delivered = true;
				arm(RESULT_TIMEOUT_MS, 'save_timeout');
			} else if (m.type === 'result') {
				finish(message as OrgSaveResult);
			}
		});
		port.onDisconnect.addListener((closed: browser.Runtime.Port) => {
			const chromeApi = (globalThis as { chrome?: { runtime?: { lastError?: { message?: string } } } }).chrome;
			const text = chromeApi?.runtime?.lastError?.message || closed?.error?.message || '';
			// Chrome reports a missing or unregistered host with this message.
			const code: OrgClientErrorCode = !delivered && /not found|forbidden|Specified native messaging host/i.test(text)
				? 'host_not_installed'
				: 'host_disconnected';
			finish({ type: 'clientError', code, detail: text, delivered });
		});

		arm(ACK_TIMEOUT_MS, 'ack_timeout');
		port.postMessage(payload);
	});
}
