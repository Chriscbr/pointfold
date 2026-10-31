import assert from 'node:assert/strict';
import { chromium } from 'playwright';
import { pathToFileURL } from 'node:url';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { mkdir } from 'node:fs/promises';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const screenshots = process.env.POINTFOLD_SCREENSHOTS || resolve(root, '.qa');
await mkdir(screenshots, {recursive:true});
const browser = await chromium.launch({...(process.env.POINTFOLD_CHROME ? {executablePath:process.env.POINTFOLD_CHROME} : {}),headless:true});
try {
const context = await browser.newContext({viewport:{width:1280,height:950},colorScheme:'light',permissions:['clipboard-read','clipboard-write']});
const page = await context.newPage();
const errors = [], remoteRequests = [];
const peleURL='https://cdn.jsdelivr.net/npm/pele@1.13.0/dist/pele.global.js';
page.on('pageerror', e => errors.push(e.message));
page.on('request', req => {if (/^https?:/.test(req.url())) remoteRequests.push(req.url());});
await page.goto(pathToFileURL(root+'/examples/technical-decision.html').href);
assert.equal(await page.locator('.point').count(),38);
assert.equal(await page.locator('details[open]').count(),0);
assert.equal(await page.locator('.point:visible').count(),6);
assert.equal(await page.title(),'Should Larch move from Rust to Go?');
assert.equal(await page.locator('header, footer, .instructions, .stats, .eyebrow').count(),0);
const roots=await page.locator('#outline > .points > .point').evaluateAll(points=>points.map(point=>point.getBoundingClientRect().height));
assert(roots.every(height=>height<30),'Root points should use compact, line-sized rows');
await page.screenshot({path:screenshots+'/preview.png',fullPage:true});
await page.locator('#p-3 > details > summary').click();
assert.equal(await page.locator('#p-3 > details').getAttribute('open'),'');
assert(await page.locator('[id="p-3.1"]').isVisible());

// Verify an actual browser clipboard round trip from file:// before mocking denial paths.
await page.locator('#p-3 > details > summary .copy-point').click();
assert.equal(await page.evaluate(()=>navigator.clipboard.readText()),'[3]');
await page.locator('#p-3 > details > summary > .copy-markdown').click();
assert.equal(await page.evaluate(()=>navigator.clipboard.readText()),'The largest likely cost is reproducing behavior that the current implementation already gets right.');
assert.equal(await page.locator('#p-3 > details').getAttribute('open'),'');
await page.locator('[id="p-3.2"] > .leaf-row').click({position:{x:180,y:8}});
assert.equal(await page.locator('.point.highlight').count(),1);
assert(await page.locator('[id="p-3.2"]').evaluate(node=>node.classList.contains('highlight')));

// Exercise successful copy while checking that a nested button never toggles its summary.
await page.evaluate(() => {window.copiedValue=null;Object.defineProperty(navigator,'clipboard',{configurable:true,value:{writeText:async v=>{window.copiedValue=v;}}});});
await page.locator('#p-3 > details > summary .copy-point').click();
assert.equal(await page.evaluate(()=>window.copiedValue),'[3]');
assert.equal(await page.locator('#p-3 > details').getAttribute('open'),'');

// Force the file:// fallback and observe the actual selected reference.
await page.evaluate(() => {Object.defineProperty(navigator,'clipboard',{configurable:true,value:{writeText:async()=>{throw Error('denied');}}});window.execCopyValue=null;window.originalExec=document.execCommand;document.execCommand=(command)=>{window.execCopyValue=document.activeElement.value;return command==='copy';};});
await page.locator('#p-3 > details > summary .copy-point').click();
assert.equal(await page.evaluate(()=>window.execCopyValue),'[3]');

// An unavailable clipboard must produce an honest selectable manual fallback.
await page.evaluate(()=>document.execCommand=()=>false);
await page.locator('#p-3 > details > summary .copy-point').click();
assert(await page.locator('#copy-dialog').isVisible());
assert.equal(await page.locator('#copy-value').inputValue(),'[3]');
await page.locator('#copy-dialog button').click();
await page.evaluate(()=>document.execCommand=window.originalExec);

await page.locator('#collapse-all').click();
assert.equal(await page.locator('details[open]').count(),0);
await page.locator('#expand-all').click();
assert.equal(await page.locator('details[open]').count(),await page.locator('details').count());
await page.locator('#collapse-all').click();
await page.locator('#p-5 > details > summary').click();
const xref=page.locator('[id="p-5.4"] > .leaf-row a.xref');
assert(await xref.isVisible());
await xref.click();
assert.equal(new URL(page.url()).hash,'#p-3');
assert(await page.locator('#p-3.highlight').isVisible());
await page.locator('#return-point').click();
assert.equal(new URL(page.url()).hash,'#p-5.4');

// Direct fragments reveal deeply nested points even when their ancestors start closed.
await page.goto(pathToFileURL(root+'/examples/technical-decision.html').href+'#p-3.1.1.1.1');
await page.reload();
await page.waitForFunction(()=>document.getElementById('p-3.1.1.1.1').getBoundingClientRect().height>0);
assert(await page.locator('[id="p-3.1.1.1.1"]').isVisible());
await page.screenshot({path:screenshots+'/preview-expanded.png',fullPage:true});

// Keyboard activation works on the native summary.
await page.locator('#collapse-all').click();
const summary=page.locator('#p-2 > details > summary');
await summary.focus();await page.keyboard.press('Enter');
assert.equal(await page.locator('#p-2 > details').getAttribute('open'),'');
await page.keyboard.press('Space');
assert.equal(await page.locator('#p-2 > details').getAttribute('open'),null);

// Native Markdown formatting and compact mobile layout should remain intact.
await page.setViewportSize({width:390,height:844});
await page.goto(pathToFileURL(root+'/examples/math-proof.html').href);
await page.locator('#expand-all').click();
assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
assert(Number(await page.locator('strong').first().evaluate(node=>getComputedStyle(node).fontWeight))>=600);
assert((await page.locator('code').first().evaluate(node=>getComputedStyle(node).fontFamily)).includes('monospace'));
await page.screenshot({path:screenshots+'/preview-mobile.png',fullPage:true});
await page.emulateMedia({colorScheme:'dark'});
await page.screenshot({path:screenshots+'/preview-dark.png',fullPage:true});

for(const name of ['binary-search','codebase-tour','discussion-summary','math-proof','policy-debate','research-brief','technical-decision','visual-blocks']){
  await page.goto(pathToFileURL(root+'/examples/'+name+'.html').href);
  assert.equal(await page.locator('details[open]').count(),0);
  for(const link of await page.locator('a.xref').all()){
    const id=(await link.getAttribute('href')).slice(1);
    assert.equal(await page.evaluate(id=>!!document.getElementById(id),id),true);
  }
}

await page.setViewportSize({width:1280,height:950});
await page.goto(pathToFileURL(root+'/examples/visual-blocks.html').href);
assert.equal(await page.locator('.copy-markdown').count(),await page.locator('.point').count());
assert.equal(await page.locator('.diagram svg').count(),0);
assert.equal(await page.locator('.point-content:visible').count(),0);
assert.deepEqual(remoteRequests,[]);
assert.equal(await page.evaluate(()=>typeof window.pele),'undefined');
await page.locator('#p-2 > details > summary').click({position:{x:180,y:8}});
assert(await page.locator('#p-2 table').isVisible());
assert.equal(await page.locator('.point.highlight').count(),1);
assert(await page.locator('#p-2').evaluate(node=>node.classList.contains('highlight')));
const markdownButton=page.locator('#p-2 > details > summary > .copy-markdown');
await markdownButton.click();
const copiedMarkdown=await page.evaluate(()=>navigator.clipboard.readText());
assert(copiedMarkdown.startsWith('**Comparison.**'));
assert(copiedMarkdown.includes('| Policy | First delay |'));
assert(!copiedMarkdown.includes('rather than report measured service behavior'));
assert(!copiedMarkdown.includes('{#comparison}'));
// A denied clipboard must preserve newlines in the manual Markdown-copy fallback.
await page.evaluate(()=>{
  window.savedExecCommand=document.execCommand;
  document.execCommand=()=>false;
  Object.defineProperty(navigator,'clipboard',{configurable:true,value:{writeText:async()=>{throw Error('denied');}}});
});
await markdownButton.click();
assert.equal(await page.locator('#copy-value').inputValue(),copiedMarkdown);
await page.locator('#copy-dialog button').click();
await page.evaluate(()=>{document.execCommand=window.savedExecCommand;delete navigator.clipboard;});
const copyBounds=await markdownButton.boundingBox();
const rowBounds=await page.locator('#p-2 > details > summary').boundingBox();
assert(copyBounds.x>rowBounds.x+rowBounds.width-80,'Markdown copy control should sit at the right');

await page.locator('#p-3 > details > summary').click({position:{x:180,y:8}});
const image=page.locator('#p-3 img');
await image.evaluate(node=>node.decode());
assert((await image.boundingBox()).height<=200);
await page.locator('#p-3 .image-size').click();
assert((await image.boundingBox()).height===640);
assert.equal(await page.locator('#p-3 .image-size').getAttribute('aria-pressed'),'true');
await page.locator('#p-3 .image-size').click();
assert((await image.boundingBox()).height<=200);

await page.locator('#p-4 > details > summary').click({position:{x:180,y:8}});
await page.locator('#p-4 .diagram svg').waitFor({timeout:25000});
assert.equal(await page.locator('#p-4 .diagram').getAttribute('data-rendered'),'true');
assert.deepEqual(remoteRequests,[peleURL]);
assert.equal(await page.locator('#p-5 .diagram svg').count(),0);
await page.locator('#p-5 > details > summary').click({position:{x:180,y:8}});
await page.locator('#p-5 .diagram svg').waitFor();
assert.equal(await page.locator('#p-5 .diagram').getAttribute('data-rendered'),'true');
await page.locator('#p-5 > details > summary > .copy-markdown').click();
const diagramMarkdown=await page.evaluate(()=>navigator.clipboard.readText());
assert(diagramMarkdown.includes('```mermaid\nsequenceDiagram'));
assert(!diagramMarkdown.includes('This diagram shows one path'));
await page.screenshot({path:screenshots+'/preview-visuals.png',fullPage:true});
await page.locator('#collapse-all').click();
assert.equal(await page.locator('.point-content:visible').count(),0);
await page.locator('#expand-all').click();
await page.setViewportSize({width:390,height:844});
assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
await page.locator('#p-3 .image-size').click();
assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
assert(await page.locator('#p-3 .image-frame').evaluate(node=>node.scrollWidth>node.clientWidth));
await page.locator('#p-3 .image-size').click();
await page.screenshot({path:screenshots+'/preview-visuals-mobile.png',fullPage:true});
assert.deepEqual(errors,[]);
assert.deepEqual(remoteRequests,[peleURL]);

// Standard Mermaid fences use the same renderer for every diagram type.
remoteRequests.length=0;
await page.goto(pathToFileURL(root+'/examples/diagrams.html').href);
assert.equal(await page.locator('details[open]').count(),0);
assert.equal(await page.locator('.diagram svg').count(),0);
assert.equal(await page.evaluate(()=>typeof window.pele),'undefined');
assert.deepEqual(remoteRequests,[]);
await page.locator('#p-1 > details > summary').click({position:{x:180,y:8}});
await page.locator('#p-1 .diagram svg').waitFor({timeout:25000});
assert.equal(await page.locator('#p-1 .diagram').getAttribute('data-rendered'),'true');
assert.deepEqual(remoteRequests,[peleURL]);
assert.equal(await page.locator('#p-2 .diagram svg').count(),0);
await page.locator('#p-2 > details > summary').click({position:{x:180,y:8}});
await page.locator('#p-2 .diagram svg').waitFor();
assert.equal(await page.locator('#p-2 .diagram').getAttribute('data-rendered'),'true');
assert.deepEqual(remoteRequests,[peleURL]);
await page.locator('#p-1 > details > summary > .copy-markdown').click();
assert((await page.evaluate(()=>navigator.clipboard.readText())).includes('```mermaid\nflowchart LR'));
for(const number of [3,4]) {
  await page.locator(`#p-${number} > details > summary`).click({position:{x:180,y:8}});
  await page.locator(`#p-${number} .diagram svg`).waitFor();
  assert.equal(await page.locator(`#p-${number} .diagram`).getAttribute('data-rendered'),'true');
}
assert.deepEqual(remoteRequests,[peleURL]);
assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
await page.screenshot({path:screenshots+'/preview-pele.png',fullPage:true});
assert.deepEqual(errors,[]);

// If the CDN is unavailable, keep the source and show a readable error.
await page.route(peleURL,route=>route.abort());
await page.reload();
await page.locator('#p-1 > details > summary').click({position:{x:180,y:8}});
await page.waitForFunction(()=>document.querySelector('#p-1 .diagram').dataset.rendered==='error');
assert((await page.locator('#p-1 .diagram').innerText()).includes('internet connection is required'));
assert((await page.locator('#p-1 .diagram pre').innerText()).includes('flowchart LR'));
await context.close();
const noJS=await browser.newContext({javaScriptEnabled:false});
const plain=await noJS.newPage();
await plain.goto(pathToFileURL(root+'/examples/math-proof.html').href);
await plain.locator('#p-2 > details > summary').click();
assert.equal(await plain.locator('#p-2 > details').getAttribute('open'),'');
console.log('PASS: nine examples; compact layout; Markdown and ID copying; highlighting; Mermaid flowchart, sequence, class, and mindmap via Pele; lazy CDN loading and failure fallback; 200px images; references; keyboard; mobile.');
} finally {
  await browser.close();
}
