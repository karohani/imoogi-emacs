const test = require('node:test');
const assert = require('node:assert/strict');
const pagination = require('./reader-pagination.js');
const reader = require('./reader-view.js');
test('spread uses actual content width and explicit single-page preference', () => {
    assert.equal(pagination.spreadFor(663, 'auto', 'book'), 1);
    assert.equal(pagination.spreadFor(664, 'auto', 'book'), 2);
    assert.equal(pagination.spreadFor(1200, 'single', 'book'), 1);
    assert.equal(pagination.spreadFor(1200, 'auto', 'present'), 1);
});
test('anchor rejects shifted duplicate ID and falls back to chapter on ambiguous text', () => {
    const a = { id: 'p1', text: '같은 문단 😀', path: '장', before: '앞', after: '뒤' };
    assert.equal(pagination.resolveAnchor([{ ...a, before: '다름', after: '다름' }, { ...a, id: 'p2' }], a).index, 1);
    assert.equal(pagination.resolveAnchor([{ ...a, id: 'p2' }, { ...a, id: 'p3' }, { id: 'h', text: '장', path: '장', heading: true }], a).index, 2);
});
test('reading mode keeps displayed revision while retaining only newest pending revision', () => {
    const state = reader.createRevisionState();
    assert.equal(state.receive({ revision: 1, html: 'one' }), true);
    state.applied({ revision: 1 });
    state.mode = 'book';
    assert.equal(state.receive({ revision: 2, html: 'two' }), false);
    assert.equal(state.receive({ revision: 3, html: 'three' }), false);
    assert.equal(state.displayedRevision, 1);
    assert.equal(state.pendingRevision.html, 'three');
    assert.equal(state.canNavigate(), false);
    assert.equal(state.receive({ revision: 2, html: 'old' }), false);
    assert.equal(state.pendingRevision.revision, 3);
    state.applied(state.pendingRevision);
    assert.equal(state.canNavigate(), true);
});

test('a pending message does not invalidate the current render generation', () => {
    const state = reader.createRevisionState();
    state.receive({revision: 10, html: 'current'});
    state.applied({revision: 10});
    const currentRender = state.renderGeneration;
    state.mode = 'present';
    state.receive({revision: 11, html: 'next'});
    assert.equal(state.renderGeneration, currentRender);
    state.applied(state.pendingRevision);
    assert.equal(state.renderGeneration, currentRender + 1);
});

test('document revisions apply immediately and stale messages cannot replace them', () => {
    const state = reader.createRevisionState();
    assert.equal(state.receive({revision: 2, html: 'new'}), true);
    state.applied({revision: 2});
    assert.equal(state.receive({revision: 1, html: 'old'}), false);
    assert.equal(state.displayedRevision, 2);
    assert.equal(state.pendingRevision, null);
});

test('unmatched and deleted anchors use a verified chapter or document start', () => {
    const records = [{heading:true,path:'시작',text:'시작'}, {heading:true,path:'다음',text:'다음'}];
    assert.deepEqual(pagination.resolveAnchor(records, {path:'다음',text:'삭제됨'}), {index:1,fallback:true});
    assert.deepEqual(pagination.resolveAnchor(records, {path:'없음',text:'삭제됨'}), {index:0,fallback:true});
    assert.deepEqual(pagination.resolveAnchor(records, null), {index:0,fallback:false});
});

test('a follow tab moves only in document mode and offers the move while reading', () => {
    const msg = {type: 'current', buffer_id: 'b', url: '/preview?buffer=b'};
    assert.equal(reader.followAction({follow: true, bufferID: 'a', mode: 'document'}, msg), 'go');
    assert.equal(reader.followAction({follow: true, bufferID: 'a', mode: 'book'}, msg), 'offer');
    assert.equal(reader.followAction({follow: true, bufferID: 'a', mode: 'present'}, msg), 'offer');
    assert.equal(reader.followAction({follow: true, bufferID: 'b', mode: 'book'}, msg), 'stay');
    assert.equal(reader.followAction({follow: false, bufferID: 'a', mode: 'document'}, msg), 'ignore');
    assert.equal(reader.followAction({follow: true, bufferID: 'a', mode: 'document'}, {...msg, url: ''}), 'ignore');
    assert.equal(reader.followAction({follow: true, bufferID: 'a', mode: 'document'}, {html: 'x'}), 'ignore');
});

test('same-DOM reflow uses original element identity despite repeated context', () => {
    const first = {}, second = {};
    const common = {text:'반복',path:'장',before:'반복',after:'반복'};
    const records = [{...common,el:first},{...common,el:second}];
    assert.deepEqual(pagination.resolveAnchor(records, {...common,el:second}), {index:1,fallback:false});
});
