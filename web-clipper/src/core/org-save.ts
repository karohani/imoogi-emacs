// Popup / side-panel UI for the Org save target.
import browser from '../utils/browser-polyfill';
import { getLocalStorage, setLocalStorage } from '../utils/storage-utils';
import { getMessage } from '../utils/i18n';
import {
	DEFAULT_ORG_CATEGORY,
	OrgCategory,
	buildOrgSaveRequest,
	isOrgCategory,
} from '../utils/org-clip';
import type { OrgSaveResponse } from '../utils/org-native';

const CATEGORY_KEY = 'orgLastCategory';
const ASSETS_KEY = 'orgDownloadAssets';

// The request id survives retries of the same draft so the host can dedupe a
// resend whose first answer was lost; any change to the draft gets a new id.
let draftId: string | null = null;
let draftFingerprint = '';
let busy = false;

export async function initializeOrgOptions(visible: boolean): Promise<void> {
	const container = document.getElementById('org-options');
	const select = document.getElementById('org-category-select') as HTMLSelectElement | null;
	const assets = document.getElementById('org-download-assets') as HTMLInputElement | null;
	if (!container || !select || !assets) return;

	const stored = await getLocalStorage(CATEGORY_KEY);
	select.value = isOrgCategory(stored) ? stored : DEFAULT_ORG_CATEGORY;
	const storedAssets = await getLocalStorage(ASSETS_KEY);
	assets.checked = storedAssets !== false;

	select.onchange = () => {
		if (isOrgCategory(select.value)) setLocalStorage(CATEGORY_KEY, select.value);
	};
	assets.onchange = () => setLocalStorage(ASSETS_KEY, assets.checked);
	setOrgOptionsVisible(visible);
}

export function setOrgOptionsVisible(visible: boolean): void {
	const container = document.getElementById('org-options');
	if (container) container.style.display = visible ? 'flex' : 'none';
}

function selectedCategory(): OrgCategory {
	const select = document.getElementById('org-category-select') as HTMLSelectElement | null;
	return select && isOrgCategory(select.value) ? select.value : DEFAULT_ORG_CATEGORY;
}

function downloadAssetsChecked(): boolean {
	const assets = document.getElementById('org-download-assets') as HTMLInputElement | null;
	return assets ? assets.checked : true;
}

type StatusKind = 'progress' | 'success' | 'warning' | 'error';

function showStatus(kind: StatusKind, lines: string[]): void {
	const box = document.getElementById('org-status');
	if (!box) return;
	box.className = `org-status org-status-${kind}`;
	box.textContent = '';
	for (const line of lines) {
		const p = document.createElement('div');
		p.textContent = line; // textContent only: host strings are never parsed as HTML
		box.appendChild(p);
	}
	box.style.display = 'block';
}

function fingerprint(parts: string[]): string {
	// Cheap content identity for retry reuse; collisions only cost a fresh id.
	let h = 0;
	for (const part of parts) {
		for (let i = 0; i < part.length; i++) h = (Math.imul(31, h) + part.charCodeAt(i)) | 0;
		h = (Math.imul(31, h) + 0x7f) | 0;
	}
	return `${parts.map((p) => p.length).join(':')}:${h}`;
}

export async function handleSaveOrg(variables: Record<string, string>, button: HTMLElement | null): Promise<void> {
	if (busy) return;
	setOrgOptionsVisible(true);

	const title = variables['{{title}}'] || '';
	const sourceUrl = variables['{{url}}'] || '';
	const contentHtml = variables['{{contentHtml}}'] || '';
	const category = selectedCategory();
	const downloadAssets = downloadAssetsChecked();

	const fp = fingerprint([title, sourceUrl, contentHtml, category, String(downloadAssets)]);
	if (!draftId || fp !== draftFingerprint) {
		draftId = crypto.randomUUID();
		draftFingerprint = fp;
	}

	let built;
	try {
		built = buildOrgSaveRequest({ requestId: draftId, category, title, sourceUrl, contentHtml, downloadAssets });
	} catch (error) {
		console.error('Org request build failed:', error);
		showStatus('error', [getMessage('orgErrorInvalid')]);
		return;
	}

	busy = true;
	button?.setAttribute('disabled', 'true');
	showStatus('progress', [getMessage('orgSaving')]);
	try {
		const response = await browser.runtime.sendMessage({ action: 'saveOrg', payload: built.request }) as OrgSaveResponse | undefined;
		renderResponse(response, built.unsupportedImages);
	} catch (error) {
		console.error('Org save message failed:', error);
		showStatus('error', [getMessage('orgErrorDisconnected'), getMessage('orgRetryHint')]);
	} finally {
		busy = false;
		button?.removeAttribute('disabled');
	}
}

function renderResponse(response: OrgSaveResponse | undefined, unsupportedImages: number): void {
	if (!response) {
		showStatus('error', [getMessage('orgErrorDisconnected'), getMessage('orgRetryHint')]);
		return;
	}
	if (response.type === 'clientError') {
		const key = {
			forbidden_sender: 'orgErrorInvalid',
			invalid_request: 'orgErrorInvalid',
			host_not_installed: 'orgErrorNotInstalled',
			host_disconnected: 'orgErrorDisconnected',
			ack_timeout: 'orgErrorTimeout',
			save_timeout: 'orgErrorTimeout',
			unsupported_browser: 'orgErrorUnsupported',
		}[response.code];
		const lines = [getMessage(key)];
		// "Delivered but no answer" differs from "never delivered": the host may
		// still finish, and a retry with the same id returns its stored result.
		if (response.delivered) lines.push(getMessage('orgDeliveredNoResult'));
		lines.push(getMessage('orgRetryHint'));
		showStatus('error', lines);
		return;
	}

	const extra: string[] = [];
	if (unsupportedImages > 0) extra.push(getMessage('orgImagesUnsupported', String(unsupportedImages)));
	const failed = response.failedAssets?.filter((f) => f.reason !== 'skipped') ?? [];

	switch (response.status) {
		case 'saved':
			showStatus(unsupportedImages > 0 ? 'warning' : 'success', [
				getMessage(response.duplicate ? 'orgAlreadySaved' : 'orgSaved'),
				response.path || '',
				...extra,
			]);
			break;
		case 'partial':
			showStatus('warning', [
				getMessage('orgSavedPartial', String(failed.length)),
				response.path || '',
				...failed.slice(0, 5).map((f) => `• ${f.id}: ${f.reason}`),
				...extra,
			]);
			break;
		case 'in_progress':
			showStatus('progress', [getMessage('orgInProgress'), getMessage('orgRetryHint')]);
			break;
		default: {
			const code = response.error?.code || 'internal';
			const key = ({
				not_configured: 'orgErrorNotConfigured',
				root_unavailable: 'orgErrorRootUnavailable',
				permission_denied: 'orgErrorPermission',
				disk_full: 'orgErrorDiskFull',
				too_large: 'orgErrorTooLarge',
				invalid_request: 'orgErrorInvalid',
				unknown_category: 'orgErrorInvalid',
			} as Record<string, string>)[code] || 'orgErrorInternal';
			showStatus('error', [getMessage(key), getMessage('orgRetryHint')]);
		}
	}
}
