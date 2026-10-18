// Lint rules for Persian UI copy. Errors must have zero false positives on correct Persian.
import { PL, ZWNJ, ARG, HAS_PERSIAN, flattenProse } from './lib.mjs';
import { glossaryHits } from './glossary.mjs';

const P = `[${PL}]`;
const NOT_P = `(?<![${PL}${ZWNJ}])`;

export const ERROR_RULES = [
  { id: 'arabic-chars', re: /[يكةى]/, msg: 'Arabic ي/ك/ة/ى' },
  { id: 'arabic-digits', re: /[٠-٩]/, msg: 'Arabic-Indic digits (use Persian ۰-۹)' },
  { id: 'dash', test: (s, raw) => raw !== '—' && /[—–]/.test(s.replace(/(?<=[\d۰-۹\uFFFC])–(?=[\d۰-۹\uFFFC])/g, '')), msg: 'em/en dash in prose (numeric ranges like {min}–{max} are allowed)' },
  { id: 'mibashad', re: new RegExp(`${NOT_P}ن?می[${ZWNJ} ]?باشد`), msg: 'می‌باشد' },
  { id: 'lazem-be-zekr', re: /لازم به ذکر/, msg: 'لازم به ذکر' },
  { id: 'dar-rastaye', re: new RegExp(`${NOT_P}در راستای`), msg: 'در راستای' },
  { id: 'space-before-punct', re: /[ \t][،؛؟]/, msg: 'space before ، ؛ ؟' },
  { id: 'ascii-quote', re: /"/, msg: 'ASCII " inside Persian prose (use «»)' },
  // «می »/«نمی » + word. The lookbehind keeps میز، میوه، اسمی … out; the space (not ZWNJ) is the error.
  { id: 'mi-space', re: new RegExp(`${NOT_P}ن?می (?=${P})`), msg: 'می/نمی followed by a space (needs ZWNJ)' },
  // detached plural «ها»/«های» after a Persian word: «کتاب ها»
  { id: 'ha-space', re: new RegExp(`(?<=${P}) ها(?:ی)?(?![${PL}${ZWNJ}])`), msg: 'plural ها detached (needs ZWNJ)' },
];

const COLLOQUIAL = [
  'ن?می[‌ ]?(?:شه|کنه|ره|گه|ده|خوره|خواد|تونه|تونی|تونم|دونه|دونی|دونم|خوای|ذاره|بینه|گیره|کشه|مونه|گذره|رسه|افته|آد|اد)',
  'اومد(?:ه|ن|ی|م)?', 'یه', 'رو(?! به)', 'برات', 'بهت', 'بهش', 'ازت', 'ازش', 'اون', 'اونا', 'توی', 'دیگه', 'چیه', 'اگه',
  'بذار', 'نمیشه', 'میشه', 'بزار', 'باشه', 'کنه', 'بشه',
];
const COLLOQUIAL_RE = new RegExp(`${NOT_P}(?:${COLLOQUIAL.join('|')})(?![${PL}${ZWNJ}])`);

const LATIN_DIGIT_MASK = /[A-Za-z][A-Za-z0-9._/-]*\d[A-Za-z0-9._/-]*|\d[A-Za-z0-9._/-]*[A-Za-z][A-Za-z0-9._/-]*|https?:\/\/\S+/g;
const BUTTONISH = /(cta|button|tab|chip|label|action)/i;

/** @returns {{level:'error'|'warn', rule:string}[]} */
export function lintValue(key, value, { base, glossary } = {}) {
  const out = [];
  if (typeof value !== 'string' || !HAS_PERSIAN.test(value)) return out;
  const prose = flattenProse(value);
  for (const r of ERROR_RULES) {
    if (r.test ? r.test(prose, value) : r.re.test(prose)) out.push({ level: 'error', rule: r.id });
  }
  if (COLLOQUIAL_RE.test(prose)) out.push({ level: 'warn', rule: 'colloquial' });
  if (/[0-9]/.test(prose.replace(LATIN_DIGIT_MASK, ' '))) out.push({ level: 'warn', rule: 'latin-digits' });
  if (glossary) {
    for (const h of glossaryHits(prose, glossary)) out.push({ level: 'warn', rule: `glossary:${h.canonical}` });
  }
  const last = key.split('.').pop();
  if (typeof base === 'string' && BUTTONISH.test(last) && value.length > base.length * 1.6 && value.length - base.length >= 4) {
    out.push({ level: 'warn', rule: 'longer-than-base' });
  }
  // rules above never fire on arguments: ARG is neither a letter nor a space
  void ARG;
  return out;
}
