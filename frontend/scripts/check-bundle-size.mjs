#!/usr/bin/env node
/**
 * Bundle-size gate (T-M12-27, finding P-01) — run after `npm run build`.
 *
 *   node scripts/check-bundle-size.mjs            check against the budgets below
 *   node scripts/check-bundle-size.mjs --report   print the numbers only (never fails)
 *
 * It reads the prerendered `/fa/*` pages in `.next/server/app/fa` (the `<link rel=stylesheet>` and `<script src>`
 * tags a first visit downloads) and checks four things:
 *
 *   1. global CSS   — the stylesheets EVERY route downloads (intersection over all prerendered pages), minified bytes.
 *                     This is globals.css + whatever the root layout imports. Screen CSS lives next to its screen
 *                     (`<slice>/ui/<prefix>.css`, shared ones in `shared/ui/styles/`) and must NOT creep back in here.
 *   2. heaviest route CSS — global + that route's own CSS, the worst page, minified bytes.
 *   3. first-load JS of /fa/home and /fa/log — gzip bytes of the scripts in the page HTML (what Next prints as "First Load JS").
 *
 * Budgets = the measured value at T-M12-27 × 1.1. Raise one only with a reason in docs/quality-2026-11/perf.md.
 */
import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { gzipSync } from 'node:zlib';

const ROOT = resolve(new URL('..', import.meta.url).pathname);
const NEXT = join(ROOT, '.next');
const PAGES = join(NEXT, 'server', 'app', 'fa');

/** Bytes. CSS measured at T-M12-27, JS at T-M12-28 (see docs/quality-2026-11/perf.md) × 1.1, rounded up to 1 KB. */
const BUDGET = {
  globalCss: 243_712, //          measured 221_148
  heaviestRouteCss: 411_648, //   measured 373_777 (/fa/home)
  homeFirstLoadJsGzip: 423_936, // measured 385_062 at T-M12-28 (was 427_339 / budget 457_728)
  logFirstLoadJsGzip: 410_624, //  measured 372_902 at T-M12-28 (was 407_211)
};

const REPORT = process.argv.includes('--report');

function htmlFiles(dir) {
  const out = [];
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) out.push(...htmlFiles(p));
    else if (name.endsWith('.html')) out.push(p);
  }
  return out;
}

const size = (asset) => statSync(join(NEXT, decodeURIComponent(asset.replace(/^\/_next\//, '')))).size;
const gz = (asset) => gzipSync(readFileSync(join(NEXT, decodeURIComponent(asset.replace(/^\/_next\//, ''))))).length;

function assetsOf(html) {
  const css = [...html.matchAll(/href="(\/_next\/static\/[^"]+\.css)"/g)].map((m) => m[1]);
  const js = [...html.matchAll(/src="(\/_next\/static\/[^"]+\.js)"/g)].map((m) => m[1]);
  return { css: [...new Set(css)], js: [...new Set(js)] };
}

if (!existsSync(PAGES)) {
  console.error('✖ bundle gate: .next/server/app/fa not found — run `npm run build` first.');
  process.exit(2);
}

const pages = htmlFiles(PAGES).map((file) => ({ file, ...assetsOf(readFileSync(file, 'utf8')) }));
if (!pages.length) {
  console.error('✖ bundle gate: no prerendered /fa pages found.');
  process.exit(2);
}

const common = pages.map((p) => new Set(p.css)).reduce((a, b) => new Set([...a].filter((x) => b.has(x))));
const globalCss = [...common].reduce((n, a) => n + size(a), 0);

let heaviest = { file: '', bytes: 0, n: 0 };
for (const p of pages) {
  const bytes = p.css.reduce((n, a) => n + size(a), 0);
  if (bytes > heaviest.bytes) heaviest = { file: p.file.slice(PAGES.length), bytes, n: p.css.length };
}

const home = pages.find((p) => p.file.endsWith(join('fa', 'home.html')));
if (!home) {
  console.error('✖ bundle gate: /fa/home was not prerendered.');
  process.exit(2);
}
const homeJs = home.js.reduce((n, a) => n + gz(a), 0);
const log = pages.find((p) => p.file.endsWith(join('fa', 'log.html')));
if (!log) {
  console.error('✖ bundle gate: /fa/log was not prerendered.');
  process.exit(2);
}
const logJs = log.js.reduce((n, a) => n + gz(a), 0);

const kb = (n) => `${(n / 1024).toFixed(1)} KB`;
const rows = [
  ['global CSS (every route)', globalCss, BUDGET.globalCss],
  [`heaviest route CSS (${heaviest.file}, ${heaviest.n} files)`, heaviest.bytes, BUDGET.heaviestRouteCss],
  ['/fa/home first-load JS (gzip)', homeJs, BUDGET.homeFirstLoadJsGzip],
  ['/fa/log first-load JS (gzip)', logJs, BUDGET.logFirstLoadJsGzip],
];
for (const [name, value, budget] of rows) console.log(`${name.padEnd(58)} ${String(value).padStart(8)} B ${kb(value).padStart(10)}   budget ${kb(budget)}`);

if (REPORT) process.exit(0);
const over = rows.filter(([, value, budget]) => value > budget);
if (over.length) {
  console.error(`\n✖ bundle gate: ${over.length} over budget — ${over.map((r) => r[0]).join('; ')}.`);
  console.error('  A screen stylesheet belongs next to its screen (imported there), not in globals.css; see docs/quality-2026-11/perf.md.');
  process.exit(1);
}
console.log('\n✔ bundle gate passed');
