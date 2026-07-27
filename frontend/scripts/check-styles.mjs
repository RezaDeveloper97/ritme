#!/usr/bin/env node
/**
 * Style gate — enforces CLAUDE.md §10.1 and §12.
 *
 * Three checks:
 *   1. static `style={{ … }}` props   → belongs in a class in globals.css
 *   2. hex colour literals            → does not flip in dark mode
 *   3. `var(--x)` for an undeclared x → silently resolves to nothing
 *
 * It is a RATCHET, not a wall. The repo still carries a lot of pre-existing
 * inline styling, so failing on every one of them would just be noise. Instead
 * each file has a baseline count; the gate fails when a file goes ABOVE its
 * baseline, and tells you to lower the baseline when you go below it.
 *
 *   node scripts/check-styles.mjs [files…]     check (default: all of src/)
 *   node scripts/check-styles.mjs --update-baseline
 */
import { readFileSync, writeFileSync, readdirSync, statSync, existsSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';

const ROOT = resolve(new URL('..', import.meta.url).pathname);
const SRC = join(ROOT, 'src');
const GLOBALS = join(SRC, 'app', 'globals.css');
const BASELINE = join(ROOT, 'scripts', 'styles-baseline.json');

// ── helpers ──────────────────────────────────────────────────────

function walk(dir) {
  const out = [];
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) out.push(...walk(p));
    else if (/\.tsx?$/.test(name) && !/\.(test|spec)\.tsx?$/.test(name)) out.push(p);
  }
  return out;
}

/** Blank out comments and string bodies so scans don't trip on prose. */
function maskComments(src) {
  return src
    .replace(/\/\*[\s\S]*?\*\//g, m => ' '.repeat(m.length))
    .replace(/(^|[^:])\/\/[^\n]*/g, (m, p) => p + ' '.repeat(m.length - p.length));
}

/** Extract the balanced `{ … }` object body of each `style=` prop. */
function styleObjects(src) {
  const out = [];
  const re = /style=\{/g;
  let m;
  while ((m = re.exec(src))) {
    let depth = 0, i = m.index + 'style='.length, start = i;
    for (; i < src.length; i += 1) {
      if (src[i] === '{') depth += 1;
      else if (src[i] === '}') {
        depth -= 1;
        if (depth === 0) break;
      }
    }
    const body = src.slice(start, i + 1);
    out.push({ body, line: src.slice(0, m.index).split('\n').length });
  }
  return out;
}

/**
 * A style object is allowed only when at least one value comes from data.
 * Strip the quoted literals and the property names; if nothing alphabetic is
 * left, every value was a literal and the whole thing belongs in a class.
 */
function isStaticStyle(body) {
  const stripped = body
    .replace(/`(?:[^`\\]|\\.)*`/g, ' ')     // template literals → data
    .replace(/'(?:[^'\\]|\\.)*'/g, ' ')
    .replace(/"(?:[^"\\]|\\.)*"/g, ' ')
    .replace(/\b[A-Za-z_$][\w$]*\s*:/g, ' ') // property names
    .replace(/\bvar\b|\bundefined\b/g, ' '); // var(--x) / undefined are literals
  // A backtick or `?` means the value was computed even if the identifiers
  // were consumed above.
  if (/[`?]/.test(body.replace(/'(?:[^'\\]|\\.)*'/g, ' ').replace(/"(?:[^"\\]|\\.)*"/g, ' '))) {
    return false;
  }
  return !/[A-Za-z_$]/.test(stripped);
}

function declaredVars() {
  const css = readFileSync(GLOBALS, 'utf8');
  const set = new Set();
  for (const m of css.matchAll(/(--[\w-]+)\s*:/g)) set.add(m[1]);
  return set;
}

// ── the checks ───────────────────────────────────────────────────

function scan(file, vars) {
  const raw = readFileSync(file, 'utf8');
  const src = maskComments(raw);
  const rel = relative(ROOT, file);
  const findings = [];

  for (const { body, line } of styleObjects(src)) {
    if (isStaticStyle(body)) {
      const preview = body.replace(/\s+/g, ' ').slice(0, 68);
      findings.push({ rel, line, kind: 'static-style', msg: `static style prop → move to a class: ${preview}` });
    }
  }

  for (const m of src.matchAll(/#(?:[0-9A-Fa-f]{8}|[0-9A-Fa-f]{6}|[0-9A-Fa-f]{3})\b/g)) {
    const line = src.slice(0, m.index).split('\n').length;
    findings.push({ rel, line, kind: 'hex', msg: `hex literal ${m[0]} → use a token from globals.css` });
  }

  for (const m of src.matchAll(/var\(\s*(--[\w-]+)/g)) {
    if (!vars.has(m[1])) {
      const line = src.slice(0, m.index).split('\n').length;
      findings.push({ rel, line, kind: 'unknown-var', msg: `var(${m[1]}) is not declared in globals.css` });
    }
  }

  return findings;
}

// ── run ──────────────────────────────────────────────────────────

const args = process.argv.slice(2);
const update = args.includes('--update-baseline');
const explicit = args.filter(a => !a.startsWith('--'));

const vars = declaredVars();
const files = explicit.length
  ? explicit.map(f => resolve(f)).filter(f => existsSync(f) && /\.tsx?$/.test(f) && f.startsWith(SRC))
  : walk(SRC);

const counts = {};
const all = [];
for (const f of files) {
  const found = scan(f, vars);
  if (found.length) {
    counts[relative(ROOT, f)] = found.length;
    all.push(...found);
  }
}

if (update) {
  writeFileSync(BASELINE, JSON.stringify(counts, null, 2) + '\n');
  console.log(`baseline written: ${Object.keys(counts).length} files, ${all.length} findings`);
  process.exit(0);
}

// `--list` ignores the baseline and prints everything still outstanding — the
// working view while cleaning a file up.
if (args.includes('--list')) {
  for (const f of all) console.log(`${f.rel}:${f.line}  ${f.msg}`);
  console.log(`\n${all.length} findings in ${Object.keys(counts).length} files`);
  process.exit(0);
}

const baseline = existsSync(BASELINE) ? JSON.parse(readFileSync(BASELINE, 'utf8')) : {};
let failed = false;
const improved = [];

for (const [rel, n] of Object.entries(counts)) {
  const allowed = baseline[rel] ?? 0;
  if (n > allowed) {
    failed = true;
    console.error(`\n✖ ${rel} — ${n} style findings, baseline allows ${allowed}`);
    for (const f of all.filter(x => x.rel === rel)) {
      console.error(`    ${rel}:${f.line}  ${f.msg}`);
    }
  }
}
// Only nag about lowering the baseline for files we actually looked at.
for (const rel of files.map(f => relative(ROOT, f))) {
  const allowed = baseline[rel] ?? 0;
  const n = counts[rel] ?? 0;
  if (allowed > 0 && n < allowed) improved.push(`${rel}: ${allowed} → ${n}`);
}

if (failed) {
  console.error('\nSee CLAUDE.md §10.1. Static styles belong in src/app/globals.css;');
  console.error('only values that come from data may stay inline.');
  console.error('If a finding is a deliberate exception, run: npm run lint:styles:accept\n');
  process.exit(1);
}

if (improved.length) {
  console.log('✔ style gate passed — and these files improved:');
  for (const i of improved) console.log(`    ${i}`);
  console.log('  lock it in with: npm run lint:styles:accept');
} else {
  console.log(`✔ style gate passed (${files.length} files)`);
}
