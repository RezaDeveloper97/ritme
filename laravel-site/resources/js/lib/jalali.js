/**
 * Jalali calendar + the /tools calculators, as pure functions (no DOM) — imported only by modules/calculators.js.
 *
 * Twin of the PHP side: App\Support\Jalali\JalaliCalendar (same jalaali-js algorithm), App\Domain\Content\Tools\
 * {CalculatorInput, DueDateCalculator, FertilityWindowCalculator, ResultText}. Change both together; the shared
 * vectors in tests/js/fixtures/calculator-vectors.json are checked by `node --test tests/js` and tests/Unit/Tools.
 *
 * A "day" is { jy, jm, jd, n } where n is the Julian Day Number, so date maths is integer addition.
 */

export const MONTHS = ['فروردین', 'اردیبهشت', 'خرداد', 'تیر', 'مرداد', 'شهریور', 'مهر', 'آبان', 'آذر', 'دی', 'بهمن', 'اسفند'];

export const DEFAULT_CYCLE = 28;
export const MIN_CYCLE = 21;
export const MAX_CYCLE = 45;
export const MIN_YEAR = 1300;
export const MAX_YEAR = 1499;

const PREGNANCY_DAYS = 280;
const RANGE_DAYS = 14;
const PROGRESS_MAX_DAYS = 299;
const LUTEAL_DAYS = 14;
const WINDOW_LEAD_DAYS = 5;

const BREAKS = [-61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210, 1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178];
const FA = '۰۱۲۳۴۵۶۷۸۹';
const AR = '٠١٢٣٤٥٦٧٨٩';

/* ---------- digits ---------- */

export const toPersian = (value) => String(value).replace(/[0-9]/g, (d) => FA[+d]).replace(/[٠-٩]/g, (d) => FA[AR.indexOf(d)]);

export const toLatin = (value) =>
    String(value ?? '')
        .replace(/[۰-۹]/g, (d) => FA.indexOf(d))
        .replace(/[٠-٩]/g, (d) => AR.indexOf(d))
        .replace(/٬/g, ',')
        .replace(/٫/g, '.');

/* ---------- calendar (jalaali-js, Behrang Noruzi Niya; integer division truncates toward zero like PHP intdiv) ---------- */

const div = (a, b) => Math.trunc(a / b);
const mod = (a, b) => a - Math.trunc(a / b) * b;

function cal(jy) {
    const gy = jy + 621;
    let leapJ = -14;
    let jp = BREAKS[0];
    let jump = 0;
    for (let i = 1; i < BREAKS.length; i += 1) {
        const jm = BREAKS[i];
        jump = jm - jp;
        if (jy < jm) break;
        leapJ += div(jump, 33) * 8 + div(mod(jump, 33), 4);
        jp = jm;
    }
    let n = jy - jp;
    leapJ += div(n, 33) * 8 + div(mod(n, 33) + 3, 4);
    if (mod(jump, 33) === 4 && jump - n === 4) leapJ += 1;
    const leapG = div(gy, 4) - div((div(gy, 100) + 1) * 3, 4) - 150;
    const march = 20 + leapJ - leapG;
    if (jump - n < 6) n = n - jump + div(jump + 4, 33) * 33;
    let leap = mod(mod(n + 1, 33) - 1, 4);
    if (leap === -1) leap = 4;
    return { leap, gy, march };
}

function g2d(gy, gm, gd) {
    const d = div((gy + div(gm - 8, 6) + 100100) * 1461, 4) + div(153 * mod(gm + 9, 12) + 2, 5) + gd - 34840408;
    return d - div(div(gy + 100100 + div(gm - 8, 6), 100) * 3, 4) + 752;
}

function d2g(jdn) {
    let j = 4 * jdn + 139361631;
    j += div(div(4 * jdn + 183187720, 146097) * 3, 4) * 4 - 3908;
    const i = div(mod(j, 1461), 4) * 5 + 308;
    const gd = div(mod(i, 153), 5) + 1;
    const gm = mod(div(i, 153), 12) + 1;
    const gy = div(j, 1461) - 100100 + div(8 - gm, 6);
    return { gy, gm, gd };
}

export const isLeapYear = (jy) => cal(jy).leap === 0;

export const monthLength = (jy, jm) => (jm <= 6 ? 31 : jm <= 11 ? 30 : isLeapYear(jy) ? 30 : 29);

export const isValid = (jy, jm, jd) =>
    Number.isInteger(jy) && Number.isInteger(jm) && Number.isInteger(jd) &&
    jy >= BREAKS[0] && jy < BREAKS[BREAKS.length - 1] && jm >= 1 && jm <= 12 && jd >= 1 && jd <= monthLength(jy, jm);

/** Julian Day Number of a Jalali date. */
export function toDayNumber(jy, jm, jd) {
    const r = cal(jy);
    return g2d(r.gy, 3, r.march) + (jm - 1) * 31 - div(jm, 7) * (jm - 7) + jd - 1;
}

/** Jalali day of a Julian Day Number. */
export function fromDayNumber(n) {
    const gy = d2g(n).gy;
    let jy = gy - 621;
    const r = cal(jy);
    let k = n - g2d(gy, 3, r.march);
    if (k >= 0) {
        if (k <= 185) return { jy, jm: 1 + div(k, 31), jd: mod(k, 31) + 1, n };
        k -= 186;
    } else {
        jy -= 1;
        k += 179;
        if (r.leap === 1) k += 1;
    }
    return { jy, jm: 7 + div(k, 30), jd: mod(k, 30) + 1, n };
}

export const day = (jy, jm, jd) => ({ jy, jm, jd, n: toDayNumber(jy, jm, jd) });

export const addDays = (d, days) => fromDayNumber(d.n + days);

export const toGregorian = (jy, jm, jd) => d2g(toDayNumber(jy, jm, jd));

export const fromGregorian = (gy, gm, gd) => fromDayNumber(g2d(gy, gm, gd));

/** Today in the visitor's local calendar (the PHP side uses Asia/Tehran). */
export function today(now = new Date()) {
    return fromGregorian(now.getFullYear(), now.getMonth() + 1, now.getDate());
}

export const iso = (d) => `${String(d.jy).padStart(4, '0')}-${String(d.jm).padStart(2, '0')}-${String(d.jd).padStart(2, '0')}`;

/** «۱۷ بهمن ۱۴۰۵» or, without the year, «۱۷ بهمن». */
export const formatDay = (d, withYear = true) => toPersian(`${d.jd} ${MONTHS[d.jm - 1]}${withYear ? ` ${d.jy}` : ''}`);

/* ---------- input ---------- */

const normalise = (text) =>
    toLatin(text).replace(/ي/g, 'ی').replace(/ك/g, 'ک').replace(/‌/g, '').replace(/\s+/gu, ' ').trim();

function validDay(jy, jm, jd) {
    return jy >= MIN_YEAR && jy <= MAX_YEAR && isValid(jy, jm, jd) ? day(jy, jm, jd) : null;
}

/** «۱۴۰۵/۰۲/۱۲», «1405-2-12», «1405.2.12», «۱۲ اردیبهشت ۱۴۰۵» → day, else null. */
export function parseDate(text) {
    const s = normalise(text);
    let m = s.match(/^(\d{4})[/.-](\d{1,2})[/.-](\d{1,2})$/);
    if (m) return validDay(+m[1], +m[2], +m[3]);
    m = s.match(/^(\d{1,2}) (\S+) (\d{4})$/u);
    if (m) {
        const month = MONTHS.indexOf(m[2]);
        return month >= 0 ? validDay(+m[3], month + 1, +m[1]) : null;
    }
    return null;
}

/** Digits of the text («۲۸ روز» → 28); empty → 28; outside 21–45 → null. */
export function parseCycle(text) {
    const digits = toLatin(text).replace(/\D/g, '');
    if (digits === '') return DEFAULT_CYCLE;
    const days = digits.length > 3 ? 0 : parseInt(digits, 10);
    return days >= MIN_CYCLE && days <= MAX_CYCLE ? days : null;
}

/* ---------- calculators ---------- */

/** Naegele + cycle adjustment. `now` (a day) null → no «how far along». */
export function dueDate(lmp, cycle, now = null) {
    const due = addDays(lmp, PREGNANCY_DAYS + cycle - DEFAULT_CYCLE);
    const elapsed = now ? now.n - lmp.n : -1;
    const show = elapsed >= 0 && elapsed <= PROGRESS_MAX_DAYS;
    return {
        due,
        rangeFrom: addDays(due, -RANGE_DAYS),
        rangeTo: addDays(due, RANGE_DAYS),
        weeks: show ? div(elapsed, 7) : null,
        days: show ? elapsed % 7 : null,
    };
}

export function fertilityWindow(lmp, cycle) {
    const ovulation = addDays(lmp, cycle - LUTEAL_DAYS);
    return { windowFrom: addDays(ovulation, -WINDOW_LEAD_DAYS), windowTo: ovulation, ovulation, nextPeriod: addDays(lmp, cycle) };
}

/* ---------- result text (templates = lang/fa/tools.php `text`) ---------- */

const fill = (template, values) =>
    String(template ?? '').replace(/:([a-z_]+)/g, (all, name) => (name in values ? toPersian(values[name]) : all));

export function formatRange(from, to, t) {
    const sameYear = from.jy === to.jy;
    const start = sameYear && from.jm === to.jm ? toPersian(from.jd) : formatDay(from, !sameYear);
    return fill(t.range, { from: start, to: formatDay(to) });
}

export function dueText(r, t) {
    const parts = [];
    const weeks = r.weeks ?? 0;
    const days = r.days ?? 0;
    if (weeks > 0) {
        parts.push(days > 0 ? fill(t.due_progress, { weeks, days }) : fill(t.due_progress_whole, { weeks }));
    } else if (days > 0) {
        parts.push(fill(t.due_progress_days, { days }));
    }
    parts.push(fill(t.due_range, { range: formatRange(r.rangeFrom, r.rangeTo, t) }));
    return { value: formatDay(r.due), detail: parts.join(t.separator) };
}

export function fertilityText(r, t) {
    return {
        value: formatRange(r.windowFrom, r.windowTo, t),
        detail: fill(t.fert_detail, { ovulation: formatDay(r.ovulation, false), next: formatDay(r.nextPeriod) }),
    };
}

/**
 * One calculation, same contract as ToolsCalculators::evaluate(): { error } (invalid_date | future_date |
 * cycle_range) or { value, detail }. `now` null skips the future-date check and the «how far along» line.
 */
export function evaluate(kind, lmpText, cycleText, t, now = null) {
    const lmp = parseDate(lmpText);
    const cycle = parseCycle(cycleText);
    if (!lmp) return { error: 'invalid_date' };
    if (now && lmp.n > now.n) return { error: 'future_date' };
    if (cycle === null) return { error: 'cycle_range' };
    return kind === 'due' ? dueText(dueDate(lmp, cycle, now), t) : fertilityText(fertilityWindow(lmp, cycle), t);
}
