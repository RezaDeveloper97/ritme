#!/usr/bin/env node
// Copies frontend/messages/fa/<ns>.json to backend-go/resources/translations/fa/.
import fs from 'node:fs';
import path from 'node:path';
import { FA_DIR, BACKEND_FA_DIR, listNamespaces } from './lib.mjs';

const args = process.argv.slice(2);
const names = args.includes('--all') ? listNamespaces() : args.filter((a) => !a.startsWith('--')).map((a) => a.replace(/\.json$/, ''));
if (!names.length) {
  console.error('usage: sync.mjs <namespace…> | --all');
  process.exit(2);
}
let changed = 0;
for (const ns of names) {
  const src = path.join(FA_DIR, `${ns}.json`);
  const dst = path.join(BACKEND_FA_DIR, `${ns}.json`);
  const buf = fs.readFileSync(src);
  if (!fs.existsSync(dst) || !fs.readFileSync(dst).equals(buf)) {
    fs.writeFileSync(dst, buf);
    changed++;
    console.log(`synced ${ns}`);
  }
}
console.log(`${names.length} namespace(s), ${changed} updated`);
