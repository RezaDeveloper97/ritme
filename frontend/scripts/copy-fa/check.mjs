#!/usr/bin/env node
// copy-fa checker: structure gate, sync gate, lint. See tasks/M8/T-M8-02.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { FA_DIR, EN_DIR, BACKEND_FA_DIR, listNamespaces, readJson, flatten, gitShow, REPO_ROOT } from './lib.mjs';
import { compareStructure, checkIcu } from './structure.mjs';
import { lintValue } from './lint.mjs';
import { parseGlossary, compileGlossary } from './glossary.mjs';

export function checkNamespace(ns, { base = 'HEAD', glossary, faDir = FA_DIR, enDir = EN_DIR, backendDir = BACKEND_FA_DIR, syncGate = true } = {}) {
  const issues = []; // {level, kind, key, rule, value}
  const push = (arr, level, kind) => arr.forEach((e) => issues.push({ level, kind, ...e }));
  const faFile = path.join(faDir, `${ns}.json`);
  const cur = readJson(faFile);

  const enFile = path.join(enDir, `${ns}.json`);
  const enObj = fs.existsSync(enFile) ? readJson(enFile) : null;
  const enFlat = enObj ? flatten(enObj) : {};
  const baseText = gitShow(base, `frontend/messages/fa/${ns}.json`);
  let baseObj = null;
  if (baseText !== null) {
    baseObj = JSON.parse(baseText);
    // A key added since <base> is legitimate when en carries it too (uncommitted feature work in the
    // tree); a rewrite never adds keys, so only keys unknown to en are flagged.
    const vsBase = compareStructure(cur, baseObj, base).filter(
      (e) => !(e.rule.startsWith('structure: extra key') && e.key in enFlat),
    );
    push(vsBase, 'error', 'structure');
  }
  if (enObj) push(compareStructure(cur, enObj, 'en'), 'error', 'structure');
  push(checkIcu(cur, baseObj), 'error', 'structure');

  if (syncGate) {
    const be = path.join(backendDir, `${ns}.json`);
    if (!fs.existsSync(be)) issues.push({ level: 'error', kind: 'sync', key: '*', rule: 'sync: missing in backend-go/resources/translations/fa', value: '' });
    else if (!fs.readFileSync(be).equals(fs.readFileSync(faFile))) {
      issues.push({ level: 'error', kind: 'sync', key: '*', rule: 'sync: differs from backend-go/resources/translations/fa', value: '' });
    }
  }

  const baseFlat = baseObj ? flatten(baseObj) : {};
  for (const [key, value] of Object.entries(flatten(cur))) {
    for (const l of lintValue(key, value, { base: baseFlat[key], glossary })) {
      issues.push({ level: l.level, kind: 'lint', key, rule: l.rule, value: String(value) });
    }
  }
  return issues;
}

function parseArgs(argv) {
  const o = { ns: [], all: false, baseline: false, base: 'HEAD', verbose: false };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--all') o.all = true;
    else if (a === '--baseline') o.baseline = true;
    else if (a === '--verbose') o.verbose = true;
    else if (a === '--base') o.base = argv[++i];
    else if (a.startsWith('--')) { console.error(`unknown option ${a}`); process.exit(2); }
    else o.ns.push(a.replace(/\.json$/, ''));
  }
  return o;
}

function main() {
  const o = parseArgs(process.argv.slice(2));
  const names = o.all ? listNamespaces() : o.ns;
  if (!names.length) {
    console.error('usage: check.mjs <namespace…> | --all [--baseline] [--base <git-ref>] [--verbose]');
    process.exit(2);
  }
  const glossary = compileGlossary(parseGlossary());
  const rows = [];
  let hardErrors = 0;
  let lintErrors = 0;
  for (const ns of names) {
    const issues = checkNamespace(ns, { base: o.base, glossary });
    const hard = issues.filter((i) => i.level === 'error' && i.kind !== 'lint').length;
    const lintE = issues.filter((i) => i.level === 'error' && i.kind === 'lint').length;
    const warn = issues.filter((i) => i.level === 'warn').length;
    rows.push({ ns, hard, lintE, warn });
    hardErrors += hard;
    lintErrors += lintE;
    if (!o.baseline || o.verbose) {
      for (const i of issues) console.log(`${ns} ${i.key}: ${i.level === 'warn' ? 'warn ' : ''}${i.rule}${i.value ? ` — ${i.value}` : ''}`);
    }
  }
  if (o.baseline) {
    const w = Math.max(...rows.map((r) => r.ns.length), 9);
    console.log(`${'namespace'.padEnd(w)}  struct/sync-err  lint-err  warn`);
    for (const r of rows) console.log(`${r.ns.padEnd(w)}  ${String(r.hard).padStart(15)}  ${String(r.lintE).padStart(8)}  ${String(r.warn).padStart(4)}`);
    const t = rows.reduce((a, r) => ({ hard: a.hard + r.hard, lintE: a.lintE + r.lintE, warn: a.warn + r.warn }), { hard: 0, lintE: 0, warn: 0 });
    console.log(`${'TOTAL'.padEnd(w)}  ${String(t.hard).padStart(15)}  ${String(t.lintE).padStart(8)}  ${String(t.warn).padStart(4)}   (${rows.length} namespaces)`);
    process.exit(hardErrors ? 1 : 0);
  }
  console.error(`${names.length} namespace(s): ${hardErrors} structure/sync error(s), ${lintErrors} lint error(s)`);
  process.exit(hardErrors + lintErrors ? 1 : 0);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main();
void REPO_ROOT;
