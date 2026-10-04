// node --test tests/js — resources/js/lib/jalali.js against the vectors shared with tests/Unit/Tools (PHP).
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import {
    day,
    evaluate,
    formatDay,
    fromDayNumber,
    fromGregorian,
    isLeapYear,
    iso,
    monthLength,
    parseCycle,
    parseDate,
    toDayNumber,
    toGregorian,
    today,
} from '../../resources/js/lib/jalali.js';

const vectors = JSON.parse(readFileSync(new URL('./fixtures/calculator-vectors.json', import.meta.url), 'utf8'));
const ymd = (s) => s.split('-').map(Number);

test('calendar vectors: Jalali ⇄ Gregorian and day numbers', () => {
    for (const v of vectors.calendar) {
        const [jy, jm, jd] = ymd(v.jalali);
        const [gy, gm, gd] = ymd(v.gregorian);
        assert.deepEqual(toGregorian(jy, jm, jd), { gy, gm, gd }, v.jalali);
        assert.equal(iso(fromGregorian(gy, gm, gd)), v.jalali, v.gregorian);
        assert.equal(toDayNumber(jy, jm, jd), v.jdn, v.jalali);
        assert.equal(iso(fromDayNumber(v.jdn)), v.jalali);
    }
});

test('leap years (1403 and 1408 are leap, 1402/1404 are not)', () => {
    for (const [year, leap] of Object.entries(vectors.leapYears)) {
        assert.equal(isLeapYear(+year), leap, year);
        assert.equal(monthLength(+year, 12), leap ? 30 : 29, year);
    }
    assert.equal(isLeapYear(1403), true);
    assert.equal(isLeapYear(1408), true);
    assert.equal(parseDate('1403/12/30')?.n, toDayNumber(1403, 12, 30));
    assert.equal(parseDate('1408/12/30')?.n, toDayNumber(1408, 12, 30));
    assert.equal(parseDate('1404/12/30'), null);
});

test('day arithmetic crosses month and year ends', () => {
    const d = day(1403, 12, 30);
    assert.equal(iso(fromDayNumber(d.n + 1)), '1404-01-01');
    assert.equal(iso(fromDayNumber(day(1405, 6, 31).n + 1)), '1405-07-01');
});

test('input parsing', () => {
    for (const text of ['۱۴۰۵/۰۲/۱۲', '1405-2-12', '1405.02.12', '۱۲ اردیبهشت ۱۴۰۵', '  ۱۲   ارديبهشت ۱۴۰۵ ', '١٤٠٥/٠٢/١٢']) {
        assert.equal(iso(parseDate(text)), '1405-02-12', text);
    }
    for (const text of ['', 'hello', '1405/2', '12 فلان 1405', '1299/12/01', '1500/01/01', '1405/07/31']) {
        assert.equal(parseDate(text), null, text);
    }
    assert.equal(parseCycle('۲۸ روز'), 28);
    assert.equal(parseCycle(''), 28);
    assert.equal(parseCycle('21'), 21);
    assert.equal(parseCycle('45'), 45);
    assert.equal(parseCycle('20'), null);
    assert.equal(parseCycle('46'), null);
    assert.equal(parseCycle('0028'), null);
});

test('formatting uses Persian digits and month names', () => {
    assert.equal(formatDay(day(1405, 11, 17)), '۱۷ بهمن ۱۴۰۵');
    assert.equal(formatDay(day(1405, 7, 16), false), '۱۶ مهر');
});

test('today() reads the local calendar day', () => {
    assert.equal(iso(today(new Date(2026, 9, 4, 23, 30))), '1405-07-12');
});

test('calculation vectors match the PHP implementation', () => {
    for (const v of vectors.calculations) {
        const now = v.today ? day(...ymd(v.today)) : null;
        assert.deepEqual(evaluate(v.kind, v.lmp, v.cycle, vectors.templates, now), v.expect, `${v.kind} ${v.lmp} ${v.cycle} ${v.today}`);
    }
});
