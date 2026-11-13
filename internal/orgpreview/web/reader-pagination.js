(function (global) {
    'use strict';
    const heading = el => /^H[1-6]$/.test(el.tagName);
    const text = el => (el.textContent || '').replace(/\s+/g, ' ').trim();
    function spreadFor(width, preference, mode) {
        return mode === 'book' && preference !== 'single' && width >= 664 ? 2 : 1;
    }
    function describe(blocks) {
        const path = [];
        return blocks.map((el, i) => {
            if (heading(el)) {
                const level = Number(el.tagName.slice(1));
                path.length = level;
                path[level - 1] = text(el);
            }
            return { id: el.dataset.orgId || '', text: text(el), path: path.filter(Boolean).join(' / '), heading: heading(el),
                before: i ? text(blocks[i - 1]) : '', after: i + 1 < blocks.length ? text(blocks[i + 1]) : '', el };
        });
    }
    function resolveAnchor(records, anchor) {
        if (!anchor)
            return { index: 0, fallback: false };
        const sameElement = anchor.el ? records.findIndex(r => r.el === anchor.el) : -1;
        if (sameElement >= 0) return {index:sameElement,fallback:false};
        const candidates = records.map((r, i) => ({ r, i })).filter(({ r }) => r.text === anchor.text && r.path === anchor.path);
        const contextual = candidates.filter(({ r }) => r.before === anchor.before && r.after === anchor.after);
        if (contextual.length === 1)
            return { index: contextual[0].i, fallback: false };
        if (candidates.length === 1)
            return { index: candidates[0].i, fallback: false };
        const chapter = records.findIndex(r => r.heading && r.path === anchor.path);
        return { index: Math.max(0, chapter), fallback: true };
    }
    // Move original direct children only; nested source IDs remain inside their blocks.
    function create(main) {
        const blocks = Array.from(main.children), records = describe(blocks);
        let pages = [], wrappers = [], metadata = null;
        function restore() {
            blocks.forEach(el => main.appendChild(el));
            wrappers.forEach(el => el.remove());
            wrappers = [];
            if (metadata) {
                metadata.remove();
                metadata = null;
            }
            pages = [];
        }
        function paginate(options) {
            restore();
            const measure = main.ownerDocument.createElement('div');
            measure.className = 'reader-measure reader-page';
            measure.style.width = options.width + 'px';
            measure.setAttribute('aria-hidden', 'true');
            measure.inert = true;
            main.appendChild(measure);
            const content = blocks.filter(el => !el.matches('table.org-properties[data-org-format="yaml"]'));
            const heights = new Map();
            // All original blocks are measured in the live render tree, then restored synchronously.
            content.forEach(el => {
                const parent = el.parentNode, next = el.nextSibling;
                measure.appendChild(el);
                const style = main.ownerDocument.defaultView.getComputedStyle(el);
                heights.set(el, el.getBoundingClientRect().height + (parseFloat(style.marginTop) || 0) + (parseFloat(style.marginBottom) || 0));
                parent.insertBefore(el, next);
            });
            measure.remove();
            blocks.filter(el => !content.includes(el)).forEach(el => {
                if (!metadata) {
                    metadata = main.ownerDocument.createElement('details');
                    metadata.className = 'reader-metadata';
                    const summary = main.ownerDocument.createElement('summary');
                    summary.textContent = '문서 정보';
                    metadata.appendChild(summary);
                    main.appendChild(metadata);
                }
                metadata.appendChild(el);
            });
            // The detached measurement element has the same CSS inset as every page.
            const inset = 32;
            const metadataHeight = metadata && main.ownerDocument.defaultView.getComputedStyle(metadata).display !== 'none'
                ? metadata.getBoundingClientRect().height + 24 : 0;
            const pageHeight = Math.max(80, options.height - metadataHeight);
            const contentHeight = Math.max(48, pageHeight - inset);
            const levels = content.filter(heading).map(el => Number(el.tagName.slice(1)));
            const top = levels.length ? Math.min(...levels) : 0;
            pages = [];
            let page = [], used = 0;
            const flush = () => {
                if (page.length)
                    pages.push(page);
                page = [];
                used = 0;
            };
            content.forEach((el, i) => {
                const size = heights.get(el), next = content[i + 1];
                if (heading(el) && Number(el.tagName.slice(1)) === top)
                    flush();
                const together = heading(el) && next ? size + heights.get(next) : size;
                if (page.length && (used + size > contentHeight || together <= contentHeight && used + together > contentHeight))
                    flush();
                page.push(el);
                used += size;
                if (size > contentHeight)
                    flush();
            });
            flush();
            pages.forEach((items, i) => {
                const wrapper = main.ownerDocument.createElement('section');
                wrapper.className = 'reader-page';
                wrapper.style.width = options.width + 'px';
                wrapper.style.height = pageHeight + 'px';
                wrapper.setAttribute('aria-label', (i + 1) + '쪽');
                items.forEach(el => {
                    if (heights.get(el) > contentHeight) {
                        const scroll = main.ownerDocument.createElement('div');
                        scroll.className = 'reader-overflow';
                        scroll.setAttribute('tabindex', '0');
                        scroll.setAttribute('role', 'region');
                        scroll.setAttribute('aria-label', '아래 내용 더 있음 · 스크롤하여 읽기');
                        scroll.style.setProperty('--reader-block-height', contentHeight + 'px');
                        scroll.appendChild(el);
                        wrapper.appendChild(scroll);
                    }
                    else {
                        wrapper.appendChild(el);
                    }
                });
                main.appendChild(wrapper);
                wrappers.push(wrapper);
            });
            return pages;
        }
        function show(index, spread) {
            wrappers.forEach((el, i) => {
                el.hidden = i < index || i >= index + spread;
                el.inert = el.hidden;
            });
        }
        function pageFor(el) {
            return Math.max(0, pages.findIndex(page => page.some(block => block === el || block.contains(el))));
        }
        return { blocks, records, paginate, restore, show, pageFor, get pages() {
                return pages;
            } };
    }
    const api = { spreadFor, describe, resolveAnchor, create };
    if (typeof module !== 'undefined' && module.exports)
        module.exports = api;
    else
        global.OrgReaderPagination = api;
})(globalThis);
