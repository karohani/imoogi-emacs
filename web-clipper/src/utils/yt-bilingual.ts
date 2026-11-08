// Bilingual YouTube transcripts: when a video's original (spoken) language is
// not Korean, rebuild the transcript as "original line + Korean line" pairs so
// the clip needs no separate translation. Korean comes from YouTube's
// auto-translation of the original track (tlang=ko, cue-aligned), then from an
// uploaded Korean track (time-window aligned); without either, the original
// alone replaces whatever track defuddle picked.

const TARGET_LANG = 'ko';
const FETCH_TIMEOUT_MS = 8000;
const GROUP_MAX_SECONDS = 30;
const GROUP_MIN_SECONDS = 10;
const GROUP_GAP_SECONDS = 20;
const INNERTUBE_URL = 'https://www.youtube.com/youtubei/v1/player?prettyPrint=false';
const IOS_CONTEXT = { client: { clientName: 'IOS', clientVersion: '20.10.3' } };

interface CaptionTrack {
	baseUrl?: string;
	languageCode?: string;
	kind?: string;
}

interface Cue {
	start: number;
	text: string;
}

interface Group {
	start: number;
	text: string;
	ko?: string;
}

export interface BilingualTranscript {
	html: string;
	text: string;
	language: string;
}

function baseLang(code: string | undefined): string {
	return (code || '').trim().replace(/_/g, '-').toLowerCase().split('-')[0];
}

export function getVideoId(url: string): string {
	try {
		const u = new URL(url);
		if (u.hostname === 'youtu.be') return u.pathname.slice(1);
		if (u.pathname.includes('/shorts/')) return u.pathname.split('/shorts/')[1].split('/')[0];
		return u.searchParams.get('v') || '';
	} catch {
		return '';
	}
}

function captionTracks(player: any): CaptionTrack[] {
	const t = player?.captions?.playerCaptionsTracklistRenderer?.captionTracks;
	return Array.isArray(t) ? t.filter((x: CaptionTrack) => x?.baseUrl) : [];
}

// The spoken language: the audio track not flagged as an AI dub, else the
// language of the auto-generated (speech-recognised) track, else the only track.
export function originalLanguage(player: any): string {
	const formats = player?.streamingData?.adaptiveFormats;
	if (Array.isArray(formats)) {
		const audio = formats.map((f: any) => f?.audioTrack).filter(Boolean);
		if (audio.length > 0) {
			const original = audio.find((a: any) => !a.isAutoDubbed);
			if (original?.id) return baseLang(String(original.id).split('.')[0]);
		}
	}
	const tracks = captionTracks(player);
	const asr = tracks.find(t => t.kind === 'asr');
	if (asr) return baseLang(asr.languageCode);
	return tracks.length === 1 ? baseLang(tracks[0].languageCode) : '';
}

function pickTrack(tracks: CaptionTrack[], lang: string): CaptionTrack | undefined {
	const matches = tracks.filter(t => baseLang(t.languageCode) === lang);
	return matches.find(t => t.kind !== 'asr') ?? matches[0];
}

function decodeEntities(text: string): string {
	return text
		.replace(/&amp;/g, '&')
		.replace(/&lt;/g, '<')
		.replace(/&gt;/g, '>')
		.replace(/&quot;/g, '"')
		.replace(/&#39;/g, "'")
		.replace(/&apos;/g, "'")
		.replace(/&#x([0-9a-fA-F]+);/g, (_, hex) => String.fromCodePoint(parseInt(hex, 16)))
		.replace(/&#(\d+);/g, (_, dec) => String.fromCodePoint(parseInt(dec, 10)));
}

function clean(raw: string): string {
	return decodeEntities(raw.replace(/<[^>]+>/g, '').replace(/\n/g, ' ')).replace(/\s{2,}/g, ' ').trim();
}

// Parses both timedtext formats: <text start="s"> and srv3 <p t="ms">.
export function parseCaptionXml(xml: string): Cue[] {
	const cues: Cue[] = [];
	for (const m of xml.matchAll(/<text\s+start="([^"]*)"[^>]*>([\s\S]*?)<\/text>/g)) {
		const text = clean(m[2]);
		if (text) cues.push({ start: parseFloat(m[1]), text });
	}
	if (cues.length > 0) return cues;
	for (const m of xml.matchAll(/<p\s+t="(\d+)"[^>]*>([\s\S]*?)<\/p>/g)) {
		const text = clean(m[2]);
		if (text) cues.push({ start: parseInt(m[1], 10) / 1000, text });
	}
	return cues;
}

// Groups cues into readable paragraphs; returns index ranges [from, to).
export function groupCues(cues: Cue[]): Array<[number, number]> {
	const ranges: Array<[number, number]> = [];
	let from = 0;
	for (let i = 0; i < cues.length; i++) {
		const next = cues[i + 1];
		const span = cues[i].start - cues[from].start;
		const sentenceEnd = /[.!?…。？！]["')\]]?$/.test(cues[i].text);
		if (!next
			|| next.start - cues[i].start > GROUP_GAP_SECONDS
			|| next.start - cues[from].start >= GROUP_MAX_SECONDS
			|| (sentenceEnd && span >= GROUP_MIN_SECONDS)) {
			ranges.push([from, i + 1]);
			from = i + 1;
		}
	}
	return ranges;
}

const join = (cues: Cue[]) => cues.map(c => c.text).join(' ').replace(/\s{2,}/g, ' ').trim();

// Pairs original groups with Korean text. A translation of the same track keeps
// cue boundaries, so it maps by index; an independent track maps by time window.
export function pairGroups(orig: Cue[], ko: Cue[] | undefined): Group[] {
	const ranges = groupCues(orig);
	const sameCues = !!ko && ko.length === orig.length
		&& ko.every((c, i) => Math.abs(c.start - orig[i].start) < 0.05);
	return ranges.map(([from, to], gi) => {
		const group: Group = { start: orig[from].start, text: join(orig.slice(from, to)) };
		if (ko && ko.length > 0) {
			let part: Cue[];
			if (sameCues) {
				part = ko.slice(from, to);
			} else {
				const lo = gi === 0 ? -Infinity : group.start;
				const hi = gi + 1 < ranges.length ? orig[ranges[gi + 1][0]].start : Infinity;
				part = ko.filter(c => c.start >= lo && c.start < hi);
			}
			const text = join(part);
			if (text) group.ko = text;
		}
		return group;
	});
}

function formatTimestamp(seconds: number): string {
	const h = Math.floor(seconds / 3600);
	const m = Math.floor((seconds % 3600) / 60);
	const s = Math.floor(seconds % 60);
	return h > 0
		? `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
		: `${m}:${String(s).padStart(2, '0')}`;
}

function parseTimestamp(ts: string): number | null {
	const parts = ts.trim().split(':').map(Number);
	if (parts.length < 2 || parts.length > 3 || parts.some(isNaN)) return null;
	return parts.reduce((acc, p) => acc * 60 + p, 0);
}

function escapeHtml(s: string): string {
	return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

export interface Chapter {
	start: number;
	title: string;
}

// Mirrors defuddle's transcript markup so downstream converters treat it the same.
export function buildTranscript(groups: Group[], chapters: Chapter[]): { html: string; text: string } {
	const sorted = [...chapters].sort((a, b) => a.start - b.start);
	let ci = 0;
	const html: string[] = [];
	const text: string[] = [];
	for (const g of groups) {
		while (ci < sorted.length && sorted[ci].start <= g.start) {
			html.push(`<h3>${escapeHtml(sorted[ci].title)}</h3>`);
			if (text.length > 0) text.push('');
			text.push(`### ${sorted[ci].title}`, '');
			ci++;
		}
		const ts = formatTimestamp(g.start);
		const ko = g.ko ? `<br>${escapeHtml(g.ko)}` : '';
		html.push(`<p class="transcript-segment"><strong><span class="timestamp" data-timestamp="${g.start}">${ts}</span></strong> · ${escapeHtml(g.text)}${ko}</p>`);
		text.push(`**${ts}** · ${g.text}`);
		if (g.ko) text.push(g.ko);
	}
	return {
		html: `<div class="youtube transcript">\n<h2>Transcript</h2>\n${html.join('\n')}\n</div>`,
		text: text.join('\n'),
	};
}

async function fetchText(url: string, init?: RequestInit): Promise<string | undefined> {
	try {
		const resp = await fetch(url, { ...init, signal: AbortSignal.timeout(FETCH_TIMEOUT_MS) });
		return resp.ok ? await resp.text() : undefined;
	} catch {
		return undefined;
	}
}

async function fetchCues(url: string): Promise<Cue[] | undefined> {
	const xml = await fetchText(url);
	const cues = xml ? parseCaptionXml(xml) : [];
	return cues.length > 0 ? cues : undefined;
}

// The page's own player response, unless YouTube's SPA navigation left a stale one.
function inlinePlayerResponse(doc: Document, videoId: string): any {
	for (const script of Array.from(doc.querySelectorAll('script'))) {
		const src = script.textContent || '';
		const m = /ytInitialPlayerResponse\s*=\s*\{/.exec(src);
		if (!m) continue;
		const open = m.index + m[0].length - 1;
		let depth = 0;
		let inStr = false;
		for (let i = open; i < src.length; i++) {
			const ch = src[i];
			if (inStr) {
				if (ch === '\\') i++;
				else if (ch === '"') inStr = false;
			} else if (ch === '"') inStr = true;
			else if (ch === '{') depth++;
			else if (ch === '}' && --depth === 0) {
				try {
					const data = JSON.parse(src.slice(open, i + 1));
					return data?.videoDetails?.videoId === videoId ? data : undefined;
				} catch {
					return undefined;
				}
			}
		}
	}
	return undefined;
}

async function apiPlayerResponse(videoId: string): Promise<any> {
	const body = await fetchText(INNERTUBE_URL, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ context: IOS_CONTEXT, videoId }),
	});
	try {
		return body ? JSON.parse(body) : undefined;
	} catch {
		return undefined;
	}
}

/**
 * Builds a bilingual transcript for the video at url, or returns undefined
 * when the video is Korean-original, has no captions, or the fetch fails —
 * in which case the caller keeps defuddle's transcript unchanged.
 */
export async function fetchBilingualTranscript(doc: Document, url: string, chapters: Chapter[]): Promise<BilingualTranscript | undefined> {
	const videoId = getVideoId(url);
	if (!videoId) return undefined;
	const sources = [() => inlinePlayerResponse(doc, videoId), () => apiPlayerResponse(videoId)];
	for (const load of sources) {
		const player = await load();
		const tracks = captionTracks(player);
		if (tracks.length === 0) continue;
		const lang = originalLanguage(player);
		if (!lang || lang === TARGET_LANG) return undefined;
		const origTrack = pickTrack(tracks, lang);
		if (!origTrack?.baseUrl) return undefined;
		const orig = await fetchCues(origTrack.baseUrl);
		if (!orig) continue;
		let ko = await fetchCues(`${origTrack.baseUrl}&tlang=${TARGET_LANG}`);
		if (!ko) {
			const koTrack = pickTrack(tracks, TARGET_LANG);
			if (koTrack?.baseUrl) ko = await fetchCues(koTrack.baseUrl);
		}
		const { html, text } = buildTranscript(pairGroups(orig, ko), chapters);
		return { html, text, language: lang };
	}
	return undefined;
}

/**
 * Swaps the transcript block inside defuddle's content HTML for a bilingual
 * one, keeping its chapter headings. Returns the new content, or undefined
 * when nothing changed.
 */
export async function applyBilingualTranscript(doc: Document, url: string, contentHtml: string): Promise<{ content: string; transcript: string; language: string } | undefined> {
	const parsed = new DOMParser().parseFromString(`<body>${contentHtml}</body>`, 'text/html');
	const heading = Array.from(parsed.querySelectorAll('h2')).find(h => h.textContent?.trim() === 'Transcript');
	const container = heading?.parentElement && heading.parentElement !== parsed.body ? heading.parentElement : undefined;
	// Without a wrapper the transcript is the heading plus everything after it
	// (defuddle appends it last).
	const blocks: Element[] = container ? Array.from(container.children) : [];
	if (heading && !container) {
		for (let el: Element | null = heading; el; el = el.nextElementSibling) blocks.push(el);
	}
	const chapters: Chapter[] = [];
	{
		let pending: string[] = [];
		for (const el of blocks) {
			if (el.tagName === 'H3') {
				pending.push(el.textContent?.trim() || '');
			} else if (el.tagName === 'P' && pending.length > 0) {
				const start = parseTimestamp(el.querySelector('strong')?.textContent || '');
				if (start !== null) pending.forEach(title => chapters.push({ start, title }));
				pending = [];
			}
		}
	}
	const result = await fetchBilingualTranscript(doc, url, chapters);
	if (!result) return undefined;
	if (container) {
		container.outerHTML = result.html;
	} else if (heading) {
		blocks.slice(1).forEach(el => el.remove());
		heading.outerHTML = result.html;
	} else {
		parsed.body.insertAdjacentHTML('beforeend', result.html);
	}
	return { content: parsed.body.innerHTML, transcript: result.text, language: result.language };
}
