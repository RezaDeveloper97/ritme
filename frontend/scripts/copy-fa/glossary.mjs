// Parses docs/copy-fa/GLOSSARY.md: rejected variants (warn rules) and allowed fixed phrases.
import fs from 'node:fs';
import { GLOSSARY_PATH, PL, ZWNJ, ARG } from './lib.mjs';

const EXTRA_ALLOWED = ['آخرین قاعدگی', 'پیش از قاعدگی', 'اختلال قاعدگی', 'چرخهٔ قاعدگی'];

function stripParens(s) {
  return s.replace(/\([^)]*\)/g, '').replace(/\s+/g, ' ').trim();
}

export function parseGlossary(file = GLOSSARY_PATH) {
  const text = fs.readFileSync(file, 'utf8');
  const rules = []; // {canonical, term}
  const allowed = new Set(EXTRA_ALLOWED);
  for (const line of text.split('\n')) {
    if (!line.startsWith('|')) continue;
    const cells = line.split('|').slice(1, -1).map((c) => c.trim());
    if (cells.length < 4 || /^-+$/.test(cells[0].replace(/[\s:]/g, '')) || cells[0] === 'Concept') continue;
    const canonical = cells[1];
    const rejectedCell = cells[2];
    const notes = cells[3];
    if (!rejectedCell) continue;
    // phrases explicitly exempted in the rejected cell: "(except home/rapid tests: تست بارداری، تست LH)"
    for (const m of rejectedCell.matchAll(/\(except[^:)]*:([^)]*)\)/g)) {
      for (const p of m[1].split(/[،,]/)) if (p.trim()) allowed.add(p.trim());
    }
    // «…» phrases in the notes column are allowed fixed expressions
    for (const m of notes.matchAll(/«([^»]+)»/g)) {
      const p = stripParens(m[1]);
      if (/[؀-ۿ]/.test(p)) allowed.add(p);
    }
    for (const raw of rejectedCell.replace(/\(except[^)]*\)/g, '').split(/[،,]/)) {
      const term = stripParens(raw).replace(/\{[^}]*\}/g, ARG);
      if (!term || !new RegExp(`[${PL}]`).test(term)) continue; // Latin-only notes
      if (/^[A-Za-z]/.test(term)) continue;
      rules.push({ canonical, term });
    }
  }
  // never let a rejected term exempt itself
  const terms = new Set(rules.map((r) => r.term));
  const allowedPhrases = [...allowed].filter((p) => !terms.has(p));
  return { rules, allowedPhrases };
}

const escapeRe = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
const SUFFIXES = ['ها', 'های', 'ات', 'ام', 'اش', 'مان', 'تان', 'شان', 'ای', 'ی', 'تر', 'ترین'];

/** Whole-word matcher: no Persian letter / ZWNJ glued on the left; right side may take a ZWNJ + clitic suffix. */
export function termRegex(term) {
  const body = escapeRe(term);
  const right = `(?:(?=${ZWNJ}(?:${SUFFIXES.join('|')})(?![${PL}]))|(?![${PL}${ZWNJ}]))`;
  return new RegExp(`(?<![${PL}${ZWNJ}])${body}${right}`);
}

export function compileGlossary(g = parseGlossary()) {
  return {
    rules: g.rules.map((r) => ({ ...r, re: termRegex(r.term) })),
    // longest first, so a short phrase («پریود») can't break a longer one («سوابق پریود») before it is masked
    allowed: [...g.allowedPhrases].sort((x, y) => y.length - x.length).map((p) => ({ p, re: new RegExp(escapeRe(p), 'g') })),
  };
}

/** Returns canonical names whose rejected term occurs in prose (allowed phrases masked first). */
export function glossaryHits(prose, compiled) {
  let masked = prose;
  for (const a of compiled.allowed) masked = masked.replace(a.re, (m) => ' '.repeat(m.length));
  const hits = [];
  for (const r of compiled.rules) if (r.re.test(masked)) hits.push(r);
  return hits;
}
