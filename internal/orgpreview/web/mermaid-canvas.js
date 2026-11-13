/* Zoom and pan canvas for rendered Mermaid diagrams in the document view. */
(function (global) {
  'use strict';
  const MIN_SCALE = 0.1, MAX_SCALE = 8, FULLSCREEN_MAX = 3, STEP = 1.25;
  const clampScale = s => Math.min(MAX_SCALE, Math.max(MIN_SCALE, s));
  // Fit the column width; a diagram that already fits keeps its natural size.
  function fitWidth(contentWidth, viewWidth) {
    return contentWidth > 0 ? Math.min(1, viewWidth / contentWidth) : 1;
  }
  function fitContain(contentWidth, contentHeight, viewWidth, viewHeight) {
    return Math.min(FULLSCREEN_MAX, viewWidth / contentWidth, viewHeight / contentHeight);
  }
  // Scale around (px, py) so the diagram point under it stays put.
  function zoomAt(view, factor, px, py) {
    const scale = clampScale(view.scale * factor), ratio = scale / view.scale;
    return {scale, x: px - (px - view.x) * ratio, y: py - (py - view.y) * ratio};
  }
  // Keep the diagram reachable: center an axis that fits, otherwise forbid empty edges.
  function clampPan(view, contentWidth, contentHeight, viewWidth, viewHeight) {
    const axis = (pos, size, room) => size <= room ? (room - size) / 2 : Math.min(0, Math.max(room - size, pos));
    return {scale: view.scale, x: axis(view.x, contentWidth * view.scale, viewWidth),
      y: axis(view.y, contentHeight * view.scale, viewHeight)};
  }
  // Block ids hash the diagram text, so position is the key that survives editing the diagram.
  function keyFor(index) {
    return 'diagram:' + index;
  }
  function naturalSize(svg) {
    const box = svg.viewBox && svg.viewBox.baseVal;
    if (box && box.width > 0 && box.height > 0) return [box.width, box.height];
    const rect = svg.getBoundingClientRect();
    return [rect.width || 1, rect.height || 1];
  }
  function create(options) {
    const {document, root} = options, win = options.window || global;
    // Zoom survives re-rendering on every Emacs revision.
    const saved = new Map(), entries = new WeakMap(), observed = new Set();
    let expandedKey = null;
    const observer = typeof win.ResizeObserver === 'function' ?
      new win.ResizeObserver(records => records.forEach(r => { const e = entries.get(r.target); if (e && active(e)) refit(e); })) : null;
    const expanded = e => e.node.classList.contains('mermaid-expanded');
    // Book and present views lay diagrams out themselves; the canvas only acts in the document view.
    const active = e => expanded(e) || (document.body.dataset.readerMode || 'document') === 'document';
    function button(parent, label, title, fn) {
      const b = document.createElement('button');
      b.type = 'button'; b.textContent = label; b.title = title; b.setAttribute('aria-label', title);
      b.onclick = event => { event.stopPropagation(); fn(); };
      parent.appendChild(b); return b;
    }
    function apply(e, zoomed) {
      const v = e.view;
      e.stage.style.transform = 'translate(' + v.x + 'px,' + v.y + 'px) scale(' + v.scale + ')';
      e.zoom.textContent = Math.round(v.scale * 100) + '%';
      if (zoomed !== undefined) e.zoomed = zoomed;
      saved.set(e.key, {view: v, zoomed: e.zoomed});
    }
    function size(e) { return [e.viewport.clientWidth, e.viewport.clientHeight]; }
    function fit(e) {
      const [vw] = size(e);
      if (vw <= 0) return;
      let scale;
      if (expanded(e)) {
        scale = fitContain(e.width, e.height, vw, size(e)[1]);
      } else {
        scale = fitWidth(e.width, vw);
        e.fitHeight = Math.max(60, Math.ceil(e.height * scale));
        e.viewport.style.height = e.fitHeight + 'px';
      }
      const [w, h] = size(e);
      e.view = clampPan({scale, x: 0, y: 0}, e.width, e.height, w, h);
      apply(e, false);
    }
    function refit(e) {
      if (!e.zoomed) return fit(e);
      const [w, h] = size(e);
      e.view = clampPan(e.view, e.width, e.height, w, h); apply(e);
    }
    // Zooming in grows a short box (up to 70% of the window); fit shrinks it back.
    function grow(e, scale) {
      if (expanded(e) || !e.fitHeight) return;
      const wanted = Math.min(Math.ceil(e.height * scale), Math.floor(win.innerHeight * 0.7));
      e.viewport.style.height = Math.max(e.fitHeight, wanted) + 'px';
    }
    function zoomBy(e, factor, px, py) {
      const [w0, h0] = size(e);
      grow(e, clampScale(e.view.scale * factor));
      const view = zoomAt(e.view, factor, px === undefined ? w0 / 2 : px, py === undefined ? h0 / 2 : py);
      const [w, h] = size(e);
      e.view = clampPan(view, e.width, e.height, w, h); apply(e, true);
    }
    // The large view lives in the browser's top layer (a manual popover), so no z-index or
    // stacking context on the page can paint over it.
    function layer(node, on) {
      node.classList.toggle('mermaid-expanded', on);
      if (on) {
        node.setAttribute('popover', 'manual');
        try { if (!node.matches(':popover-open')) node.showPopover(); } catch (_) { /* class fallback still covers the window */ }
      } else if (node.hasAttribute('popover')) {
        try { if (node.matches(':popover-open')) node.hidePopover(); } catch (_) { /* already closed */ }
        node.removeAttribute('popover');
      }
    }
    function setExpanded(e, on) {
      layer(e.node, on);
      expandedKey = on ? e.key : null;
      win.requestAnimationFrame(() => fit(e));
    }
    function current() {
      const node = expandedKey && root.querySelector('.mermaid-expanded .mermaid-viewport');
      return node && entries.get(node);
    }
    function attach(node) {
      const diagram = node.firstElementChild, svg = diagram && diagram.querySelector('svg');
      if (!svg || node.classList.contains('mermaid-error')) return;
      const index = Array.prototype.indexOf.call(root.querySelectorAll('.mermaid'), node);
      const [width, height] = naturalSize(svg);
      svg.removeAttribute('width'); svg.removeAttribute('height');
      svg.style.maxWidth = 'none'; svg.style.width = width + 'px'; svg.style.height = height + 'px';
      svg.style.setProperty('--mermaid-w', width + 'px');
      const toolbar = document.createElement('div');
      toolbar.className = 'mermaid-toolbar'; toolbar.dataset.mermaidControls = 'canvas';
      toolbar.setAttribute('role', 'toolbar'); toolbar.setAttribute('aria-label', '다이어그램 확대');
      const viewport = document.createElement('div');
      viewport.className = 'mermaid-viewport'; viewport.tabIndex = 0;
      viewport.setAttribute('aria-label', '다이어그램 · 드래그로 이동, ⌘/Ctrl+휠로 확대');
      const stage = document.createElement('div');
      stage.className = 'mermaid-stage'; stage.style.width = width + 'px'; stage.style.height = height + 'px';
      stage.appendChild(diagram); viewport.appendChild(stage);
      const zoom = document.createElement('span'); zoom.className = 'mermaid-zoom';
      const e = {node, viewport, stage, zoom, width, height, key: keyFor(index), view: {scale: 1, x: 0, y: 0}, zoomed: false};
      button(toolbar, '−', '축소', () => zoomBy(e, 1 / STEP));
      toolbar.appendChild(zoom);
      button(toolbar, '+', '확대', () => zoomBy(e, STEP));
      button(toolbar, '맞춤', '폭에 맞춤', () => fit(e));
      button(toolbar, '1:1', '원래 크기', () => zoomBy(e, 1 / e.view.scale));
      button(toolbar, '⛶', '크게 보기', () => setExpanded(e, !expanded(e)));
      node.replaceChildren(toolbar, viewport);
      entries.set(viewport, e);
      if (expandedKey === e.key) layer(node, true); // an Emacs edit replaced the open diagram
      const previous = saved.get(e.key); // read before fit() records the fitted view
      fit(e);
      if (previous && previous.zoomed) {
        grow(e, previous.view.scale);
        const [w, h] = size(e);
        e.view = clampPan(previous.view, width, height, w, h); apply(e, true);
      }
      wire(e);
      if (observer) {
        observed.forEach(v => { if (!v.isConnected) { observer.unobserve(v); observed.delete(v); } });
        observer.observe(viewport); observed.add(viewport);
      }
    }
    function wire(e) {
      const {viewport} = e;
      let drag = null, suppressClick = false;
      viewport.addEventListener('wheel', event => {
        if (!active(e) || (!event.ctrlKey && !event.metaKey)) return; // plain wheel keeps scrolling the page
        event.preventDefault();
        const r = viewport.getBoundingClientRect(), delta = Math.max(-50, Math.min(50, event.deltaY));
        zoomBy(e, Math.exp(-delta * 0.004), event.clientX - r.left, event.clientY - r.top);
      }, {passive: false});
      viewport.addEventListener('pointerdown', event => {
        suppressClick = false;
        if (event.button !== 0 || !active(e)) return;
        drag = {id: event.pointerId, x: event.clientX, y: event.clientY, view: e.view, moved: false};
      });
      viewport.addEventListener('pointermove', event => {
        if (!drag || drag.id !== event.pointerId) return;
        const dx = event.clientX - drag.x, dy = event.clientY - drag.y;
        if (!drag.moved && Math.hypot(dx, dy) < 4) return;
        if (!drag.moved) { drag.moved = true; viewport.setPointerCapture(event.pointerId); viewport.classList.add('dragging'); }
        const [w, h] = size(e);
        const view = clampPan({scale: drag.view.scale, x: drag.view.x + dx, y: drag.view.y + dy}, e.width, e.height, w, h);
        const changed = view.x !== e.view.x || view.y !== e.view.y;
        e.view = view; apply(e, changed ? true : undefined);
      });
      const end = event => {
        if (!drag || drag.id !== event.pointerId) return;
        suppressClick = drag.moved; drag = null; viewport.classList.remove('dragging');
      };
      viewport.addEventListener('pointerup', end);
      viewport.addEventListener('pointercancel', end);
      // A drag must not count as a click that moves Emacs to the diagram source.
      viewport.addEventListener('click', event => {
        if (!suppressClick) return;
        suppressClick = false; event.stopPropagation(); event.preventDefault();
      });
      viewport.addEventListener('keydown', event => {
        const actions = {'+': () => zoomBy(e, STEP), '=': () => zoomBy(e, STEP), '-': () => zoomBy(e, 1 / STEP), '0': () => fit(e)};
        if (!actions[event.key] || !active(e) || event.altKey || event.ctrlKey || event.metaKey) return;
        event.preventDefault(); actions[event.key]();
      });
    }
    document.addEventListener('keydown', event => {
      const e = current();
      if (e && event.key === 'Escape') { event.preventDefault(); setExpanded(e, false); }
    });
    return {attach};
  }
  const api = {MIN_SCALE, MAX_SCALE, fitWidth, fitContain, zoomAt, clampPan, keyFor, create};
  if (typeof module === 'object' && module.exports) module.exports = api;
  else global.OrgMermaidCanvas = api;
})(globalThis);
