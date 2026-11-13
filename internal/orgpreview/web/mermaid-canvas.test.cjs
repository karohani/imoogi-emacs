const test = require('node:test');
const assert = require('node:assert/strict');
const canvas = require('./mermaid-canvas.js');

test('document view fits a wide diagram to the column and never enlarges a small one', () => {
  assert.equal(canvas.fitWidth(2000, 800), 0.4);
  assert.equal(canvas.fitWidth(300, 800), 1);
  assert.equal(canvas.fitWidth(0, 800), 1);
});

test('fullscreen fits the whole diagram on screen and may enlarge it up to a limit', () => {
  assert.equal(canvas.fitContain(2000, 500, 1000, 1000), 0.5);
  assert.equal(canvas.fitContain(500, 2000, 1000, 1000), 0.5);
  assert.equal(canvas.fitContain(100, 100, 1000, 1000), 3);
});

test('zooming keeps the point under the cursor fixed and stays within limits', () => {
  const view = canvas.zoomAt({scale: 1, x: 0, y: 0}, 2, 100, 50);
  assert.deepEqual(view, {scale: 2, x: -100, y: -50});
  // The diagram point under (100,50) is the same before and after.
  assert.equal((100 - view.x) / view.scale, 100);
  assert.equal(canvas.zoomAt({scale: 7, x: 0, y: 0}, 4, 0, 0).scale, canvas.MAX_SCALE);
  assert.equal(canvas.zoomAt({scale: 0.2, x: 0, y: 0}, 0.1, 0, 0).scale, canvas.MIN_SCALE);
});

test('panning cannot lose the diagram and centers it when it is smaller than the box', () => {
  // 1000x400 content at scale 1 in a 500x400 box: x may range over [-500, 0].
  assert.deepEqual(canvas.clampPan({scale: 1, x: 300, y: 0}, 1000, 400, 500, 400), {scale: 1, x: 0, y: 0});
  assert.deepEqual(canvas.clampPan({scale: 1, x: -900, y: 0}, 1000, 400, 500, 400), {scale: 1, x: -500, y: 0});
  // 200x100 content in a 500x400 box is centered.
  assert.deepEqual(canvas.clampPan({scale: 1, x: -50, y: 70}, 200, 100, 500, 400), {scale: 1, x: 150, y: 150});
});

test('a diagram keeps its zoom across revisions by its position, even when its text changes', () => {
  assert.equal(canvas.keyFor(3), 'diagram:3');
});
