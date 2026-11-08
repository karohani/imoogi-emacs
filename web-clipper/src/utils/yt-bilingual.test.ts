// @vitest-environment jsdom
import { describe, test, expect, vi, afterEach } from 'vitest';
import {
	applyBilingualTranscript,
	buildTranscript,
	groupCues,
	originalLanguage,
	pairGroups,
	parseCaptionXml,
} from './yt-bilingual';
import { htmlToOrgBlocks } from './org-clip';

const enXml = '<?xml version="1.0"?><transcript>'
	+ '<text start="0.2" dur="3">Hello. I&amp;#39;m Max.</text>'
	+ '<text start="12.5" dur="3">Today we build an app.</text>'
	+ '<text start="25" dur="3">It is fun.</text></transcript>';
const koTlangXml = '<?xml version="1.0"?><transcript>'
	+ '<text start="0.2" dur="3">안녕하세요. 맥스입니다.</text>'
	+ '<text start="12.5" dur="3">오늘은 앱을 만듭니다.</text>'
	+ '<text start="25" dur="3">재미있어요.</text></transcript>';

describe('parseCaptionXml', () => {
	test('decodes double-escaped entities in <text> format', () => {
		expect(parseCaptionXml(enXml)[0]).toEqual({ start: 0.2, text: "Hello. I'm Max." });
	});
	test('reads srv3 <p t> format', () => {
		expect(parseCaptionXml('<p t="1500" d="1000"><s>Hi</s><s> there</s></p>')).toEqual([{ start: 1.5, text: 'Hi there' }]);
	});
});

describe('originalLanguage', () => {
	test('prefers the non-dubbed audio track', () => {
		const player = {
			streamingData: { adaptiveFormats: [
				{ audioTrack: { id: 'en-US.10', isAutoDubbed: true } },
				{ audioTrack: { id: 'ko.4' } },
			] },
			captions: { playerCaptionsTracklistRenderer: { captionTracks: [
				{ baseUrl: 'u', languageCode: 'en', kind: 'asr' },
			] } },
		};
		expect(originalLanguage(player)).toBe('ko');
	});
	test('falls back to the speech-recognised track', () => {
		const player = { captions: { playerCaptionsTracklistRenderer: { captionTracks: [
			{ baseUrl: 'a', languageCode: 'ko' },
			{ baseUrl: 'b', languageCode: 'en', kind: 'asr' },
		] } } };
		expect(originalLanguage(player)).toBe('en');
	});
});

describe('pairing', () => {
	test('groups at sentence ends after the minimum span', () => {
		expect(groupCues(parseCaptionXml(enXml))).toEqual([[0, 2], [2, 3]]);
	});
	test('maps a translation of the same track by cue index', () => {
		const groups = pairGroups(parseCaptionXml(enXml), parseCaptionXml(koTlangXml));
		expect(groups.map(g => g.ko)).toEqual(['안녕하세요. 맥스입니다. 오늘은 앱을 만듭니다.', '재미있어요.']);
	});
	test('maps an independent track by time window', () => {
		const ko = [{ start: 0.1, text: '가' }, { start: 13, text: '나' }, { start: 24.9, text: '다' }, { start: 26, text: '라' }];
		const groups = pairGroups(parseCaptionXml(enXml), ko);
		expect(groups.map(g => g.ko)).toEqual(['가 나 다', '라']);
	});
	test('renders to Org blocks as original line + Korean line', () => {
		const { html, text } = buildTranscript(pairGroups(parseCaptionXml(enXml), parseCaptionXml(koTlangXml)), [{ start: 0, title: 'Intro' }]);
		expect(text.split('\n').slice(0, 4)).toEqual(['### Intro', '', "**0:00** · Hello. I'm Max. Today we build an app.", '안녕하세요. 맥스입니다. 오늘은 앱을 만듭니다.']);
		const blocks = JSON.stringify(htmlToOrgBlocks(html, 'https://www.youtube.com/watch?v=x'));
		expect(blocks).toContain('안녕하세요');
		expect(blocks).toContain('Intro');
	});
});

describe('applyBilingualTranscript', () => {
	afterEach(() => vi.unstubAllGlobals());

	const player = {
		videoDetails: { videoId: 'vid1' },
		captions: { playerCaptionsTracklistRenderer: { captionTracks: [
			{ baseUrl: 'https://www.youtube.com/api/timedtext?v=vid1&lang=en', languageCode: 'en', kind: 'asr' },
		] } },
	};
	const pageDoc = () => {
		const d = document.implementation.createHTMLDocument('');
		const s = d.createElement('script');
		s.textContent = `var ytInitialPlayerResponse = ${JSON.stringify(player)};var x = {};`;
		d.head.appendChild(s);
		return d;
	};
	const content = '<p>desc</p><div><h2>Transcript</h2><h3>Intro</h3><p><strong>0:00</strong> · old</p></div>';

	test('replaces the transcript block and keeps chapters', async () => {
		vi.stubGlobal('fetch', vi.fn(async (url: string) => new Response(url.includes('tlang=ko') ? koTlangXml : enXml)));
		const res = await applyBilingualTranscript(pageDoc(), 'https://www.youtube.com/watch?v=vid1', content);
		expect(res?.language).toBe('en');
		expect(res?.content).toContain('<p>desc</p>');
		expect(res?.content).not.toContain('old');
		expect(res?.content).toContain('<h3>Intro</h3>');
		expect(res?.content).toContain('<br>안녕하세요');
	});

	test('keeps the original alone when no Korean is available', async () => {
		vi.stubGlobal('fetch', vi.fn(async (url: string) => url.includes('tlang=ko') ? new Response('', { status: 429 }) : new Response(enXml)));
		const res = await applyBilingualTranscript(pageDoc(), 'https://www.youtube.com/watch?v=vid1', content);
		expect(res?.content).toContain('Hello.');
		expect(res?.content).not.toContain('<br>');
	});

	test('leaves Korean-original videos untouched', async () => {
		const koPlayer = { ...player, captions: { playerCaptionsTracklistRenderer: { captionTracks: [
			{ baseUrl: 'https://www.youtube.com/api/timedtext?lang=ko', languageCode: 'ko', kind: 'asr' },
		] } } };
		const d = document.implementation.createHTMLDocument('');
		const s = d.createElement('script');
		s.textContent = `var ytInitialPlayerResponse = ${JSON.stringify(koPlayer)};`;
		d.head.appendChild(s);
		const fetchMock = vi.fn();
		vi.stubGlobal('fetch', fetchMock);
		expect(await applyBilingualTranscript(d, 'https://www.youtube.com/watch?v=vid1', content)).toBeUndefined();
		expect(fetchMock).not.toHaveBeenCalled();
	});
});
