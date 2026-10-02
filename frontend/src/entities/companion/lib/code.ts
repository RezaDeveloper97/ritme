import { COMPANION_CODE_LENGTH } from '../model/companion-side';

const PERSIAN = '۰۱۲۳۴۵۶۷۸۹';
const ARABIC = '٠١٢٣٤٥٦٧٨٩';

/**
 * What the code field keeps from typed or pasted text: Latin letters
 * upper-cased, Persian/Arabic digits read as ASCII, everything else (spaces,
 * dashes, a pasted sentence around the code) dropped, at most 6 characters.
 */
export function normalizeCompanionCode(raw: string): string {
  // A pasted message («کد همدم من: RT7K2X») — take the standalone 6-character token.
  const token = toAscii(raw).match(/(?:^|[^A-Za-z0-9])([A-Za-z0-9]{6})(?![A-Za-z0-9])/);
  if (token && /[^A-Za-z0-9]/.test(raw.trim())) return token[1]!.toUpperCase();
  let out = '';
  for (const ch of toAscii(raw)) {
    const c = ch.toUpperCase();
    if (/^[A-Z0-9]$/.test(c)) out += c;
    if (out.length === COMPANION_CODE_LENGTH) break;
  }
  return out;
}

function toAscii(raw: string): string {
  let out = '';
  for (const ch of raw) {
    const p = PERSIAN.indexOf(ch);
    const a = ARABIC.indexOf(ch);
    out += p >= 0 ? String(p) : a >= 0 ? String(a) : ch;
  }
  return out;
}

export function isCompleteCompanionCode(code: string): boolean {
  return code.length === COMPANION_CODE_LENGTH;
}
