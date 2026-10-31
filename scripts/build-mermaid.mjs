import {build} from 'esbuild';
import {readFile, writeFile, readdir} from 'node:fs/promises';
import {dirname, resolve, join} from 'node:path';
import {fileURLToPath} from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const result = await build({
  stdin:{contents:"import mermaid from 'mermaid'; window.mermaid = mermaid;",resolveDir:root},
  bundle:true,
  format:'iife',
  platform:'browser',
  target:'es2022',
  minify:true,
  legalComments:'eof',
  metafile:true,
  outfile:join(root,'web/mermaid.min.js')
});

// Keep the licenses for packages whose code is included in the checked-in bundle.
const packages = new Map();
for (const input of Object.keys(result.metafile.inputs)) {
  if (!input.includes('node_modules/')) continue;
  let folder = dirname(resolve(input));
  while (folder !== dirname(folder)) {
    try {
      const metadata = JSON.parse(await readFile(join(folder,'package.json'),'utf8'));
      if (metadata.name) { packages.set(folder,metadata); break; }
    } catch {}
    folder = dirname(folder);
  }
}
const notices = [];
for (const [folder,metadata] of [...packages].sort((a,b)=>a[1].name.localeCompare(b[1].name))) {
  const files = (await readdir(folder)).filter(name=>/^(licen[sc]e|copying)([.-]|$)/i.test(name));
  const texts = [];
  for (const name of files) {
    try { texts.push(await readFile(join(folder,name),'utf8')); } catch {}
  }
  notices.push(`${metadata.name} ${metadata.version}\nLicense: ${typeof metadata.license==='string'?metadata.license:JSON.stringify(metadata.license)}\n\n${texts.join('\n\n')}`);
}
await writeFile(join(root,'web/mermaid.LICENSE.txt'),notices.join('\n\n'+'='.repeat(72)+'\n\n'));
console.log(`Built Mermaid bundle and ${packages.size} dependency license notices.`);
