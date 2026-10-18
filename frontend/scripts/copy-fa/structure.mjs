// Structure gate: key tree, ICU signature and parseability vs base / en.
import { flatten, icuSignature, assertParses } from './lib.mjs';

const same = (a, b) => a.length === b.length && a.every((x, i) => x === b[i]);

/** @returns {{key:string, rule:string, value:string}[]} */
export function compareStructure(cur, ref, label) {
  const errs = [];
  const c = flatten(cur);
  const r = flatten(ref);
  for (const k of Object.keys(r)) if (!(k in c)) errs.push({ key: k, rule: `structure: key missing vs ${label}`, value: '' });
  for (const k of Object.keys(c)) if (!(k in r)) errs.push({ key: k, rule: `structure: extra key vs ${label}`, value: String(c[k]) });
  return errs;
}

/** ICU gate for current values (vs base values when a base exists). */
export function checkIcu(cur, base) {
  const errs = [];
  const c = flatten(cur);
  const b = base ? flatten(base) : null;
  for (const [k, v] of Object.entries(c)) {
    if (typeof v !== 'string') {
      errs.push({ key: k, rule: 'structure: value is not a string', value: String(v) });
      continue;
    }
    let sig;
    try {
      assertParses(v);
      sig = icuSignature(v);
    } catch (e) {
      errs.push({ key: k, rule: `icu: does not parse (${String(e.message).split('\n')[0]})`, value: v });
      continue;
    }
    if (b && typeof b[k] === 'string') {
      let bsig;
      try { bsig = icuSignature(b[k]); } catch { continue; }
      if (!same(sig, bsig)) {
        const miss = bsig.filter((x) => !sig.includes(x));
        const extra = sig.filter((x) => !bsig.includes(x));
        errs.push({ key: k, rule: `icu: signature differs from base (missing ${JSON.stringify(miss)}, extra ${JSON.stringify(extra)})`, value: v });
      }
    }
  }
  return errs;
}
