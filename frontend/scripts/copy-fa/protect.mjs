// ICU-safe protection: swap ICU syntax (and Latin technical tokens) for sentinels so the
// prose can go through persian_cleanup.py, then swap back.
import { splitIcu, HAS_PERSIAN } from './lib.mjs';

const BASE = 0xe000;
export const SEPARATOR = '';
const TECH = /https?:\/\/\S+|[\w.+-]+@[\w-]+\.[\w.-]+|[A-Za-z][A-Za-z0-9._/-]*\d[A-Za-z0-9._/-]*|\d[A-Za-z0-9._/-]*[A-Za-z][A-Za-z0-9._/-]*/g;

/** @returns {{masked:string, tokens:string[]}|null} null when the value must be left alone */
export function protect(value) {
  if (!HAS_PERSIAN.test(value)) return null;
  if (/[-]/.test(value)) return null;
  let chunks;
  try { chunks = splitIcu(value); } catch { return null; }
  const tokens = [];
  const stash = (s) => {
    tokens.push(s);
    return String.fromCharCode(BASE + tokens.length - 1);
  };
  let masked = '';
  for (const c of chunks) {
    masked += c.prose ? c.text.replace(TECH, stash) : stash(c.text);
  }
  if (tokens.length > 0x1000) return null;
  return { masked, tokens };
}

/** Restores tokens; returns null when a sentinel was lost, duplicated or reordered. */
export function restore(cleaned, tokens) {
  let expected = 0;
  let ok = true;
  const out = cleaned.replace(/[-]/g, (ch) => {
    const i = ch.charCodeAt(0) - BASE;
    if (i !== expected) ok = false;
    expected++;
    return tokens[i] ?? (ok = false, '');
  });
  return ok && expected === tokens.length ? out : null;
}
