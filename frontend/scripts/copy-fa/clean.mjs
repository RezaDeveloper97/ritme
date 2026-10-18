#!/usr/bin/env node
// Value-only Persian cleanup for messages/fa/*.json (never runs the cleaner on raw JSON).
//   clean.mjs <namespace…> | --all [--cleanup <persian_cleanup.py>] [--dir <messagesDir>]
//             [--ezafe] [--dry-run] [--diff[=N]]
// The cleaner path comes from --cleanup or env PERSIAN_CLEANUP. The skill is not copied into the repo.
import fs from 'node:fs';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { FA_DIR, listNamespaces, icuSignature, assertParses } from './lib.mjs';
import { protect, restore, SEPARATOR } from './protect.mjs';

// persian_cleanup.py EDITOR_STEPS minus fix_ezafe (unless --ezafe).
const STEPS = [
  'fix_arabic_chars', 'convert_digits', 'fix_persian_punctuation', 'fix_zwnj_verbs', 'fix_zwnj_suffixes',
  'fix_zwnj_possessives', 'fix_punctuation_spacing', 'fix_ellipsis',
  'fix_repeated_punctuation', 'remove_extra_spaces',
];
// Opt-in only (--compound): it glues «باز کردن» -> «بازکردن», which is the opposite of STYLE.md.
const COMPOUND = 'fix_zwnj_compound_verbs';

function runCleaner(cleanupPath, input, opts) {
  const steps = [...STEPS];
  if (opts.compound) steps.splice(5, 0, COMPOUND);
  if (opts.ezafe) steps.splice(steps.indexOf('fix_punctuation_spacing') + 1, 0, 'fix_ezafe');
  const r = spawnSync('python3', ['-I', cleanupPath, '--steps', steps.join(',')], {
    input, encoding: 'utf8', maxBuffer: 256 * 1024 * 1024,
  });
  if (r.status !== 0) throw new Error(`persian_cleanup failed: ${r.stderr || r.error}`);
  return r.stdout.endsWith('\n') ? r.stdout.slice(0, -1) : r.stdout;
}

/** Cleans a list of masked strings in one python call (falls back to one call each on a count mismatch). */
function cleanMany(cleanupPath, masked, opts) {
  if (!masked.length) return [];
  // leading/trailing whitespace is significant in fragments (« و »): keep it out of the cleaner
  const edges = masked.map((m) => [m.match(/^\s*/)[0], m.match(/\s*$/)[0]]);
  const cores = masked.map((m) => m.trim());
  const joined = runCleaner(cleanupPath, cores.join(`\n${SEPARATOR}\n`), opts);
  let parts = joined.split(`\n${SEPARATOR}\n`);
  if (parts.length !== cores.length) parts = cores.map((m) => runCleaner(cleanupPath, m, opts));
  return parts.map((p, i) => (cores[i] ? edges[i][0] + p + edges[i][1] : masked[i]));
}

function sameSig(a, b) {
  const x = icuSignature(a);
  const y = icuSignature(b);
  return x.length === y.length && x.every((v, i) => v === y[i]);
}

/** Core: returns {obj, changes:[{path, before, after}], skipped:[{path, reason}]} */
export function cleanObject(obj, cleanupPath, opts = {}) {
  const items = []; // {holder, key, path, value, p}
  const walk = (node, prefix) => {
    for (const [k, v] of Object.entries(node)) {
      const p = prefix ? `${prefix}.${k}` : k;
      if (typeof v === 'string') {
        const pr = protect(v);
        if (pr) items.push({ holder: node, key: k, path: p, value: v, pr });
      } else if (v && typeof v === 'object') walk(v, p);
    }
  };
  walk(obj, '');
  const cleaned = cleanMany(cleanupPath, items.map((i) => i.pr.masked), opts);
  const changes = [];
  const skipped = [];
  items.forEach((it, idx) => {
    const restored = restore(cleaned[idx], it.pr.tokens);
    if (restored === null) { skipped.push({ path: it.path, reason: 'protected token lost or reordered' }); return; }
    if (restored === it.value) return;
    try {
      assertParses(restored);
      if (!sameSig(restored, it.value)) throw new Error('ICU signature changed');
    } catch (e) {
      skipped.push({ path: it.path, reason: `ICU check failed: ${e.message}` });
      return;
    }
    it.holder[it.key] = restored;
    changes.push({ path: it.path, before: it.value, after: restored });
  });
  return { obj, changes, skipped };
}

function main() {
  const argv = process.argv.slice(2);
  const o = { ns: [], all: false, cleanup: process.env.PERSIAN_CLEANUP, dir: FA_DIR, ezafe: false, compound: false, dry: false, diff: 0 };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--all') o.all = true;
    else if (a === '--cleanup') o.cleanup = argv[++i];
    else if (a === '--dir') o.dir = path.resolve(argv[++i]);
    else if (a === '--ezafe') o.ezafe = true;
    else if (a === '--compound') o.compound = true;
    else if (a === '--dry-run') o.dry = true;
    else if (a === '--diff') o.diff = 20;
    else if (a.startsWith('--diff=')) o.diff = Number(a.slice(7));
    else if (a.startsWith('--')) { console.error(`unknown option ${a}`); process.exit(2); }
    else o.ns.push(a.replace(/\.json$/, ''));
  }
  if (!o.cleanup || !fs.existsSync(o.cleanup)) {
    console.error('persian_cleanup.py not found: pass --cleanup <path> or set PERSIAN_CLEANUP');
    process.exit(2);
  }
  const names = o.all ? listNamespaces(o.dir) : o.ns;
  if (!names.length) { console.error('usage: clean.mjs <namespace…> | --all [--cleanup <path>] [--dir <dir>] [--ezafe] [--dry-run] [--diff[=N]]'); process.exit(2); }
  for (const ns of names) {
    const file = path.join(o.dir, `${ns}.json`);
    const text = fs.readFileSync(file, 'utf8');
    const obj = JSON.parse(text);
    const { changes, skipped } = cleanObject(obj, o.cleanup, { ezafe: o.ezafe, compound: o.compound });
    if (!o.dry && changes.length) {
      // Rewrite the raw text literal-by-literal so formatting and key order stay byte-identical
      // (JSON.stringify would reorder integer-like keys and expand inline objects).
      let out = text;
      for (const c of changes) {
        const lit = JSON.stringify(c.before).replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        out = out.replace(new RegExp(`${lit}(?!\\s*:)`, 'g'), () => JSON.stringify(c.after));
      }
      if (JSON.stringify(JSON.parse(out)) !== JSON.stringify(obj)) {
        console.error(`${ns}: text rewrite diverged from the cleaned tree; file left untouched`);
        process.exitCode = 1;
        continue;
      }
      fs.writeFileSync(file, out);
    }
    console.log(`${ns}: ${changes.length} value(s) ${o.dry ? 'would change' : 'changed'}, ${skipped.length} skipped`);
    for (const s of skipped) console.log(`  skipped ${s.path}: ${s.reason}`);
    for (const c of changes.slice(0, o.diff)) console.log(`  ${c.path}\n    - ${c.before}\n    + ${c.after}`);
  }
}

import { fileURLToPath } from 'node:url';
if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main();
