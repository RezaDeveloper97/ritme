// Jalali (Solar Hijri) ⇄ Gregorian conversion — a port of the jalaali-js
// algorithm (Borkowski's break table, exact for Jalali years -61…3177).
//
// It is the SAME algorithm the backend uses (`backend-go/internal/checkups/
// engine/jalali.go`), line for line, so a Jalali date or month boundary can
// never disagree between the two. The previous library (jalaliday 2.3.0) used a
// different leap rule and made 1404 the leap year instead of 1403, which shifted
// every date from Nowruz 1404 to 2026-03-20 by one day (T-M12-21b).
//
// Internal to `shared/lib/date`: callers use the helpers in `./calendar`.

const BREAKS = [
  -61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210,
  1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178,
] as const;

/** Integer division truncating toward zero, like Go's `/` on ints. */
const div = (a: number, b: number) => Math.trunc(a / b);

interface JalCal {
  /** Gregorian year in which Farvardin 1 of `jy` falls. */
  gy: number;
  /** Day of March on which Farvardin 1 falls. */
  march: number;
  /** Years since the last leap year (0 = `jy` itself is leap). */
  leap: number;
}

function jalCal(jy: number): JalCal {
  if (jy < BREAKS[0] || jy >= BREAKS[BREAKS.length - 1]) {
    throw new RangeError(`Jalali year ${jy} is out of the supported range`);
  }
  const gy = jy + 621;
  let leapJ = -14;
  let jp: number = BREAKS[0];
  let jump = 0;
  for (let i = 1; i < BREAKS.length; i += 1) {
    const jm = BREAKS[i];
    jump = jm - jp;
    if (jy < jm) break;
    leapJ += div(jump, 33) * 8 + div(jump % 33, 4);
    jp = jm;
  }
  let n = jy - jp;
  leapJ += div(n, 33) * 8 + div((n % 33) + 3, 4);
  if (jump % 33 === 4 && jump - n === 4) leapJ += 1;
  const leapG = div(gy, 4) - div((div(gy, 100) + 1) * 3, 4) - 150;
  const march = 20 + leapJ - leapG;
  if (jump - n < 6) n = n - jump + div(jump + 4, 33) * 33;
  let leap = (((n + 1) % 33) - 1) % 4;
  if (leap === -1) leap = 4;
  return { gy, march, leap };
}

const DAY_MS = 86_400_000;

/** Days since the epoch of a Gregorian calendar day (timezone-free). */
function dayNumber(gy: number, gm: number, gd: number): number {
  return Math.round(Date.UTC(gy, gm - 1, gd) / DAY_MS);
}

export interface YMD {
  year: number;
  /** 1-based month. */
  month: number;
  day: number;
}

/** Gregorian calendar day → Jalali parts. */
export function toJalaali(gy: number, gm: number, gd: number): YMD {
  let jy = gy - 621;
  const { gy: fy, march, leap } = jalCal(jy);
  let k = dayNumber(gy, gm, gd) - dayNumber(fy, 3, march);
  if (k >= 0) {
    if (k <= 185) return { year: jy, month: 1 + div(k, 31), day: (k % 31) + 1 };
    k -= 186;
  } else {
    // Before Farvardin 1 of `jy`: the date is in the second half of `jy − 1`.
    // `leap === 1` (one year past a leap year) means `jy − 1` was leap, so its
    // Esfand has 30 days. Same as the backend and jalaali-js.
    jy -= 1;
    k += 179;
    if (leap === 1) k += 1;
  }
  return { year: jy, month: 7 + div(k, 30), day: (k % 30) + 1 };
}

/** Jalali parts → Gregorian calendar day. */
export function toGregorian(jy: number, jm: number, jd: number): YMD {
  const { gy, march } = jalCal(jy);
  const offset = (jm - 1) * 31 - div(jm, 7) * (jm - 7) + jd - 1;
  const d = new Date(Date.UTC(gy, 2, march + offset));
  return { year: d.getUTCFullYear(), month: d.getUTCMonth() + 1, day: d.getUTCDate() };
}

/** Whether Jalali year `jy` is a leap year (Esfand has 30 days). */
export function isLeapJalaaliYear(jy: number): boolean {
  return jalCal(jy).leap === 0;
}

/** Days in a Jalali month: 31 for the first six, 30 for the next five, Esfand 29/30. */
export function jalaaliMonthLength(jy: number, jm: number): number {
  if (jm <= 6) return 31;
  if (jm <= 11) return 30;
  return isLeapJalaaliYear(jy) ? 30 : 29;
}
