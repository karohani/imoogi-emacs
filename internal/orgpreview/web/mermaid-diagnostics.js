/* Runs with the bundled Mermaid runtime; never contacts an external service. */
(function (global) {
  'use strict';
  function create(options) {
    const {document, runtime, version, isCurrent, canNavigate, navigate, clipboard} = options;
    const states = new WeakMap();
    let queue = Promise.resolve();
    let sequence = 0;
    function element(tag, text) {
      const el = document.createElement(tag);
      if (text !== undefined) el.textContent = text;
      return el;
    }
    function current(node, state, attempt, revision) {
      return node.isConnected && state.attempt === attempt && isCurrent(revision);
    }
    function errorInfo(error, stage, source) {
      // Parser locations are relative to the diagram, never Org byte offsets.
      const candidate = error && error.hash && error.hash.loc && error.hash.loc.first_line;
      const preprocessed = /^[ \t\r]*\n/.test(source) || /^\s*---(?:\r?\n|$)/.test(source) || /^\s*%%/m.test(source);
      const line = stage === 'parse' && !preprocessed && Number.isInteger(candidate) && candidate > 0 ? candidate : null;
      const message = error && error.message ? String(error.message) : String(error);
      return {line, message, stage, locationNote: stage === 'parse' && preprocessed};
    }
    function requestText(state) {
      const failure = state.error;
      return '내장 Mermaid ' + version + '에서 다음 다이어그램이 실패했습니다.\n' +
        (failure.stage === 'parse' ? '문법 오류' : '렌더링 오류 (문법 검사는 통과함)') +
        (failure.line ? ' · 다이어그램 내부 ' + failure.line + '행' : '') + '\n' + failure.message +
        (failure.locationNote ? '\n파서 메시지의 행 번호는 원문과 다를 수 있습니다.' : '') +
        '\n\n의미를 유지하며 이 버전에서 동작하도록 수정하고 Mermaid 코드만 반환하세요. ' +
        '단순한 노드 ID와 따옴표로 감싼 표시 문구를 사용하세요. 렌더링 오류라면 실행 환경 문제인지도 구분하세요.\n\n' + state.source;
    }
    function controls(node, state, revision) {
      const attempt = state.attempt;
      const bar = element('div');
      const status = element('p');
      status.className = 'mermaid-control-status'; status.setAttribute('role', 'status');
      let copyRequest = 0;
      const active = () => current(node, state, attempt, revision) && bar.isConnected;
      function setStatus(text) { if (active()) status.textContent = text; }
      bar.className = 'mermaid-controls';
      bar.dataset.mermaidControls = 'true';
      function button(label, action) {
        const b = element('button', label); b.type = 'button';
        b.onclick = event => { event.preventDefault(); event.stopPropagation(); if (active()) return action(); };
        bar.append(b); return b;
      }
      const source = button('원본으로 이동', () => {
        if (!isCurrent(revision) || !navigate(node)) {
          setStatus('원본 연결이 끊어졌습니다. 원본 버퍼의 미리보기에서 다시 시도하세요.');
        }
      });
      source.disabled = !canNavigate();
      if (source.disabled) source.title = '저장된 링크 미리보기에서는 원본 버퍼로 이동할 수 없습니다.';
      button('다시 검사', () => run(node, state, revision));
      button('LLM 수정 요청 복사', async () => {
        const text = requestText(state);
        const request = ++copyRequest;
        try {
          if (!clipboard || !clipboard.writeText) throw new Error('clipboard unavailable');
          await clipboard.writeText(text);
          if (!active() || request !== copyRequest) return;
          if (state.copyField) { state.copyField.remove(); state.copyField = null; }
          setStatus('수정 요청을 복사했습니다.');
        } catch (_) {
          if (!active() || request !== copyRequest) return;
          if (state.copyField) state.copyField.remove();
          setStatus('자동 복사를 사용할 수 없습니다. 아래 내용을 선택해 직접 복사하세요.');
          const field = element('textarea');
          field.className = 'mermaid-copy'; field.readOnly = true; field.value = text;
          field.setAttribute('aria-label', 'LLM 수정 요청'); field.dataset.mermaidControls = 'true';
          node.append(field); state.copyField = field; field.focus(); field.select();
        }
      });
      node.append(bar, status);
      if (source.disabled) setStatus('원본 이동은 원본 버퍼의 미리보기에서 사용할 수 있습니다.');
    }
    function run(node, state, revision) {
      const attempt = ++state.attempt;
      const work = async () => {
        if (!current(node, state, attempt, revision)) return;
        let stage = 'parse', scratch;
        try {
          const input = state.source.trim();
          await runtime.parse(input);
          if (!current(node, state, attempt, revision)) return;
          stage = 'render';
          scratch = element('div'); scratch.style = 'position:absolute;left:-100000px;top:0';
          scratch.setAttribute('aria-hidden', 'true'); document.body.append(scratch);
          const result = await runtime.render('org_mermaid_' + (++sequence), input, scratch);
          if (!current(node, state, attempt, revision)) return;
          const diagram = element('div'); diagram.innerHTML = result.svg;
          node.replaceChildren(diagram); node.classList.remove('mermaid-error');
          if (result.bindFunctions) result.bindFunctions(diagram);
        } catch (error) {
          if (!current(node, state, attempt, revision)) return;
          state.error = errorInfo(error, stage, state.source);
          const info = state.error;
          const message = element('p', (stage === 'parse' ? 'Mermaid 문법 오류' : 'Mermaid 렌더링 오류 (문법 검사는 통과함)') +
            (info.line ? ' · 다이어그램 내부 ' + info.line + '행' : '') + ': ' + info.message +
            (info.locationNote ? '\n파서 메시지의 행 번호는 원문과 다를 수 있습니다.' : ''));
          message.setAttribute('role', 'status');
          const source = element('pre', state.source);
          node.replaceChildren(message, source); node.classList.add('mermaid-error');
          controls(node, state, revision);
        } finally {
          if (scratch) scratch.remove();
        }
      };
      // Mermaid uses shared renderer state. Serialize both revisions and retries.
      const pending = queue.then(work); queue = pending.catch(() => {}); return pending;
    }
    return {
      render(nodes, revision) {
        // Capture every source before Mermaid can mutate any diagram DOM.
        const captured = Array.from(nodes, node => {
          let state = states.get(node);
          if (!state) { state = {source: node.textContent, attempt: 0}; states.set(node, state); }
          return [node, state];
        });
        return Promise.all(captured.map(([node, state]) => run(node, state, revision)));
      }
    };
  }
  const api = {create};
  if (typeof module === 'object' && module.exports) module.exports = api;
  else global.OrgMermaidDiagnostics = api;
})(globalThis);
