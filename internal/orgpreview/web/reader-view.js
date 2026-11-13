(function (global) {
    'use strict';
    function createRevisionState() {
        return { mode: 'document', displayedRevision: -1, latestReceivedRevision: -1, pendingRevision: null, renderGeneration: 0,
            receive(msg) {
                if (typeof msg.revision !== 'number' || msg.revision < this.latestReceivedRevision)
                    return false;
                this.latestReceivedRevision = msg.revision;
                if (this.mode === 'document')
                    return true;
                this.pendingRevision = msg;
                return false;
            },
            applied(msg) {
                this.displayedRevision = msg.revision;
                this.renderGeneration++;
                this.pendingRevision = null;
            },
            canNavigate() {
                return this.displayedRevision === this.latestReceivedRevision;
            } };
    }
    function create(options) {
        const { root, document } = options, state = createRevisionState(), P = global.OrgReaderPagination;
        let layout = null, index = 0, spread = 1, preference = 'auto', font = 18, timer = 0, layoutGeneration = 0;
        const bar = document.createElement('nav');
        bar.className = 'reader-toolbar';
        bar.setAttribute('aria-label', '읽기 보기');
        const footer = document.createElement('nav');
        footer.className = 'reader-footer';
        footer.hidden = true;
        footer.setAttribute('aria-label', '쪽 이동');
        const status = document.createElement('span');
        status.className = 'reader-status';
        status.setAttribute('role', 'status');
        status.setAttribute('aria-live', 'polite');
        const button = (label, fn, parent = bar) => {
            const b = document.createElement('button');
            b.type = 'button';
            b.textContent = label;
            b.onclick = fn;
            parent.appendChild(b);
            return b;
        };
        const modes = {};
        ['document', 'book', 'present'].forEach((mode, i) => modes[mode] = button(['문서', '책자', '발표'][i], () => setMode(mode)));
        const select = document.createElement('select');
        select.setAttribute('aria-label', '쪽 배치');
        [['auto', '자동 두 쪽'], ['single', '한 쪽']].forEach(([value, label]) => {
            const o = document.createElement('option');
            o.value = value;
            o.textContent = label;
            select.appendChild(o);
        });
        select.onchange = () => {
            preference = select.value;
            reflow();
        };
        bar.appendChild(select);
        button('글자 작게', () => {
            font = Math.max(14, font - 2);
            reflow();
        });
        button('글자 크게', () => {
            font = Math.min(32, font + 2);
            reflow();
        });
        const update = button(options.linked ? '다시 불러오기' : '최신 내용 적용', () => options.linked ? options.reload() : applyPending());
        const source = button('원문으로 이동', () => {
            if (!state.canNavigate()) {
                announce('최신 내용 적용 후 이동하세요.');
                return;
            }
            const el = currentBlock();
            if (el)
                options.navigate(el);
        });
        source.disabled = !!options.linked;
        const fullscreen = button('전체 화면', async () => {
            try {
                if (document.fullscreenElement)
                    await document.exitFullscreen();
                else
                    await document.documentElement.requestFullscreen();
            }
            catch (_) {
                announce('전체 화면을 열 수 없습니다. 일반 창에서 발표를 계속할 수 있습니다.');
            }
        });
        const previous = button('← 이전', () => move(-spread), footer);
        const count = document.createElement('span');
        footer.appendChild(count);
        const next = button('다음 →', () => move(spread), footer);
        bar.appendChild(status);
        const shell = document.createElement('div');
        shell.className = 'reader-shell';
        root.before(shell);
        shell.append(bar, root, footer);
        function announce(message) {
            status.textContent = message;
        }
        function currentBlock() {
            return layout && layout.pages[index] && layout.pages[index][0];
        }
        function anchor() {
            if (layout) {
                const el = currentBlock();
                return layout.records.find(r => r.el === el);
            }
            const main = root.querySelector('main.org-preview');
            if (!main)
                return null;
            const records = P.describe(Array.from(main.children));
            return records.find(r => r.el.classList.contains('active-block') || r.el.querySelector('.active-block')) || records.find(r => r.el.getBoundingClientRect().bottom > bar.getBoundingClientRect().bottom) || records[0];
        }
        function reflow(saved) {
            if (state.mode === 'document' || !layout)
                return;
            const token = ++layoutGeneration, position = saved || anchor(), main = root.querySelector('main.org-preview');
            main.style.setProperty('--reader-font', (state.mode === 'present' ? font + 4 : font) + 'px');
            const width = main.clientWidth;
            if (width <= 0)
                return;
            spread = P.spreadFor(width, preference, state.mode);
            const pageWidth = (width - (spread - 1) * 24) / spread;
            const height = Math.max(80, main.clientHeight - 2);
            layout.paginate({ width: pageWidth, height });
            if (token !== layoutGeneration)
                return;
            const match = P.resolveAnchor(layout.records, position);
            index = layout.pageFor(layout.records[match.index]?.el);
            index = Math.floor(index / spread) * spread;
            show();
            if (match.fallback)
                announce('읽던 문단을 찾지 못해 장 또는 문서 시작으로 이동했습니다.');
        }
        function show() {
            layout.show(index, spread);
            const total = layout.pages.length;
            count.textContent = total ? (index + 1) + (spread === 2 && index + 1 < total ? '–' + Math.min(total, index + spread) : '') + ' / ' + total + '쪽' : '내용 없음';
            previous.disabled = index === 0;
            next.disabled = index + spread >= total;
            announce((state.pendingRevision ? '새 내용 있음 · ' : '') + count.textContent + ' · ' + (layout.records.find(r => r.el === currentBlock())?.path || ''));
        }
        function move(delta) {
            if (!layout)
                return;
            index = Math.max(0, Math.min(Math.max(0, Math.floor((layout.pages.length - 1) / spread) * spread), index + delta));
            show();
        }
        function setMode(mode) {
            if (mode === state.mode)
                return;
            if (mode !== 'document' && !root.querySelector('main.org-preview')) {
                announce('문서를 불러온 뒤 선택하세요.');
                return;
            }
            if (mode !== 'document' && root.querySelector('main.org-preview iframe')) {
                announce('이 콘텐츠는 문서 보기에서 확인하세요.');
                return;
            }
            if (mode === 'book')
                document.querySelector('.tab[data-panel="toc"]')?.click();
            const saved = anchor();
            if (layout) {
                layout.restore();
                layout = null;
            }
            state.mode = mode;
            document.body.dataset.readerMode = mode;
            footer.hidden = mode === 'document';
            Object.entries(modes).forEach(([key, b]) => b.setAttribute('aria-pressed', String(key === mode)));
            if (mode === 'document') {
                if (state.pendingRevision)
                    options.apply(state.pendingRevision);
                const main = root.querySelector('main.org-preview');
                if (main) {
                    main.style.removeProperty('--reader-font');
                    const records = P.describe(Array.from(main.children)), match = P.resolveAnchor(records, saved);
                    records[match.index]?.el.scrollIntoView({ block: 'start' });
                    if (match.fallback)
                        announce('읽던 문단을 찾지 못해 장 또는 문서 시작으로 이동했습니다.');
                }
            }
            else {
                const main = root.querySelector('main.org-preview');
                if (main && !main.querySelector('iframe')) {
                    layout = P.create(main);
                    reflow(saved);
                }
                else
                    announce('이 콘텐츠는 문서 보기에서 확인하세요.');
            }
            refresh();
        }
        function refresh() {
            update.hidden = !options.linked && !state.pendingRevision;
            select.hidden = state.mode !== 'book';
            source.hidden = state.mode === 'document';
            fullscreen.hidden = state.mode !== 'present';
            if (state.pendingRevision)
                announce('새 내용 있음');
        }
        function applyPending() {
            if (!state.pendingRevision)
                return;
            const saved = anchor();
            if (layout) {
                layout.restore();
                layout = null;
            }
            options.apply(state.pendingRevision);
            const main = root.querySelector('main.org-preview');
            if (state.mode !== 'document' && main) {
                layout = P.create(main);
                reflow(saved);
            }
            refresh();
        }
        function replaceDocument(apply) {
            const saved = anchor();
            if (layout) {
                layout.restore();
                layout = null;
            }
            apply();
            const main = root.querySelector('main.org-preview');
            if (state.mode !== 'document' && main && !main.querySelector('iframe')) {
                layout = P.create(main);
                reflow(saved);
            }
            refresh();
        }
        function changed() {
            refresh();
        }
        function schedule() {
            clearTimeout(timer);
            timer = global.setTimeout(() => reflow(), 80);
        }
        root.addEventListener('load', schedule, true);
        root.addEventListener('toggle', schedule, true);
        root.addEventListener('error', event => {
            if (event.target.tagName !== 'IMG')
                return;
            event.target.setAttribute('aria-label', '그림을 불러올 수 없습니다');
            announce(options.linked ? '그림을 불러올 수 없습니다. 다시 불러오기를 사용하세요.' : '그림을 불러올 수 없습니다. 최신 내용 적용을 사용하세요.');
            if (!options.linked)
                update.hidden = false;
        }, true);
        global.addEventListener('resize', schedule);
        if (document.fonts)
            document.fonts.ready.then(schedule);
        document.addEventListener('fullscreenchange', () => {
            fullscreen.textContent = document.fullscreenElement ? '전체 화면 닫기' : '전체 화면';
            schedule();
        });
        document.addEventListener('keydown', event => {
            if (state.mode === 'document' || event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey || !document.hasFocus())
                return;
            if (event.target.closest('input,textarea,select,[contenteditable="true"],pre,code,.reader-overflow,iframe'))
                return;
            if (event.key === 'Escape') {
                if (!document.fullscreenElement && state.mode === 'present') {
                    event.preventDefault();
                    setMode('book');
                }
                return;
            }
            const actions = { ArrowLeft: () => move(-spread), PageUp: () => move(-spread), ArrowRight: () => move(spread), PageDown: () => move(spread), Home: () => move(-Infinity), End: () => move(Infinity) };
            if (actions[event.key]) {
                event.preventDefault();
                actions[event.key]();
            }
        });
        refresh();
        modes.document.setAttribute('aria-pressed', 'true');
        return { state, setMode, changed, announce, applyPending, replaceDocument, rendered: schedule,
            jump(el) {
                if (!layout)
                    return false;
                index = Math.floor(layout.pageFor(el) / spread) * spread;
                show();
                return true;
            },
            guard() {
                if (state.canNavigate())
                    return true;
                announce('최신 내용 적용 후 이동하세요.');
                return false;
            } };
    }
    const api = { createRevisionState, create };
    if (typeof module !== 'undefined' && module.exports)
        module.exports = api;
    else
        global.OrgReaderView = api;
})(globalThis);
