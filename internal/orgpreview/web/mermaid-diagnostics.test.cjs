const test = require('node:test');
const assert = require('node:assert/strict');
const diagnostics = require('./mermaid-diagnostics.js');
class Element {
  constructor(tag='div') { this.tagName=tag; this.children=[]; this.dataset={}; this.textContent=''; this.isConnected=true; this.classList={add(){},remove(){}}; }
  append(...children) { children.forEach(child=>{child.parent=this;child.isConnected=true;}); this.children.push(...children); }
  replaceChildren(...children) { this.children.forEach(child=>child.isConnected=false); this.children=[]; this.append(...children); this.textContent=''; }
  remove(){ this.isConnected=false; }
  setAttribute(k,v){ this[k]=v; }
  focus(){ this.focused=true; }
  select(){ this.selected=true; }
}
const document={createElement:tag=>new Element(tag),body:new Element('body')};
test('leading blank frontmatter is normalized for runtime but original error source is preserved', async () => {
  const node = new Element();
  const source = '\n\n---\ntitle: t\n---\nflowchart LR\nA-->';
  node.textContent = source;
  let parsed;
  await setup({parse: async text => {
    parsed = text;
    throw Object.assign(new Error('parser line 3'), {hash: {loc: {first_line: 2}}});
  }}).render([node], 1);
  assert.equal(parsed, source.trim());
  assert.equal(node.children[1].textContent, source);
  assert.doesNotMatch(node.children[0].textContent, /다이어그램 내부 2행/);
});
function setup(runtime, extra={}) { return diagnostics.create({document,runtime,version:'12.0.0',isCurrent:()=>true,canNavigate:()=>true,navigate:()=>true,...extra}); }
function buttons(node){return node.children.find(x=>x.tagName==='div' && x.dataset.mermaidControls).children;}
function click(button){return button.onclick({stopPropagation(){},preventDefault(){}});}
test('invalid sibling preserves source, gives safe syntax error and copy fallback; valid sibling renders',async()=>{
  const bad=new Element(), good=new Element(); bad.textContent='<script>bad</script>'; good.textContent='flowchart LR\nA-->B';
  let renders=0;
  const api=setup({parse:async text=>{if(text===bad.textContent)throw Object.assign(new Error('<img> syntax'),{hash:{loc:{first_line:2}}});},render:async()=>{renders++; return {svg:'<svg></svg>'};}});
  await api.render([bad,good],1);
  assert.equal(renders,1); assert.equal(good.children[0].innerHTML,'<svg></svg>');
  assert.equal(bad.children[1].textContent,'<script>bad</script>');
  assert.match(bad.children[0].textContent,/문법.*2.*<img>/);
  await click(buttons(bad)[2]);
  const fallback=bad.children.at(-1); assert.match(fallback.value,/Mermaid 12.0.0/); assert.match(fallback.value,/<script>bad/); assert.equal(fallback.selected,true);
});
test('retry uses original source after DOM mutation; controls stop bubbling',async()=>{
  const node=new Element();node.textContent='original';let calls=[];let navigated=0;
  const api=setup({parse:async text=>{calls.push(text);throw new Error('oops');}},{navigate:()=>{navigated++;return true;}});
  await api.render([node],1); node.textContent='mutated';
  await click(buttons(node)[1]); assert.deepEqual(calls,['original','original']);
  click(buttons(node)[0]);assert.equal(navigated,1);
});
test('render failures differ from syntax failures; stale completion ignored and scratch removed',async()=>{
  let resolve;let current=true;const node=new Element();node.textContent='source';
  const api=setup({parse:async()=>{},render:()=>new Promise(r=>resolve=r)},{isCurrent:()=>current});
  const pending=api.render([node],1); await new Promise(r=>setImmediate(r)); current=false;resolve({svg:'stale'}); await pending;
  assert.equal(node.children.length,0);assert.equal(document.body.children.at(-1).isConnected,false);
  const failure=new Element();failure.textContent='valid'; await setup({parse:async()=>{},render:async()=>{throw new Error('DOM unavailable');}}).render([failure],1);
  assert.match(failure.children[0].textContent,/렌더링/);
});
test('linked snapshots disable navigation and denied clipboard has manual fallback',async()=>{
  const node=new Element();node.textContent='bad';
  await setup({parse:async()=>{throw new Error('bad');}},{canNavigate:()=>false,clipboard:{writeText:async()=>{throw new Error('denied');}}}).render([node],1);
  assert.equal(buttons(node)[0].disabled,true);await click(buttons(node)[2]); assert.equal(node.children.at(-1).readOnly,true);
});
test('clipboard success copies exact source; unknown locations omitted; controls consume click',async()=>{
  const node=new Element(); node.textContent='flowchart LR\nA["<script>alert(1)</script>"]';let copied, stops=0, prevents=0;
  await setup({parse:async()=>{throw Object.assign(new Error('Unexpected token'),{hash:{line:99}});}},{clipboard:{writeText:async text=>{copied=text;}}}).render([node],1);
  assert.doesNotMatch(node.children[0].textContent,/99|행/);
  await buttons(node)[2].onclick({stopPropagation(){stops++;},preventDefault(){prevents++;}});
  assert.equal(stops,1); assert.equal(prevents,1); assert.ok(copied.endsWith(node.children[1].textContent));
  assert.equal(node.children.some(child=>child.tagName==='textarea'),false);
});
test('a retry supersedes an in-flight attempt',async()=>{
  const node=new Element();node.textContent='source';let resolve, count=0;
  const api=setup({parse:async()=>{},render:async()=>{count++; if(count===1)return new Promise(r=>resolve=r);return {svg:'latest'};}});
  const first=api.render([node],1); await new Promise(r=>setImmediate(r));
  const retry=api.render([node],1);resolve({svg:'stale'});await first;
  assert.equal(node.children.length,0);await retry;assert.equal(node.children[0].innerHTML,'latest');
});
test('delayed clipboard success or rejection cannot change a retried diagram',async()=>{
  for (const denied of [false,true]) {
    const node=new Element();node.textContent='bad';let settle;
    const api=setup({parse:async()=>{throw new Error('bad');}},{clipboard:{writeText:()=>new Promise((resolve,reject)=>{settle=()=>denied?reject(new Error('denied')):resolve();})}});
    await api.render([node],1);const pending=click(buttons(node)[2]);
    await click(buttons(node)[1]);const before=node.children.length;settle();await pending;
    assert.equal(node.children.length,before);assert.equal(node.children.some(x=>x.tagName==='textarea'),false);
  }
});
test('revision replacement detaches controls and ignores delayed copy and render',async()=>{
  let revision=1,resolve;const root=new Element(), old=new Element();old.textContent='bad';root.append(old);
  const api=setup({parse:async text=>{if(text==='bad')throw new Error('bad');},render:async()=>({svg:'new'})},{isCurrent:r=>r===revision,clipboard:{writeText:()=>new Promise(r=>resolve=r)}});
  await api.render([old],1);const pending=click(buttons(old)[2]);const before=old.children.length;
  const next=new Element();next.textContent='good';revision=2;root.replaceChildren(next);await api.render([next],2);
  resolve();await pending;assert.equal(old.children.length,before);assert.equal(next.children[0].innerHTML,'new');
});
test('navigation and repeated copy reuse a single status message',async()=>{
  const node=new Element();node.textContent='bad';
  await setup({parse:async()=>{throw new Error('bad');}},{navigate:()=>false,clipboard:{writeText:async()=>{}}}).render([node],1);
  click(buttons(node)[0]);click(buttons(node)[0]);await click(buttons(node)[2]);await click(buttons(node)[2]);
  const statuses=node.children.filter(x=>x.className==='mermaid-control-status');assert.equal(statuses.length,1);assert.equal(statuses[0].textContent,'수정 요청을 복사했습니다.');
});
test('frontmatter, comments and init directives do not claim preprocessed positions as source lines',async()=>{
  for(const source of ['---\ntitle: Example\n---\nflowchart LR\nA-->', '%%{init: {"theme":"dark"}}%%\nflowchart LR\nA-->', 'flowchart LR\n%% note\nA-->']) {
    const node=new Element();node.textContent=source;
    await setup({parse:async()=>{throw Object.assign(new Error('parse location unavailable'),{hash:{loc:{first_line:2}}});}}).render([node],1);
    assert.doesNotMatch(node.children[0].textContent,/내부 2행/);assert.equal(node.children[1].textContent,source);
  }
});
test('trailing whitespace does not suppress available source line numbers',async()=>{
  const node=new Element();node.textContent='flowchart LR\nA-->\n';
  await setup({parse:async()=>{throw Object.assign(new Error('bad'),{hash:{loc:{first_line:2}}});}}).render([node],1);
  assert.match(node.children[0].textContent,/내부 2행/);
});
