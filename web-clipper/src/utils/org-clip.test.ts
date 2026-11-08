// @vitest-environment jsdom
import { describe, test, expect } from 'vitest';
import {
	buildOrgSaveRequest,
	htmlToOrgBlocks,
	normalizeTitle,
	validateOrgSaveRequest,
	ORG_LIMITS,
	OrgSaveRequest,
} from './org-clip';

const ID = '8c3f0f8e-1f8a-4c55-9a59-0e5a3f8f9d21';
const BASE = 'https://example.com/posts/1';

function build(html: string, overrides: Partial<Parameters<typeof buildOrgSaveRequest>[0]> = {}) {
	return buildOrgSaveRequest({
		requestId: ID,
		category: 'clippings',
		title: 'Title',
		sourceUrl: BASE,
		contentHtml: html,
		downloadAssets: true,
		clippedAt: new Date('2026-11-08T10:00:00+09:00'),
		...overrides,
	});
}

describe('htmlToOrgBlocks', () => {
	test('converts paragraphs, headings, lists, quotes, code, tables, hr', () => {
		const { blocks } = htmlToOrgBlocks(`
			<h2>Intro <em>here</em></h2>
			<p>Hello <strong>bold</strong> and <a href="/x?y=1">link</a>.</p>
			<ul><li>one</li><li>two<ol><li>nested</li></ol></li></ul>
			<blockquote><p>quoted</p></blockquote>
			<pre><code>#+begin_src sh\nrm -rf ~\n</code></pre>
			<table><tr><th>a</th><th>b|c</th></tr><tr><td>1</td><td>2</td></tr></table>
			<hr>`, BASE);
		expect(blocks.map((b) => b.t)).toEqual(['heading', 'para', 'list', 'quote', 'code', 'table', 'hr']);
		expect(blocks[0]).toEqual({ t: 'heading', level: 2, inl: [{ t: 'text', v: 'Intro ' }, { t: 'italic', c: [{ t: 'text', v: 'here' }] }] });
		const para = blocks[1] as { inl: unknown[] };
		expect(para.inl).toContainEqual({ t: 'link', href: 'https://example.com/x?y=1', c: [{ t: 'text', v: 'link' }] });
		const list = blocks[2] as { items: { t: string }[][] };
		expect(list.items).toHaveLength(2);
		expect(list.items[1].map((b) => b.t)).toEqual(['para', 'list']);
		expect(blocks[4]).toEqual({ t: 'code', text: '#+begin_src sh\nrm -rf ~' });
		expect(blocks[5]).toEqual({ t: 'table', header: true, rows: [
			[[{ t: 'text', v: 'a' }], [{ t: 'text', v: 'b|c' }]],
			[[{ t: 'text', v: '1' }], [{ t: 'text', v: '2' }]],
		] });
	});

	test('drops scripts, iframes, embeds, forms and event attributes', () => {
		const { blocks } = htmlToOrgBlocks(`
			<p onclick="alert(1)">safe</p>
			<script>alert(1)</script><iframe src="https://evil"></iframe>
			<object data="x"></object><embed src="x"><svg><text>svg</text></svg>
			<form><input value="x"><button>go</button></form><style>p{}</style>`, BASE);
		expect(JSON.stringify(blocks)).not.toMatch(/alert|evil|svg|onclick|go/);
		expect(blocks).toEqual([{ t: 'para', inl: [{ t: 'text', v: 'safe' }] }]);
	});

	test('non-web links keep only their text', () => {
		const { blocks } = htmlToOrgBlocks(
			'<p><a href="javascript:alert(1)">js</a> <a href="elisp:(x)">el</a> <a href="file:///etc/passwd">f</a> <a href="shell:rm">sh</a></p>',
			BASE);
		expect(JSON.stringify(blocks)).not.toMatch(/"link"/);
		expect(JSON.stringify(blocks)).not.toMatch(/javascript|passwd|elisp:/);
	});

	test('images: only https become assets, deduplicated, others counted as unsupported', () => {
		const { blocks, assets, unsupportedImages } = htmlToOrgBlocks(`
			<p><img src="/a.png"> <img src="https://example.com/a.png"> <img src="data:image/png;base64,AAAA">
			<img src="blob:https://example.com/1"> <img src="http://insecure.example/x.jpg"> <img src="file:///x.png"></p>`, BASE);
		expect(assets).toEqual([{ id: 'a1', url: 'https://example.com/a.png' }]);
		expect(unsupportedImages).toBe(4);
		expect(JSON.stringify(blocks).match(/"image"/g)).toHaveLength(2);
	});

	test('caps asset count at the protocol limit', () => {
		const imgs = Array.from({ length: 40 }, (_, i) => `<img src="https://cdn.example.com/${i}.png">`).join('');
		const { assets, unsupportedImages } = htmlToOrgBlocks(`<p>${imgs}</p>`, BASE);
		expect(assets).toHaveLength(ORG_LIMITS.assets);
		expect(unsupportedImages).toBe(10);
	});

	test('deep nesting is flattened and still validates', () => {
		const deep = '<div>'.repeat(200) + '<ul><li>'.repeat(30) + 'deep' + '</li></ul>'.repeat(30) + '</div>'.repeat(200);
		const { request } = build(deep);
		expect(validateOrgSaveRequest(request)).toBeNull();
		expect(JSON.stringify(request.blocks)).toContain('deep');
	});

	test('rejects oversized html before parsing', () => {
		expect(() => htmlToOrgBlocks('x'.repeat(ORG_LIMITS.htmlChars + 1), BASE)).toThrow();
	});
});

describe('normalizeTitle', () => {
	test('collapses newlines and control characters', () => {
		expect(normalizeTitle('a\n#+SETUPFILE: x\r\tb\u0000c')).toBe('a #+SETUPFILE: x b c');
	});
	test('keeps Korean and truncates by code point', () => {
		expect(normalizeTitle('한글 제목')).toBe('한글 제목');
		expect(Array.from(normalizeTitle('가'.repeat(2000)))).toHaveLength(ORG_LIMITS.titleChars);
	});
	test('empty becomes Untitled', () => {
		expect(normalizeTitle(' \n ')).toBe('Untitled');
	});
});

describe('validateOrgSaveRequest', () => {
	const valid = () => JSON.parse(JSON.stringify(build('<p>hi <img src="https://e.com/x.png"></p>').request)) as OrgSaveRequest & Record<string, unknown>;

	test('accepts a built request', () => {
		expect(validateOrgSaveRequest(valid())).toBeNull();
	});

	test.each([
		['unknown category', (r: any) => { r.category = '../../etc'; }],
		['extra field', (r: any) => { r.path = '/tmp/x'; }],
		['missing field', (r: any) => { delete r.title; }],
		['bad request id', (r: any) => { r.requestId = 'not-a-uuid'; }],
		['non-web source url', (r: any) => { r.sourceUrl = 'file:///etc/passwd'; }],
		['newline in title', (r: any) => { r.title = 'a\nb'; }],
		['unknown block type', (r: any) => { r.blocks.push({ t: 'html', v: '<script>' }); }],
		['extra block field', (r: any) => { r.blocks[0].raw = '#+INCLUDE'; }],
		['image references unknown asset', (r: any) => { r.blocks.push({ t: 'image', asset: 'a9' }); }],
		['bad asset id', (r: any) => { r.assets[0].id = '../x'; }],
		['duplicate asset id', (r: any) => { r.assets.push({ id: 'a1', url: 'https://e.com/y.png' }); }],
		['heading level out of range', (r: any) => { r.blocks.push({ t: 'heading', level: 9, inl: [] }); }],
		['wrong schema version', (r: any) => { r.schemaVersion = 2; }],
	])('rejects %s', (_name, mutate) => {
		const r = valid();
		mutate(r);
		expect(validateOrgSaveRequest(r)).not.toBeNull();
	});

	test('rejects excessive depth', () => {
		const r = valid();
		let inl: any[] = [{ t: 'text', v: 'x' }];
		for (let i = 0; i < 20; i++) inl = [{ t: 'bold', c: inl }];
		r.blocks.push({ t: 'para', inl });
		expect(validateOrgSaveRequest(r)).toBe('depth');
	});

	test('rejects too many assets', () => {
		const r = valid();
		r.assets = Array.from({ length: 31 }, (_, i) => ({ id: `a${i + 1}`, url: `https://e.com/${i}.png` }));
		expect(validateOrgSaveRequest(r)).toBe('assets');
	});
});
