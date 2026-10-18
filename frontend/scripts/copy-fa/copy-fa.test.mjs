import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import { lintValue } from './lint.mjs';
import { compareStructure, checkIcu } from './structure.mjs';
import { protect, restore } from './protect.mjs';
import { icuSignature } from './lib.mjs';
import { parseGlossary, compileGlossary } from './glossary.mjs';
import { cleanObject } from './clean.mjs';

const rules = (v, opts) => lintValue('k', v, opts).map((x) => x.rule);
const errs = (v) => lintValue('k', v).filter((x) => x.level === 'error').map((x) => x.rule);

// ------------------------------------------------------------------ lint: one positive + one negative per rule
test('arabic-chars', () => {
  assert.ok(errs('كتاب خوب').includes('arabic-chars'));
  assert.deepEqual(errs('کتاب خوب'), []);
});
test('arabic-digits', () => {
  assert.ok(errs('بعد از ٤٨ ساعت').includes('arabic-digits'));
  assert.deepEqual(errs('بعد از ۴۸ ساعت ۲۰٪'), []);
});
test('dash', () => {
  assert.ok(errs('خاموش — اعلان‌ها').includes('dash'));
  assert.ok(errs('{from} – امروز').includes('dash'));
  assert.deepEqual(errs('—'), []); // placeholder value
  assert.deepEqual(errs('بازه ({min}–{max}) روز'), []); // numeric range
});
test('mibashad / lazem / rastaye', () => {
  assert.ok(errs('این کار انجام می‌باشد').includes('mibashad'));
  assert.ok(errs('لازم به ذکر است که').includes('lazem-be-zekr'));
  assert.ok(errs('در راستای هدف').includes('dar-rastaye'));
  assert.deepEqual(errs('انجام می‌شود و باید باشد'), []);
});
test('space-before-punct', () => {
  assert.ok(errs('سلام ، دنیا').includes('space-before-punct'));
  assert.ok(errs('خوبی ؟').includes('space-before-punct'));
  assert.deepEqual(errs('سلام، دنیا؛ خوبی؟'), []);
});
test('ascii-quote only in Persian prose', () => {
  assert.ok(errs('دکمهٔ "ذخیره" را بزن').includes('ascii-quote'));
  assert.deepEqual(errs('دکمهٔ «ذخیره» را بزن'), []);
  assert.deepEqual(errs('say "hello"'), []);
});
test('mi-space needs «می »/«نمی » + word and spares می‌ and می-words', () => {
  assert.ok(errs('او می رود').includes('mi-space'));
  assert.ok(errs('نمی شود').includes('mi-space'));
  assert.deepEqual(errs('می‌رود و نمی‌شود'), []);
  for (const w of ['میز', 'میوه', 'میان', 'میانگین', 'میل', 'میلی‌گرم', 'میزان', 'اسمی بود', 'میز می‌ماند']) {
    assert.deepEqual(errs(w), [], w);
  }
});
test('ha-space', () => {
  assert.ok(errs('همه کتاب ها را ببین').includes('ha-space'));
  assert.deepEqual(errs('همه کتاب‌ها را ببین، هاله و هاشور'), []);
});

// warnings
test('colloquial warn', () => {
  assert.ok(rules('چرخه رو ببین').includes('colloquial'));
  assert.ok(rules('نمیشه ثبت کرد').includes('colloquial'));
  assert.ok(!rules('چرخه‌ات را ببین. رو به اتمام است').includes('colloquial'));
});
test('latin-digits warn spares codes', () => {
  assert.ok(rules('بعد از 48 ساعت').includes('latin-digits'));
  assert.ok(!rules('آزمایش HbA1c و B12 و mg/dL').includes('latin-digits'));
});
test('longer-than-base warn on button-like keys', () => {
  assert.ok(lintValue('cta', 'ذخیره کن و ادامه بده حالا', { base: 'ذخیره' }).some((x) => x.rule === 'longer-than-base'));
  assert.ok(!lintValue('title', 'ذخیره کن و ادامه بده حالا', { base: 'ذخیره' }).some((x) => x.rule === 'longer-than-base'));
});
test('glossary warns on whole words only, honours allowed phrases', () => {
  const g = compileGlossary(parseGlossary());
  const has = (v, c) => rules(v, { glossary: g }).includes(`glossary:${c}`);
  assert.ok(has('روزهای قاعدگی', 'پریود'));
  assert.ok(has('قاعدگی‌ات دیر شد', 'پریود'));
  assert.ok(!has('تاریخ آخرین قاعدگی', 'پریود'));
  assert.ok(!has('پیش از قاعدگی', 'پریود'));
  assert.ok(!has('اختلال قاعدگی', 'پریود'));
  assert.ok(!has('چرخهٔ قاعدگی', 'پریود'));
  assert.ok(has('سیکل تو', 'چرخه'));
  assert.ok(!has('پریود و چرخه', 'چرخه'));
  assert.ok(has('با دکتر صحبت کن', 'پزشک'));
  assert.ok(!has('سونوگرافی', 'سونوگرافی'));
});

// ------------------------------------------------------------------ structure gate
const base = { a: 'سلام {name}', n: '{n, plural, =0{هیچ} other{# روز}}', t: 'کلیک <b>کن</b>', g: '{x, select, female{او} other{آن}}', nested: { k: 'x' } };
test('structure: identical passes', () => {
  assert.deepEqual(compareStructure(base, structuredClone(base), 'HEAD'), []);
  assert.deepEqual(checkIcu(structuredClone(base), base), []);
});
test('structure: renamed key fails', () => {
  const cur = structuredClone(base); cur.b = cur.a; delete cur.a;
  assert.equal(compareStructure(cur, base, 'HEAD').length, 2);
});
test('structure: dropped plural arm fails', () => {
  const cur = structuredClone(base); cur.n = '{n, plural, other{# روز}}';
  assert.equal(checkIcu(cur, base).length, 1);
});
test('structure: changed arg name fails', () => {
  const cur = structuredClone(base); cur.a = 'سلام {nam}';
  assert.equal(checkIcu(cur, base).length, 1);
});
test('structure: changed tag / select arm / broken ICU fail', () => {
  const t = structuredClone(base); t.t = 'کلیک <i>کن</i>';
  assert.equal(checkIcu(t, base).length, 1);
  const s = structuredClone(base); s.g = '{x, select, male{او} other{آن}}';
  assert.equal(checkIcu(s, base).length, 1);
  const b = structuredClone(base); b.a = 'سلام {name';
  assert.equal(checkIcu(b, base).length, 1);
});
test('structure: words may change', () => {
  const cur = structuredClone(base); cur.n = '{n, plural, =0{هیچ‌روز} other{# شبانه‌روز}}';
  assert.deepEqual(checkIcu(cur, base), []);
});

// ------------------------------------------------------------------ ICU protection round trip
for (const v of [
  '{n, plural, =0{هیچ} one{# روز} other{# روز}} پیش',
  '{g, select, female{او می رود} other{{n, plural, other{# نفر می روند}}}}',
  'کلیک <b>کن</b> و {name} را <a>ببین</a>',
  'بعد از {n, number} روز، HbA1c و https://x.ir را ببین',
]) {
  test(`protect/restore: ${v.slice(0, 30)}`, () => {
    const p = protect(v);
    assert.ok(p);
    assert.equal(restore(p.masked, p.tokens), v);
    // no ICU syntax leaks into the masked text
    assert.ok(!/[{}<>#]/.test(p.masked.replace(/https?:\S+/g, '')), p.masked);
    assert.deepEqual(icuSignature(restore(p.masked, p.tokens)), icuSignature(v));
  });
}
test('restore rejects lost / reordered sentinels', () => {
  const p = protect('{a} و {b}');
  assert.equal(restore(p.masked.slice(1), p.tokens), null);
  assert.equal(restore([...p.masked].reverse().join(''), p.tokens), null);
});
test('protect leaves non-Persian values alone', () => {
  assert.equal(protect('mg/dL'), null);
});

// ------------------------------------------------------------------ end to end with the real cleaner (optional)
const cleaner = process.env.PERSIAN_CLEANUP;
test('clean: keeps ICU, spaces, compound verbs and fixes ZWNJ', { skip: !(cleaner && fs.existsSync(cleaner)) && 'PERSIAN_CLEANUP not set' }, () => {
  const obj = { a: '{n, plural, other{# روز}} می رود', b: ' و ', c: 'باز کردن HbA1c', d: 'ي', e: { f: 'سلام {name}، "خوبی" ؟' } };
  const before = structuredClone(obj);
  const { changes } = cleanObject(obj, cleaner);
  assert.equal(obj.b, ' و ');
  assert.equal(obj.c, 'باز کردن HbA1c');
  assert.match(obj.a, /^\{n, plural, other\{# روز\}\} می‌رود$/);
  assert.equal(obj.d, 'ی');
  assert.ok(obj.e.f.includes('{name}'));
  assert.deepEqual(Object.keys(obj), Object.keys(before));
  assert.ok(changes.length >= 3);
});
