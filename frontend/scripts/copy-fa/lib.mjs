// Shared helpers for the copy-fa tooling (T-M8-02). Plain Node ESM, no new deps.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';
import { parse, TYPE } from '@formatjs/icu-messageformat-parser';
import { IntlMessageFormat } from 'intl-messageformat';

const here = path.dirname(fileURLToPath(import.meta.url));
export const FRONTEND_DIR = path.resolve(here, '../..');
export const REPO_ROOT = path.resolve(FRONTEND_DIR, '..');
export const FA_DIR = path.join(FRONTEND_DIR, 'messages/fa');
export const EN_DIR = path.join(FRONTEND_DIR, 'messages/en');
export const BACKEND_FA_DIR = path.join(REPO_ROOT, 'backend-go/resources/translations/fa');
export const GLOSSARY_PATH = path.join(REPO_ROOT, 'docs/copy-fa/GLOSSARY.md');

/** Persian letters (no diacritics / ezafe hamza / ZWNJ / digits). */
export const PL = '\\u0621-\\u063A\\u0641-\\u064A\\u0670-\\u06D3';
export const HAS_PERSIAN = new RegExp(`[${PL}]`);
export const ZWNJ = '‌';
/** Placeholder for an ICU argument / tag in flattened prose. */
export const ARG = '￼';

export function listNamespaces(dir = FA_DIR) {
  return fs.readdirSync(dir).filter((f) => f.endsWith('.json')).map((f) => f.slice(0, -5)).sort();
}

export function readJson(file) {
  return JSON.parse(fs.readFileSync(file, 'utf8'));
}

/** Leaf map {dotted.path: value} (arrays use numeric indexes). */
export function flatten(obj, prefix = '', out = {}) {
  if (obj !== null && typeof obj === 'object') {
    for (const [k, v] of Object.entries(obj)) flatten(v, prefix ? `${prefix}.${k}` : k, out);
  } else {
    out[prefix] = obj;
  }
  return out;
}

export function gitShow(ref, relPath) {
  const r = spawnSync('git', ['show', `${ref}:${relPath}`], {
    cwd: REPO_ROOT, encoding: 'utf8', maxBuffer: 64 * 1024 * 1024,
  });
  return r.status === 0 ? r.stdout : null;
}

// ---------------------------------------------------------------- ICU

export function parseIcu(value) {
  return parse(value, { requiresOtherClause: false, captureLocation: true });
}

/** Throws when the message is not valid ICU (intl-messageformat is the runtime used by next-intl). */
export function assertParses(value) {
  new IntlMessageFormat(value, 'fa');
}

const TYPE_NAMES = {
  [TYPE.argument]: 'argument', [TYPE.number]: 'number', [TYPE.date]: 'date',
  [TYPE.time]: 'time', [TYPE.select]: 'select', [TYPE.plural]: 'plural',
};

/**
 * Structural signature of a message: sorted unique descriptors of every argument
 * (name:type), plural/select arm keys and rich-text tag names. Words are ignored.
 */
export function icuSignature(value) {
  const sig = new Set();
  const walk = (els) => {
    for (const el of els) {
      switch (el.type) {
        case TYPE.literal: break;
        case TYPE.pound: break;
        case TYPE.tag:
          sig.add(`tag:${el.value}`);
          walk(el.children);
          break;
        case TYPE.select:
        case TYPE.plural: {
          const t = el.type === TYPE.select ? 'select' : (el.pluralType === 'ordinal' ? 'selectordinal' : 'plural');
          sig.add(`${t}:${el.value}`);
          const arms = Object.keys(el.options).sort();
          sig.add(`arms:${el.value}:[${arms.join(',')}]`);
          if (el.offset) sig.add(`offset:${el.value}:${el.offset}`);
          for (const o of Object.values(el.options)) walk(o.value);
          break;
        }
        default:
          sig.add(`${TYPE_NAMES[el.type] ?? el.type}:${el.value}`);
      }
    }
  };
  walk(parseIcu(value));
  return [...sig].sort();
}

/** Prose with every ICU argument replaced by ARG (tags dropped); falls back to the raw value if unparsable. */
export function flattenProse(value) {
  let ast;
  try { ast = parseIcu(value); } catch { return value; }
  const walk = (els) => {
    let s = '';
    for (const el of els) {
      if (el.type === TYPE.literal) s += el.value;
      else if (el.type === TYPE.tag) s += walk(el.children);
      else if (el.type === TYPE.select || el.type === TYPE.plural) {
        // arms are alternatives of the same sentence: lint each one on its own, separated by a line break
        s += ARG + Object.values(el.options).map((o) => '\n' + walk(o.value)).join('') + '\n';
      } else s += ARG;
    }
    return s;
  };
  return walk(ast);
}

/**
 * Split a message into [{text, prose}] chunks: prose = literal text (translatable),
 * non-prose = ICU syntax (`{n, plural, other{`, `#`, `}}`, `<b>`, `</b>`) kept verbatim.
 */
export function splitIcu(value) {
  const ast = parseIcu(value);
  const ranges = [];
  const walk = (els) => {
    for (const el of els) {
      if (el.type === TYPE.literal) ranges.push([el.location.start.offset, el.location.end.offset]);
      else if (el.type === TYPE.tag) walk(el.children);
      else if (el.type === TYPE.select || el.type === TYPE.plural) {
        for (const o of Object.values(el.options)) walk(o.value);
      }
    }
  };
  walk(ast);
  ranges.sort((a, b) => a[0] - b[0]);
  const chunks = [];
  let pos = 0;
  for (const [s, e] of ranges) {
    if (s > pos) chunks.push({ text: value.slice(pos, s), prose: false });
    chunks.push({ text: value.slice(s, e), prose: true });
    pos = e;
  }
  if (pos < value.length) chunks.push({ text: value.slice(pos), prose: false });
  return chunks;
}
