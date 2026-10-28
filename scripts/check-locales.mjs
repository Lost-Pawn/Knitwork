import ts from '../frontend/node_modules/typescript/lib/typescript.js';
import fs from 'node:fs';
import assert from 'node:assert/strict';
const source=fs.readFileSync(new URL('../frontend/lib/i18n.ts',import.meta.url),'utf8');
const js=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.ES2022,target:ts.ScriptTarget.ES2022}}).outputText;
const {dictionaries,languages}=await import('data:text/javascript;base64,'+Buffer.from(js).toString('base64'));
const keys=Object.keys(dictionaries.en);
for(const locale of Object.keys(languages)){
 assert.deepEqual(Object.keys(dictionaries[locale]),keys,`Missing keys in ${locale}`);
 for(const key of keys)assert.ok(typeof dictionaries[locale][key]==='string'&&dictionaries[locale][key].trim(),`${locale}.${key} is empty`);
 console.log(`${locale}: ${keys.length} translated interface messages`);
}
