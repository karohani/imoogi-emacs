// Org save target: builds and validates the native-host request.
// The wire format is defined in native-host/PROTOCOL.md (schemaVersion 1).
// The host re-validates everything; this side rejects early so a bad request
// never reaches the native port.

export const ORG_SCHEMA_VERSION = 1;
export const ORG_NATIVE_HOST = 'com.karohani.org_clipper';

export const ORG_CATEGORIES = ['articles', 'papers', 'transcripts', 'clippings', 'ai-research'] as const;
export type OrgCategory = typeof ORG_CATEGORIES[number];
export const DEFAULT_ORG_CATEGORY: OrgCategory = 'clippings';

export const ORG_LIMITS = {
	messageBytes: 8 * 1024 * 1024,
	titleChars: 1000,
	urlChars: 4096,
	textBytes: 4 * 1024 * 1024,
	nodes: 50_000,
	depth: 12,
	assets: 30,
	htmlChars: 6 * 1024 * 1024,
};

export type OrgInline =
	| { t: 'text'; v: string }
	| { t: 'bold' | 'italic' | 'strike'; c: OrgInline[] }
	| { t: 'code'; v: string }
	| { t: 'link'; href: string; c: OrgInline[] }
	| { t: 'image'; asset: string }
	| { t: 'br' };

export type OrgBlock =
	| { t: 'heading'; level: number; inl: OrgInline[] }
	| { t: 'para'; inl: OrgInline[] }
	| { t: 'list'; ordered: boolean; items: OrgBlock[][] }
	| { t: 'quote'; blocks: OrgBlock[] }
	| { t: 'table'; header: boolean; rows: OrgInline[][][] }
	| { t: 'code'; text: string }
	| { t: 'image'; asset: string }
	| { t: 'hr' };

export interface OrgAsset { id: string; url: string; }

export interface OrgSaveRequest {
	schemaVersion: number;
	type: 'saveClip';
	requestId: string;
	category: OrgCategory;
	title: string;
	sourceUrl: string;
	clippedAt: string;
	downloadAssets: boolean;
	blocks: OrgBlock[];
	assets: OrgAsset[];
}

export interface OrgSaveResult {
	type: 'result';
	requestId: string;
	status: 'saved' | 'partial' | 'failed' | 'in_progress' | 'unknown';
	path?: string;
	assetsSaved?: number;
	failedAssets?: { id: string; reason: string }[];
	duplicate?: boolean;
	error?: { code: string; message?: string };
}

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const ASSET_ID_RE = /^a[0-9]{1,3}$/;

const SKIP_TAGS = new Set([
	'script', 'style', 'noscript', 'template', 'iframe', 'frame', 'frameset', 'object', 'embed',
	'applet', 'svg', 'math', 'canvas', 'video', 'audio', 'source', 'track', 'form', 'input',
	'button', 'select', 'textarea', 'link', 'meta', 'head', 'title', 'base', 'dialog', 'portal',
]);
const BLOCK_TAGS = new Set([
	'p', 'div', 'section', 'article', 'main', 'header', 'footer', 'aside', 'nav', 'figure',
	'figcaption', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'ul', 'ol', 'li', 'blockquote', 'pre',
	'table', 'thead', 'tbody', 'tfoot', 'tr', 'td', 'th', 'hr', 'dl', 'dt', 'dd', 'details', 'summary',
]);

class BuildLimitError extends Error {}

interface BuildState {
	baseUrl: string;
	assets: OrgAsset[];
	assetByUrl: Map<string, string>;
	unsupportedImages: number;
	nodes: number;
	textBytes: number;
}

/** Converts extracted article HTML into the strict block AST plus the image asset list. */
export function htmlToOrgBlocks(html: string, baseUrl: string): { blocks: OrgBlock[]; assets: OrgAsset[]; unsupportedImages: number } {
	if (html.length > ORG_LIMITS.htmlChars) {
		throw new BuildLimitError('content too large');
	}
	const doc = new DOMParser().parseFromString(html, 'text/html');
	const state: BuildState = { baseUrl, assets: [], assetByUrl: new Map(), unsupportedImages: 0, nodes: 0, textBytes: 0 };
	const blocks = convertChildren(doc.body, state, 0);
	return { blocks, assets: state.assets, unsupportedImages: state.unsupportedImages };
}

function countNode(state: BuildState, text = ''): void {
	state.nodes++;
	state.textBytes += text.length * 3; // worst-case UTF-8 bytes per UTF-16 unit
	if (state.nodes > ORG_LIMITS.nodes || state.textBytes > ORG_LIMITS.textBytes) {
		throw new BuildLimitError('content too large');
	}
}

function tagOf(node: Node): string {
	return node.nodeType === Node.ELEMENT_NODE ? (node as Element).tagName.toLowerCase() : '';
}

// Walks a container, grouping consecutive inline content into paragraphs.
function convertChildren(parent: Node, state: BuildState, depth: number): OrgBlock[] {
	const blocks: OrgBlock[] = [];
	let pending: Node[] = [];
	const flush = () => {
		if (pending.length === 0) return;
		const inl = trimInlines(convertInlines(pending, state, depth));
		pending = [];
		if (inl.length > 0) {
			countNode(state);
			blocks.push({ t: 'para', inl });
		}
	};
	parent.childNodes.forEach((child) => {
		const tag = tagOf(child);
		if (child.nodeType === Node.ELEMENT_NODE && (SKIP_TAGS.has(tag))) return;
		if (child.nodeType === Node.ELEMENT_NODE && BLOCK_TAGS.has(tag)) {
			flush();
			blocks.push(...convertBlock(child as Element, state, depth));
		} else if (child.nodeType === Node.ELEMENT_NODE || child.nodeType === Node.TEXT_NODE) {
			pending.push(child);
		}
	});
	flush();
	return blocks;
}

function convertBlock(el: Element, state: BuildState, depth: number): OrgBlock[] {
	if (depth >= ORG_LIMITS.depth - 2) {
		// Too deep for structure: keep the text as a paragraph.
		const v = normalizeSpace(el.textContent || '');
		if (!v) return [];
		countNode(state, v);
		return [{ t: 'para', inl: [{ t: 'text', v }] }];
	}
	const tag = el.tagName.toLowerCase();
	switch (tag) {
		case 'h1': case 'h2': case 'h3': case 'h4': case 'h5': case 'h6': {
			const inl = trimInlines(convertInlines(Array.from(el.childNodes), state, depth + 1));
			if (inl.length === 0) return [];
			countNode(state);
			return [{ t: 'heading', level: Number(tag[1]), inl }];
		}
		case 'p': case 'figcaption': case 'summary': case 'dt': {
			const inl = trimInlines(convertInlines(Array.from(el.childNodes), state, depth + 1));
			if (inl.length === 0) return [];
			countNode(state);
			return [{ t: 'para', inl }];
		}
		case 'ul': case 'ol': {
			const items: OrgBlock[][] = [];
			el.childNodes.forEach((child) => {
				if (tagOf(child) !== 'li') return;
				const item = convertChildren(child, state, depth + 2);
				if (item.length > 0) items.push(item);
			});
			if (items.length === 0) return [];
			countNode(state);
			return [{ t: 'list', ordered: tag === 'ol', items }];
		}
		case 'li': case 'dd':
			return convertChildren(el, state, depth + 1);
		case 'blockquote': {
			const inner = convertChildren(el, state, depth + 1);
			if (inner.length === 0) return [];
			countNode(state);
			return [{ t: 'quote', blocks: inner }];
		}
		case 'pre': {
			const text = (el.textContent || '').replace(/\r\n?/g, '\n').replace(/\n+$/, '');
			if (!text.trim()) return [];
			countNode(state, text);
			return [{ t: 'code', text }];
		}
		case 'hr':
			countNode(state);
			return [{ t: 'hr' }];
		case 'table':
			return convertTable(el, state, depth);
		case 'thead': case 'tbody': case 'tfoot': case 'tr': case 'td': case 'th':
			// Stray table parts outside a table: keep their content.
			return convertChildren(el, state, depth + 1);
		default:
			return convertChildren(el, state, depth + 1);
	}
}

function convertTable(table: Element, state: BuildState, depth: number): OrgBlock[] {
	const rows: OrgInline[][][] = [];
	let header = false;
	table.querySelectorAll('tr').forEach((tr, index) => {
		if (tr.closest('table') !== table) return; // nested tables flatten into their cell text
		const cells: OrgInline[][] = [];
		tr.childNodes.forEach((cell) => {
			const tag = tagOf(cell);
			if (tag !== 'td' && tag !== 'th') return;
			if (index === 0 && tag === 'th') header = true;
			const v = normalizeSpace(cell.textContent || '');
			countNode(state, v);
			cells.push(v ? [{ t: 'text', v }] : []);
		});
		if (cells.length > 0) rows.push(cells);
	});
	if (rows.length === 0) return [];
	countNode(state);
	return [{ t: 'table', header, rows }];
}

function convertInlines(nodes: Node[], state: BuildState, depth: number): OrgInline[] {
	const out: OrgInline[] = [];
	for (const node of nodes) {
		if (node.nodeType === Node.TEXT_NODE) {
			const v = (node.textContent || '').replace(/\s+/g, ' ');
			if (v) {
				countNode(state, v);
				out.push({ t: 'text', v });
			}
			continue;
		}
		if (node.nodeType !== Node.ELEMENT_NODE) continue;
		const el = node as Element;
		const tag = el.tagName.toLowerCase();
		if (SKIP_TAGS.has(tag)) continue;
		if (depth >= ORG_LIMITS.depth - 1) {
			const v = normalizeSpace(el.textContent || '');
			if (v) {
				countNode(state, v);
				out.push({ t: 'text', v });
			}
			continue;
		}
		const children = () => convertInlines(Array.from(el.childNodes), state, depth + 1);
		switch (tag) {
			case 'br':
				countNode(state);
				out.push({ t: 'br' });
				break;
			case 'strong': case 'b':
				out.push(...wrap('bold', children(), state));
				break;
			case 'em': case 'i': case 'cite':
				out.push(...wrap('italic', children(), state));
				break;
			case 's': case 'del': case 'strike':
				out.push(...wrap('strike', children(), state));
				break;
			case 'code': case 'kbd': case 'samp': {
				const v = (el.textContent || '').replace(/\s+/g, ' ');
				if (v.trim()) {
					countNode(state, v);
					out.push({ t: 'code', v });
				}
				break;
			}
			case 'a': {
				const c = children();
				const href = resolveUrl(el.getAttribute('href') || '', state.baseUrl);
				if (href && /^https?:$/i.test(new URL(href).protocol)) {
					countNode(state, href);
					out.push({ t: 'link', href, c });
				} else {
					out.push(...c); // non-web links keep only their text
				}
				break;
			}
			case 'img':
				out.push(...imageInline(el, state));
				break;
			case 'picture': {
				const img = el.querySelector('img');
				if (img) out.push(...imageInline(img, state));
				break;
			}
			default:
				if (BLOCK_TAGS.has(tag)) {
					// Block element inside inline context: keep its text.
					const v = normalizeSpace(el.textContent || '');
					if (v) {
						countNode(state, v);
						out.push({ t: 'text', v: ` ${v} ` });
					}
				} else {
					out.push(...children());
				}
		}
	}
	return out;
}

function wrap(t: 'bold' | 'italic' | 'strike', c: OrgInline[], state: BuildState): OrgInline[] {
	if (c.length === 0) return [];
	countNode(state);
	return [{ t, c }];
}

function imageInline(img: Element, state: BuildState): OrgInline[] {
	const raw = img.getAttribute('src') || '';
	const url = resolveUrl(raw, state.baseUrl);
	if (!url || !url.startsWith('https://')) {
		// data:, blob:, file:, http: and relative junk are not downloaded in this version.
		state.unsupportedImages++;
		return [];
	}
	let id = state.assetByUrl.get(url);
	if (!id) {
		if (state.assets.length >= ORG_LIMITS.assets || url.length > ORG_LIMITS.urlChars) {
			state.unsupportedImages++;
			return [];
		}
		id = `a${state.assets.length + 1}`;
		state.assets.push({ id, url });
		state.assetByUrl.set(url, id);
	}
	countNode(state);
	return [{ t: 'image', asset: id }];
}

function resolveUrl(raw: string, baseUrl: string): string {
	const trimmed = raw.trim();
	if (!trimmed) return '';
	try {
		return new URL(trimmed, baseUrl).href;
	} catch {
		return '';
	}
}

function normalizeSpace(s: string): string {
	return s.replace(/\s+/g, ' ').trim();
}

// Drops leading/trailing whitespace-only text and br nodes of a paragraph.
function trimInlines(inl: OrgInline[]): OrgInline[] {
	const out = inl.slice();
	while (out.length && isBlank(out[0])) out.shift();
	while (out.length && isBlank(out[out.length - 1])) out.pop();
	if (out.length && out[0].t === 'text') out[0] = { t: 'text', v: out[0].v.trimStart() };
	const last = out.length - 1;
	if (last >= 0 && out[last].t === 'text') out[last] = { t: 'text', v: (out[last] as { v: string }).v.trimEnd() };
	return out;
}

function isBlank(n: OrgInline): boolean {
	return n.t === 'br' || (n.t === 'text' && n.v.trim() === '');
}

export function isOrgCategory(value: unknown): value is OrgCategory {
	return typeof value === 'string' && (ORG_CATEGORIES as readonly string[]).includes(value);
}

/** Single-line title as the host expects it; empty means "untitled". */
export function normalizeTitle(title: string): string {
	const single = title.replace(/[\u0000-\u001f\u007f]+/g, ' ').replace(/\s+/g, ' ').trim();
	return Array.from(single).slice(0, ORG_LIMITS.titleChars).join('') || 'Untitled';
}

export function buildOrgSaveRequest(params: {
	requestId: string;
	category: OrgCategory;
	title: string;
	sourceUrl: string;
	contentHtml: string;
	downloadAssets: boolean;
	clippedAt?: Date;
}): { request: OrgSaveRequest; unsupportedImages: number } {
	const { blocks, assets, unsupportedImages } = htmlToOrgBlocks(params.contentHtml, params.sourceUrl);
	const request: OrgSaveRequest = {
		schemaVersion: ORG_SCHEMA_VERSION,
		type: 'saveClip',
		requestId: params.requestId,
		category: params.category,
		title: normalizeTitle(params.title),
		sourceUrl: params.sourceUrl,
		clippedAt: formatRfc3339(params.clippedAt ?? new Date()),
		downloadAssets: params.downloadAssets,
		blocks,
		assets,
	};
	const problem = validateOrgSaveRequest(request);
	if (problem) throw new Error(problem);
	return { request, unsupportedImages };
}

export function formatRfc3339(d: Date): string {
	const pad = (n: number) => String(Math.abs(n)).padStart(2, '0');
	const offset = -d.getTimezoneOffset();
	const sign = offset >= 0 ? '+' : '-';
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}${sign}${pad(Math.trunc(offset / 60))}:${pad(offset % 60)}`;
}

// ---------------------------------------------------------------------------
// Validation (used by the service worker on every request it forwards).

const BLOCK_FIELDS: Record<string, string[]> = {
	heading: ['t', 'level', 'inl'], para: ['t', 'inl'], list: ['t', 'ordered', 'items'],
	quote: ['t', 'blocks'], table: ['t', 'header', 'rows'], code: ['t', 'text'],
	image: ['t', 'asset'], hr: ['t'],
};
const INLINE_FIELDS: Record<string, string[]> = {
	text: ['t', 'v'], bold: ['t', 'c'], italic: ['t', 'c'], strike: ['t', 'c'],
	code: ['t', 'v'], link: ['t', 'href', 'c'], image: ['t', 'asset'], br: ['t'],
};
const REQUEST_FIELDS = ['schemaVersion', 'type', 'requestId', 'category', 'title', 'sourceUrl', 'clippedAt', 'downloadAssets', 'blocks', 'assets'];

interface ValidateState { nodes: number; textBytes: number; assetIds: Set<string>; }

function isPlainObject(v: unknown): v is Record<string, unknown> {
	return typeof v === 'object' && v !== null && !Array.isArray(v);
}

function hasExactKeys(obj: Record<string, unknown>, allowed: string[]): boolean {
	const keys = Object.keys(obj);
	return keys.length === allowed.length && allowed.every((k) => k in obj);
}

function utf8Length(s: string): number {
	return new TextEncoder().encode(s).length;
}

function isWebUrl(s: string): boolean {
	try {
		const u = new URL(s);
		return u.protocol === 'http:' || u.protocol === 'https:';
	} catch {
		return false;
	}
}

/** Returns null when valid, otherwise a short reason. */
export function validateOrgSaveRequest(req: unknown): string | null {
	if (!isPlainObject(req) || !hasExactKeys(req, REQUEST_FIELDS)) return 'request fields';
	if (req.schemaVersion !== ORG_SCHEMA_VERSION || req.type !== 'saveClip') return 'schema';
	if (typeof req.requestId !== 'string' || !UUID_RE.test(req.requestId)) return 'requestId';
	if (!isOrgCategory(req.category)) return 'category';
	if (typeof req.title !== 'string' || !req.title || /[\u0000-\u001f\u007f]/.test(req.title) || Array.from(req.title).length > ORG_LIMITS.titleChars) return 'title';
	if (typeof req.sourceUrl !== 'string' || req.sourceUrl.length > ORG_LIMITS.urlChars || !isWebUrl(req.sourceUrl)) return 'sourceUrl';
	if (typeof req.clippedAt !== 'string' || Number.isNaN(Date.parse(req.clippedAt))) return 'clippedAt';
	if (typeof req.downloadAssets !== 'boolean') return 'downloadAssets';
	if (!Array.isArray(req.assets) || req.assets.length > ORG_LIMITS.assets) return 'assets';
	const state: ValidateState = { nodes: 0, textBytes: 0, assetIds: new Set() };
	for (const asset of req.assets) {
		if (!isPlainObject(asset) || !hasExactKeys(asset, ['id', 'url'])) return 'asset fields';
		if (typeof asset.id !== 'string' || !ASSET_ID_RE.test(asset.id) || state.assetIds.has(asset.id)) return 'asset id';
		if (typeof asset.url !== 'string' || asset.url.length > ORG_LIMITS.urlChars) return 'asset url';
		state.assetIds.add(asset.id);
	}
	if (!Array.isArray(req.blocks)) return 'blocks';
	for (const b of req.blocks) {
		const problem = validateBlock(b, state, 1);
		if (problem) return problem;
	}
	if (utf8Length(JSON.stringify(req)) > ORG_LIMITS.messageBytes) return 'too large';
	return null;
}

function bump(state: ValidateState, depth: number, text = ''): string | null {
	state.nodes++;
	state.textBytes += utf8Length(text);
	if (depth > ORG_LIMITS.depth) return 'depth';
	if (state.nodes > ORG_LIMITS.nodes || state.textBytes > ORG_LIMITS.textBytes) return 'too large';
	return null;
}

function validateBlock(b: unknown, state: ValidateState, depth: number): string | null {
	if (!isPlainObject(b) || typeof b.t !== 'string' || !(b.t in BLOCK_FIELDS) || !hasExactKeys(b, BLOCK_FIELDS[b.t])) return 'block';
	switch (b.t) {
		case 'heading':
			if (typeof b.level !== 'number' || !Number.isInteger(b.level) || b.level < 1 || b.level > 6) return 'heading level';
			return bump(state, depth) ?? validateInlines(b.inl, state, depth + 1);
		case 'para':
			return bump(state, depth) ?? validateInlines(b.inl, state, depth + 1);
		case 'list': {
			if (typeof b.ordered !== 'boolean' || !Array.isArray(b.items)) return 'list';
			const p = bump(state, depth);
			if (p) return p;
			for (const item of b.items) {
				if (!Array.isArray(item)) return 'list item';
				for (const child of item) {
					const q = validateBlock(child, state, depth + 1);
					if (q) return q;
				}
			}
			return null;
		}
		case 'quote': {
			if (!Array.isArray(b.blocks)) return 'quote';
			const p = bump(state, depth);
			if (p) return p;
			for (const child of b.blocks) {
				const q = validateBlock(child, state, depth + 1);
				if (q) return q;
			}
			return null;
		}
		case 'table': {
			if (typeof b.header !== 'boolean' || !Array.isArray(b.rows)) return 'table';
			const p = bump(state, depth);
			if (p) return p;
			for (const row of b.rows) {
				if (!Array.isArray(row)) return 'table row';
				for (const cell of row) {
					const q = validateInlines(cell, state, depth + 2);
					if (q) return q;
				}
			}
			return null;
		}
		case 'code':
			if (typeof b.text !== 'string') return 'code';
			return bump(state, depth, b.text);
		case 'image':
			if (typeof b.asset !== 'string' || !state.assetIds.has(b.asset)) return 'image asset';
			return bump(state, depth);
		case 'hr':
			return bump(state, depth);
	}
	return 'block';
}

function validateInlines(list: unknown, state: ValidateState, depth: number): string | null {
	if (!Array.isArray(list)) return 'inlines';
	for (const n of list) {
		if (!isPlainObject(n) || typeof n.t !== 'string' || !(n.t in INLINE_FIELDS) || !hasExactKeys(n, INLINE_FIELDS[n.t])) return 'inline';
		let p: string | null;
		switch (n.t) {
			case 'text': case 'code':
				if (typeof n.v !== 'string') return 'inline text';
				p = bump(state, depth, n.v);
				break;
			case 'bold': case 'italic': case 'strike':
				p = bump(state, depth) ?? validateInlines(n.c, state, depth + 1);
				break;
			case 'link':
				if (typeof n.href !== 'string' || n.href.length > ORG_LIMITS.urlChars) return 'link';
				p = bump(state, depth, n.href) ?? validateInlines(n.c, state, depth + 1);
				break;
			case 'image':
				if (typeof n.asset !== 'string' || !state.assetIds.has(n.asset)) return 'image asset';
				p = bump(state, depth);
				break;
			default:
				p = bump(state, depth);
		}
		if (p) return p;
	}
	return null;
}

export { BuildLimitError };
