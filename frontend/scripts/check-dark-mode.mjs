#!/usr/bin/env node
/**
 * Dark-mode gate — the companion to `check-styles.mjs`.
 *
 * `check-styles` proves colours go through a token; this proves the tokens
 * actually *flip*, and that what they flip to is still readable. The failure it
 * exists to catch is the one CLAUDE.md §10.2 calls the most common visual bug
 * in this codebase: a value that only ever had a light answer, so dark mode
 * silently renders dark-on-dark.
 *
 * Checks:
 *   1. plumbing   — <html data-theme> is written before first paint, the inline
 *                   bootstrap and the store agree on the storage key, and both
 *                   themes declare `color-scheme`.
 *   2. parity     — every colour token in :root either flips in
 *                   [data-theme="dark"], is derived from tokens that do, or is
 *                   listed here as deliberately theme-stable *with a reason*.
 *   3. contrast   — declared foreground/background token pairs are resolved in
 *                   BOTH themes and scored against WCAG; dark may never be
 *                   materially worse than light for the same pair.
 *   4. literals   — colour literals in rule bodies (not the token blocks) that
 *                   would not flip, outside the on-gradient allowlist.
 *   5. components — colour literals in .ts/.tsx.
 *
 *   node scripts/check-dark-mode.mjs            check
 *   node scripts/check-dark-mode.mjs --verbose  also print every passing pair
 */
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';

const ROOT = resolve(new URL('..', import.meta.url).pathname);
const SRC = join(ROOT, 'src');
const GLOBALS = join(SRC, 'app', 'globals.css');
const THEME_DIR = join(SRC, 'shared', 'theme');

const VERBOSE = process.argv.includes('--verbose');

const problems = [];
const warnings = [];
const notes = [];
const fail = (where, msg) => problems.push(`${where} — ${msg}`);
const warn = (where, msg) => warnings.push(`${where} — ${msg}`);

// ── colour maths ─────────────────────────────────────────────────

/** sRGB → relative luminance (WCAG 2.1). */
function luminance([r, g, b]) {
  const f = (c) => {
    const s = c / 255;
    return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
  };
  return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b);
}

function contrast(a, b) {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

/** Composite a possibly-translucent colour over an opaque backdrop. */
function over(fg, bg) {
  const a = fg[3] ?? 1;
  if (a >= 1) return fg.slice(0, 3);
  return [0, 1, 2].map((i) => Math.round(fg[i] * a + bg[i] * (1 - a)));
}

function parseHex(hex) {
  let h = hex.slice(1);
  if (h.length === 3) h = [...h].map((c) => c + c).join('');
  if (h.length === 8) {
    return [0, 2, 4].map((i) => parseInt(h.slice(i, i + 2), 16)).concat(parseInt(h.slice(6, 8), 16) / 255);
  }
  if (h.length !== 6) return null;
  return [0, 2, 4].map((i) => parseInt(h.slice(i, i + 2), 16));
}

/** Split `a, b, c` at top level — commas inside nested parens stay put. */
function splitTop(text) {
  const out = [];
  let depth = 0;
  let cur = '';
  for (const ch of text) {
    if (ch === '(') depth += 1;
    if (ch === ')') depth -= 1;
    if (ch === ',' && depth === 0) {
      out.push(cur.trim());
      cur = '';
    } else cur += ch;
  }
  if (cur.trim()) out.push(cur.trim());
  return out;
}

// ── the stylesheet ───────────────────────────────────────────────

const css = readFileSync(GLOBALS, 'utf8');

/** Every `--token: value;` inside the first block matching `selector`. */
function tokenBlock(selector) {
  const re = new RegExp(`(^|\\n)${selector.replace(/[[\]"]/g, '\\$&')}\\s*\\{([\\s\\S]*?)\\n\\}`, 'm');
  const m = css.match(re);
  if (!m) return null;
  const map = new Map();
  for (const d of m[2].matchAll(/^\s*(--[\w-]+)\s*:\s*([^;]+);/gm)) {
    map.set(d[1], d[2].trim());
  }
  return { body: m[2], map };
}

const light = tokenBlock(':root');
const dark = tokenBlock('[data-theme="dark"]');
if (!light) fail('globals.css', 'no :root token block found');
if (!dark) fail('globals.css', 'no [data-theme="dark"] token block found');

/**
 * Resolve a token to an rgb(a) tuple *in a given theme*. Follows `var()`,
 * evaluates `color-mix(in srgb, A p%, B)`, and refuses gradients (they have no
 * single colour). Dark falls back to the light value, exactly like the cascade.
 */
function resolveColor(value, theme, seen = new Set()) {
  const v = String(value).trim();
  if (!v) return null;
  if (v === 'transparent') return [0, 0, 0, 0];
  if (/gradient\(/.test(v)) return 'gradient';

  if (v.startsWith('#')) return parseHex(v);

  let m = v.match(/^rgba?\(([^)]+)\)$/);
  if (m) {
    const parts = splitTop(m[1].replace(/\//g, ' ')).flatMap((p) => p.split(/\s+/)).filter(Boolean);
    const n = parts.slice(0, 3).map(Number);
    const a = parts.length > 3 ? Number(parts[3]) : 1;
    return n.some(Number.isNaN) ? null : [...n, a];
  }

  m = v.match(/^var\(\s*(--[\w-]+)\s*(?:,([\s\S]+))?\)$/);
  if (m) {
    const name = m[1];
    if (seen.has(name + theme)) return null; // a cycle: report as unresolved
    const next = (theme === 'dark' ? dark.map.get(name) : undefined) ?? light.map.get(name);
    if (next === undefined) return m[2] ? resolveColor(m[2], theme, seen) : null;
    return resolveColor(next, theme, new Set(seen).add(name + theme));
  }

  m = v.match(/^color-mix\(\s*in\s+srgb\s*,([\s\S]+)\)$/);
  if (m) {
    const [aRaw, bRaw] = splitTop(m[1]);
    if (!aRaw || !bRaw) return null;
    const pct = (s) => {
      const p = s.match(/(\d+(?:\.\d+)?)%\s*$/);
      return p ? Number(p[1]) / 100 : null;
    };
    const strip = (s) => s.replace(/\s*\d+(?:\.\d+)?%\s*$/, '').trim();
    const pa = pct(aRaw);
    const a = resolveColor(strip(aRaw), theme, seen);
    const b = resolveColor(strip(bRaw), theme, seen);
    if (!a || !b || a === 'gradient' || b === 'gradient' || pa === null) return null;
    const w = pa;
    const aa = a[3] ?? 1;
    const ab = b[3] ?? 1;
    const alpha = aa * w + ab * (1 - w);
    // Non-premultiplied is close enough for a contrast gate, and exact when
    // both sides are opaque (the case for every mix in this stylesheet).
    return [0, 1, 2].map((i) => Math.round(a[i] * w + b[i] * (1 - w))).concat(alpha);
  }

  return null;
}

// ── 1. plumbing ──────────────────────────────────────────────────

const store = readFileSync(join(THEME_DIR, 'store.ts'), 'utf8');
const applier = readFileSync(join(THEME_DIR, 'ThemeApplier.tsx'), 'utf8');
const layout = readFileSync(join(SRC, 'app', '[locale]', 'layout.tsx'), 'utf8');

const storeKey = store.match(/THEME_KEY\s*=\s*'([^']+)'/)?.[1];
if (!storeKey) fail('shared/theme/store.ts', 'THEME_KEY is not a string literal');
else if (!applier.includes(`'${storeKey}'`)) {
  fail('shared/theme/ThemeApplier.tsx', `themeInitScript does not read '${storeKey}' — the pre-paint theme and the store would disagree`);
}
if (!/dataset\.theme|setAttribute\(\s*'data-theme'/.test(applier)) {
  fail('shared/theme/ThemeApplier.tsx', 'themeInitScript never writes data-theme, so dark users get a light flash');
}
// The OS must NOT reach the theme: light is the default until the user turns
// dark mode on in Profile. A `prefers-color-scheme` read creeping back into the
// theme slice would silently restore "follow the system".
if (/prefers-color-scheme/.test(applier) || /prefers-color-scheme/.test(store)) {
  fail('shared/theme', 'the theme slice reads prefers-color-scheme — the OS setting must not decide the app theme (default is light)');
}
if (!/'dark'/.test(applier)) {
  fail('shared/theme/ThemeApplier.tsx', 'themeInitScript never checks for the stored dark value');
}
if (!layout.includes('themeInitScript')) {
  fail('app/[locale]/layout.tsx', 'themeInitScript is not rendered — nothing sets the theme before hydration');
}
if (!/theme-color/.test(store)) {
  fail('shared/theme/store.ts', 'nothing rewrites <meta name="theme-color">, so the status bar keeps following the OS');
}
if (!/^\s*color-scheme:\s*light/m.test(light?.body ?? '')) {
  fail('globals.css :root', 'no `color-scheme: light` — the browser paints its own widgets (scrollbars, form controls) light-only');
}
if (!/^\s*color-scheme:\s*dark/m.test(dark?.body ?? '')) {
  fail('globals.css [data-theme="dark"]', 'no `color-scheme: dark` — scrollbars and native controls stay light on a dark page');
}

// ── 2. token parity ──────────────────────────────────────────────

/**
 * Tokens that are the same in both themes ON PURPOSE. Anything not listed here
 * and not derived from a flipping token must appear in the dark block.
 */
const THEME_STABLE = {
  '--on-accent': 'text on a saturated fill; the fill stays saturated in dark',
  '--grad-start': 'the brand gradient is identical in both themes by design',
  '--grad-end': 'the brand gradient is identical in both themes by design',
  '--gradient-brand': 'derived from --grad-start/--grad-end',
  '--pink': 'reads at 6.8:1 on the dark surface as-is',
  '--data': 'turquoise already clears 10:1 on the dark surface',
  '--green': 'clears 6.3:1 on the dark surface',
  '--green-dot': 'clears 8:1 on the dark surface',
  '--green-mid': 'clears 7.4:1 on the dark surface',
  '--amber': 'clears 8.6:1 on the dark surface',
  '--blue': 'clears 8:1 on the dark surface',
  '--blush': 'clears 8.3:1 on the dark surface',
  '--purple': 'the far end of the brand gradient; stable with it',
  '--brand-fill': 'a fill under white text; lifting it would sink white-on-brand',
};

if (light && dark) {
  for (const [name, value] of light.map) {
    if (dark.map.has(name)) continue;
    if (THEME_STABLE[name]) continue;
    // Derived: if the value only references other tokens and at least one of
    // them flips, the derived value flips with it.
    const refs = [...value.matchAll(/var\(\s*(--[\w-]+)/g)].map((m) => m[1]);
    if (refs.length && refs.some((r) => dark.map.has(r))) continue;
    fail(
      `globals.css ${name}`,
      `declared in :root but never in [data-theme="dark"] (value ${value}). Add a dark value, or list it in THEME_STABLE with the reason.`,
    );
  }
  // The reverse mistake: a dark-only token nothing declares a light value for.
  for (const name of dark.map.keys()) {
    if (name === 'color-scheme') continue;
    if (!light.map.has(name)) {
      fail(`globals.css ${name}`, 'declared only in [data-theme="dark"] — light mode resolves it to nothing');
    }
  }
}

// ── 3. contrast ──────────────────────────────────────────────────

/**
 * [foreground, background, floor, label].
 *
 * The FLOOR is what light mode already achieves, rounded down: this gate is a
 * ratchet on dark mode, not a retrofit of WCAG onto the brand's existing light
 * design. Ten of these pairs are below WCAG AA in light mode today — they are
 * reported as warnings (see AA_TARGET) so they stay visible, and the floors
 * keep them from getting any worse.
 *
 * AA for reference: 4.5 body text, 3.0 large/bold text and UI accents.
 */
// What WCAG asks of this pair, read off the label: anything described as text
// owes 4.5:1, an accent or a bold label on a fill owes 3.0:1, and a hairline or
// a day tint is not text at all — its floor is the whole requirement.
const AA_TARGET = (label, floor) => {
  if (/hairline|rail|border|standing off|day tint/.test(label)) return floor;
  if (/\btext\b|heading/.test(label)) return 4.5;
  return 3.0;
};
const PAIRS = [
  ['--ink', '--surface', 4.5, 'body text on a card'],
  ['--ink', '--page', 4.5, 'body text on the canvas'],
  ['--ink-2', '--surface', 4.5, 'strong text on a card'],
  ['--ink-3', '--surface', 4.5, 'medium text on a card'],
  ['--muted', '--surface', 4.5, 'secondary text on a card'],
  ['--muted-2', '--surface', 4.5, 'secondary label on a card'],
  ['--muted-3', '--surface', 3.0, 'tertiary text on a card'],
  ['--muted-soft', '--surface', 2.2, 'placeholder / disabled text'],
  ['--ink', '--surface-2', 4.5, 'text on the faint panel'],
  ['--ink', '--surface-3', 4.5, 'text on the inner panel'],
  ['--muted', '--page', 4.3, 'secondary text on the canvas'],

  ['--brand', '--surface', 4.1, 'brand text/icon on a card'],
  ['--brand', '--page', 3.6, 'brand text/icon on the canvas'],
  ['--brand', '--pink-bg', 3.0, 'brand glyph on its own soft tint'],
  ['--brand-deep', '--surface', 4.5, 'brand heading on a card'],
  ['--brand-strong', '--surface', 4.5, 'strongest brand text'],
  ['--on-accent', '--brand-fill', 3.0, 'white label on a brand fill'],
  ['--on-accent', '--grad-start', 3.0, 'white label on the gradient (start)'],
  ['--on-accent', '--grad-end', 2.5, 'white label on the gradient (end)'],

  ['--data-deep', '--surface', 2.9, 'data text on a card'],
  ['--data-deep', '--data-soft', 2.7, 'data text on its own tint'],
  ['--period-deep', '--surface', 4.5, 'period text on a card'],
  ['--period-deep', '--period-soft', 4.3, 'period text on a period day cell'],
  ['--period', '--surface', 3.0, 'period marker on a card'],
  ['--danger-deep', '--surface', 4.5, 'error text on a card'],
  ['--danger-deep', '--danger-soft', 4.5, 'error text on its own tint'],
  ['--danger', '--surface', 3.0, 'danger icon on a card'],
  ['--green-deep', '--surface', 4.5, 'success text on a card'],
  ['--green-deep', '--green-tint', 4.5, 'success text on its own tint'],
  ['--amber-deep', '--surface', 4.5, 'warning text on a card'],
  ['--amber-deep', '--amber-soft', 4.5, 'warning text on its own tint'],
  ['--teal-deep', '--teal-soft', 3.8, 'category text on its own tint'],
  ['--indigo-deep', '--indigo-soft', 4.3, 'category text on its own tint'],
  ['--rose-deep', '--surface', 3.0, 'rose accent on a card'],

  ['--line', '--surface', 1.06, 'hairline on a card'],
  ['--line', '--page', 1.06, 'hairline on the canvas'],
  ['--track', '--surface', 1.15, 'progress rail on a card'],
  ['--field-border', '--surface', 1.06, 'input border on a card'],
  ['--surface', '--page', 1.03, 'card standing off the canvas'],
];

/** The calendar marker ramp — every step must stay distinct from the surface. */
for (const t of ['--pink-bg', '--amber-soft', '--green-tint', '--violet-soft', '--data-soft', '--period-soft']) {
  for (const step of ['-faint', '-strong', '']) {
    PAIRS.push([`${t}${step}`, '--surface', 1.02, 'calendar day tint against the card']);
  }
}

if (light && dark) {
  for (const [fgName, bgName, min, label] of PAIRS) {
    const scores = {};
    let broken = false;
    for (const theme of ['light', 'dark']) {
      const bgRaw = resolveColor(`var(${bgName})`, theme);
      const fgRaw = resolveColor(`var(${fgName})`, theme);
      if (!bgRaw || !fgRaw || bgRaw === 'gradient' || fgRaw === 'gradient') {
        fail(`contrast ${fgName} on ${bgName}`, `cannot resolve in ${theme} mode — is the token declared?`);
        broken = true;
        break;
      }
      // A translucent token is judged as it renders: composited on its backdrop.
      const bg = over(bgRaw, theme === 'dark' ? [19, 16, 34] : [255, 255, 255]);
      scores[theme] = contrast(over(fgRaw, bg), bg);
    }
    if (broken) continue;

    const aa = AA_TARGET(label, min);
    for (const theme of ['light', 'dark']) {
      if (scores[theme] < min) {
        fail(
          `contrast ${fgName} on ${bgName} (${theme})`,
          `${scores[theme].toFixed(2)}:1 is below the ${min}:1 floor — ${label}`,
        );
      } else if (scores[theme] < aa) {
        warn(
          `contrast ${fgName} on ${bgName} (${theme})`,
          `${scores[theme].toFixed(2)}:1 is under WCAG AA (${aa}:1) — ${label}`,
        );
      }
    }
    // Dark must not be a downgrade even where both clear the floor.
    if (scores.dark < scores.light * 0.85 && scores.light >= min) {
      fail(
        `contrast ${fgName} on ${bgName}`,
        `dark ${scores.dark.toFixed(2)}:1 is materially worse than light ${scores.light.toFixed(2)}:1 — ${label}`,
      );
    }
    if (VERBOSE) {
      notes.push(`  ${fgName} on ${bgName}: light ${scores.light.toFixed(2)} / dark ${scores.dark.toFixed(2)}  (${label})`);
    }
  }
}

// ── 4. literals in rule bodies ───────────────────────────────────

/**
 * Selectors whose backdrop is a saturated brand fill in BOTH themes, so a
 * translucent-white overlay on them is correct rather than a light-mode
 * assumption. Matched as a prefix of the selector text.
 */
const ON_GRADIENT = ['.home-hero', '.home-ring', '.home-highlight', '.home-phase', '.home-more', '.pd-hero', '.preg-hero', '.preg-prog'];

const COLOUR_PROPS = /(?:^|[;{\s])(background|background-color|color|border-color|border|outline-color|fill|stroke)\s*:\s*([^;}]+)/g;
const HAS_LITERAL = /#[0-9a-fA-F]{3,8}\b|rgba?\(|hsla?\(/;

{
  const lines = css.split('\n');
  // Line ranges of the two token blocks — declarations there are the answer,
  // not the problem.
  const blockRanges = [];
  for (const sel of [':root', '[data-theme="dark"]']) {
    let depth = 0;
    let start = -1;
    lines.forEach((l, i) => {
      if (start === -1 && l.trim().startsWith(sel) && l.includes('{')) {
        start = i;
        depth = 0;
      }
      if (start !== -1) {
        depth += (l.match(/\{/g) || []).length - (l.match(/\}/g) || []).length;
        if (depth === 0 && i > start) {
          blockRanges.push([start, i]);
          start = -1;
        }
      }
    });
  }
  const inTokenBlock = (i) => blockRanges.some(([a, b]) => i >= a && i <= b);

  let selector = '';
  lines.forEach((line, idx) => {
    if (line.includes('{')) selector = line.split('{')[0].trim() || selector;
    if (inTokenBlock(idx)) return;
    if (line.trim().startsWith('--')) return;
    for (const m of line.matchAll(COLOUR_PROPS)) {
      const value = m[2].trim();
      if (!HAS_LITERAL.test(value)) continue;
      // Masks and gradient stops use #000/#fff as alpha maths, not as colour.
      if (/^(?:-webkit-)?mask/.test(m[1])) continue;
      if (ON_GRADIENT.some((p) => selector.includes(p))) continue;
      fail(
        `globals.css:${idx + 1}`,
        `\`${m[1]}: ${value.slice(0, 60)}\` on \`${selector}\` is a colour literal — it cannot flip. Use a token or color-mix(), or add the selector to ON_GRADIENT if it really does sit on a saturated fill.`,
      );
    }
  });
}

// ── 5. literals in components ────────────────────────────────────

function walk(dir) {
  const out = [];
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) out.push(...walk(p));
    else if (/\.tsx?$/.test(name) && !/\.(test|spec)\.tsx?$/.test(name)) out.push(p);
  }
  return out;
}

/**
 * `<meta name="theme-color">` cannot hold a CSS variable, so the layout's two
 * media-scoped hex values are the one sanctioned literal in the app layer; the
 * store rewrites them at runtime. Anything else must be a token.
 */
const LITERAL_EXEMPT = new Set(['src/app/[locale]/layout.tsx', 'src/app/manifest.ts']);
// Shadows and glows are alpha over whatever is behind them: they read in both
// themes, and a "dark shadow" is still the right shadow on a dark page.
const SHADOW_LINE = /(box-?[Ss]hadow|text-?[Ss]hadow|drop-?[Ss]hadow|filter)/;

for (const file of walk(SRC)) {
  const rel = relative(ROOT, file);
  if (LITERAL_EXEMPT.has(rel)) continue;
  const src = readFileSync(file, 'utf8')
    .replace(/\/\*[\s\S]*?\*\//g, (m) => ' '.repeat(m.length))
    .replace(/(^|[^:])\/\/[^\n]*/g, (m, p) => p + ' '.repeat(m.length - p.length));
  src.split('\n').forEach((line, i) => {
    if (SHADOW_LINE.test(line)) return;
    const m = line.match(/#[0-9a-fA-F]{6}\b|#[0-9a-fA-F]{3}\b|rgba?\(\s*\d/);
    if (m) fail(`${rel}:${i + 1}`, `colour literal \`${m[0]}\` — it does not flip in dark mode; use var(--token)`);
  });
}

// ── report ───────────────────────────────────────────────────────

if (VERBOSE && notes.length) {
  console.log('contrast (foreground on background):');
  for (const n of notes) console.log(n);
  console.log('');
}

if (warnings.length) {
  console.warn(`⚠ ${warnings.length} contrast pair${warnings.length === 1 ? '' : 's'} under WCAG AA (pre-existing in the brand palette — not a regression):\n`);
  for (const w of warnings) console.warn(`  ${w}`);
  console.warn('');
}

if (problems.length) {
  console.error(`✖ dark-mode gate: ${problems.length} problem${problems.length === 1 ? '' : 's'}\n`);
  for (const p of problems) console.error(`  ${p}`);
  console.error('\nSee CLAUDE.md §10.2. Every colour token needs a value in both');
  console.error(':root and [data-theme="dark"], and dark may not read worse than light.\n');
  process.exit(1);
}

console.log(`✔ dark-mode gate passed (${PAIRS.length} contrast pairs, ${light.map.size} tokens)`);
