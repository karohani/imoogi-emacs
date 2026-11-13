const {chromium} = await import(process.env.PLAYWRIGHT_MODULE || '/Users/jay/.npm/_npx/9833c18b2d85bc59/node_modules/playwright/index.mjs');
import {spawn} from 'node:child_process';
import {mkdtemp,writeFile,rm} from 'node:fs/promises';
import {once} from 'node:events';
import assert from 'node:assert/strict';
const fixture=await mkdtemp('/private/tmp/reader-fixture-');
await writeFile(fixture+'/image.svg','<svg xmlns="http://www.w3.org/2000/svg" width="200" height="100"><rect width="200" height="100" fill="red"/></svg>');
const server=spawn(process.env.BINARY || '/private/tmp/imoogi-reader-check',['-token','reader-test-token','-print-bootstrap-json'],{stdio:['ignore','pipe','pipe']});
let browser;
try {
 const [first]=await once(server.stdout,'data');const config=JSON.parse(first.toString().trim());const base='http://'+config.addr;
 let rev=0;
 const source='---\nname: reader test\n---\n# 시작\n\n![image](image.svg)\n\n'+Array.from({length:30},(_,i)=>'문단 '+i+' 한글 😀 '+('긴 내용입니다. '.repeat(25))).join('\n\n')+'\n\n# 다음 장\n\n```text\n'+('long code\n'.repeat(100))+'```\n\n끝 문단';
 const publish=async(text)=>{const r=await fetch(base+'/api/emacs/revisions',{method:'POST',headers:{'Content-Type':'application/json','X-Org-Preview-Token':'reader-test-token'},body:JSON.stringify({version:'org-preview/v1',session_id:'reader',buffer_id:'buffer',revision:++rev,event_id:'r'+rev,origin:'emacs',generation:1,path:fixture+'/test.md',syntax:'markdown',text,allowed_roots:[fixture]})});assert.equal(r.status,202);return r.json();};
 const response=await publish(source);
 browser=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
 const page=await browser.newPage({viewport:{width:1440,height:1000}});const errors=[];page.on('pageerror',e=>errors.push(e.message));let assets=0;page.on('request',r=>{if(new URL(r.url()).pathname==='/asset')assets++;});
 await page.goto(base+'/?session=reader&buffer=buffer&generation=1&view='+response.browser_view);
 await page.waitForSelector('#root img');await page.waitForFunction(()=>document.querySelector('#root img').complete);
 await page.evaluate(()=>{window.originalBlocks=Array.from(document.querySelector('#root main').children);window.originalImg=document.querySelector('#root img');});
 await page.getByRole('button',{name:'책자',exact:true}).click();
 await page.waitForTimeout(250);
 console.log('BOOK',await page.evaluate(()=>({pages:document.querySelectorAll('.reader-page:not(.reader-measure)').length,visible:[...document.querySelectorAll('.reader-page')].filter(e=>!e.hidden).length,sameImage:originalImg===document.querySelector('#root img'),text:document.querySelector('.reader-footer').textContent,overflow:[...document.querySelectorAll('.reader-overflow')].map(e=>({height:e.getBoundingClientRect().height,scroll:e.scrollHeight,client:e.clientHeight}))})));
 await assertViewport('book');
 await page.screenshot({path:'/private/tmp/reader-book.png',fullPage:true});
 await publish(source+'\n\nNEW REVISION');await page.getByText('새 내용 있음',{exact:true}).waitFor();assert.equal(await page.locator('#root').textContent().then(x=>x.includes('NEW REVISION')),false);
 console.log('PENDING retained DOM',await page.evaluate(()=>originalImg===document.querySelector('#root img')));
 await page.getByRole('button',{name:'최신 내용 적용',exact:true}).click();assert.ok((await page.locator('#root').textContent()).includes('NEW REVISION'));
 await page.getByRole('button',{name:'발표',exact:true}).click();await page.waitForTimeout(150);await assertViewport('present');await page.screenshot({path:'/private/tmp/reader-present.png',fullPage:true});
 await page.getByRole('button',{name:'문서',exact:true}).click();console.log('DOCUMENT wrappers',await page.locator('#root .reader-page').count());
 await page.setViewportSize({width:600,height:800});await page.getByRole('button',{name:'책자',exact:true}).click();await page.waitForTimeout(250);await assertViewport('narrow');await page.screenshot({path:'/private/tmp/reader-narrow.png',fullPage:true});
 console.log('NARROW',await page.evaluate(()=>({visible:[...document.querySelectorAll('.reader-page')].filter(e=>!e.hidden).length,bodyWidth:document.body.scrollWidth,viewport:innerWidth})));
 await page.keyboard.press('End');
 await page.waitForTimeout(100);
 const scroll=page.locator('.reader-page:not([hidden]) .reader-overflow');
 // The oversized code block gets its own page, so the closing paragraph follows it; walk back to it.
 for (let i=0;i<20 && await scroll.count()===0;i++) {await page.keyboard.press('ArrowLeft');await page.waitForTimeout(50);}
 assert.ok(await scroll.count()>0);
 const overflow=await scroll.first().evaluate(el=>({client:el.clientHeight,scroll:el.scrollHeight,bottom:el.getBoundingClientRect().bottom}));
 assert.ok(overflow.scroll>overflow.client);assert.ok(overflow.bottom<=800);
 await scroll.first().focus();
 const previousFooter=await page.locator('.reader-footer').textContent();
 await page.keyboard.press('PageDown');
 assert.equal(await page.locator('.reader-footer').textContent(),previousFooter);
 console.log('OVERFLOW',overflow);
 console.log('assets',assets,'errors',errors);assert.deepEqual(errors,[]);
 async function assertViewport(mode) {
 const actual=await page.evaluate(()=>({footer:document.querySelector('.reader-footer').getBoundingClientRect().bottom,height:innerHeight,body:document.body.scrollHeight}));
 console.log('VIEWPORT',mode,actual);
 assert.ok(actual.footer<=actual.height+1,mode+' footer must fit viewport');
 assert.ok(actual.body<=actual.height+1,mode+' reading layout must fit viewport');
 }
} finally {await browser?.close();server.kill('SIGINT');await once(server,'exit');await rm(fixture,{recursive:true,force:true});}
