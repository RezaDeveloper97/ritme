export interface YearMonth {
  year: number;
  month: number; // 1-12, Gregorian (the API's month param)
}

/** The month before/after, rolling the year over (Jan ↔ Dec). */
export function shiftMonth({ year, month }: YearMonth, by: number): YearMonth {
  const idx = year * 12 + (month - 1) + by;
  return { year: Math.floor(idx / 12), month: (idx % 12 + 12) % 12 + 1 };
}

/**
 * The months the home ring and week strip read: previous, current and next
 * month, fetched at mount without waiting for `/cycle/today` (T-M12-33). Any
 * day outside them (a late period, a long cycle) adds its own month — the
 * fallback — so correctness never depends on the guess. `dates` are API
 * `YYYY-MM-DD` strings. Result is ordered and de-duplicated.
 */
export function homeMonthSet(current: YearMonth, dates: readonly string[] = []): YearMonth[] {
  const out = new Map<number, YearMonth>();
  const add = (ym: YearMonth) => out.set(ym.year * 12 + ym.month, ym);
  add(shiftMonth(current, -1));
  add(current);
  add(shiftMonth(current, 1));
  for (const d of dates) {
    const [y, m] = d.split('-');
    const year = Number(y);
    const month = Number(m);
    if (Number.isInteger(year) && month >= 1 && month <= 12) add({ year, month });
  }
  return [...out.entries()].sort((a, b) => a[0] - b[0]).map(([, v]) => v);
}
