import type { FertilityInsights } from "@/entities/fertility";
import type { Locale } from "@/shared/i18n";
import { fromApiDate, shiftMonth, toParts } from "@/shared/lib/date";

/** Fewer than two logged cycles → the low-data state (Scope §5). */
export const MIN_CYCLES = 2;

export function isLowData(data: FertilityInsights): boolean {
  return data.cyclesUsed < MIN_CYCLES || data.window === null;
}

export type DayMark = "ovulation" | "window" | null;

/** How a `Y-m-d` day sits relative to the estimated window. */
export function markFor(
  day: string,
  window: FertilityInsights["window"],
): DayMark {
  if (!window) return null;
  if (window.ovulation === day) return "ovulation";
  if (day >= window.start && day <= window.end) return "window";
  return null;
}

/**
 * Every (year, month) of the locale's calendar that the window touches, in
 * order — a window that crosses a month boundary (27 مهر → 2 آبان) yields both
 * months so the calendar can mark every window day. `null` → no months.
 */
export function windowMonths(
  window: FertilityInsights["window"],
  locale: Locale,
): { year: number; month: number }[] {
  if (!window) return [];
  const first = toParts(fromApiDate(window.start), locale);
  const last = toParts(fromApiDate(window.end), locale);
  const months: { year: number; month: number }[] = [];
  let cur = { year: first.year, month: first.month };
  // Bounded loop: a window is days long, never more than a handful of months.
  for (let i = 0; i < 12; i++) {
    months.push(cur);
    if (cur.year > last.year || (cur.year === last.year && cur.month >= last.month)) break;
    cur = shiftMonth(cur.year, cur.month, 1);
  }
  return months;
}

/** Evidence row icon, picked by the server key. */
export function evidenceIcon(
  key: string,
): "chart" | "thermo" | "flaskLh" | "drop" | "calendar" | "info" {
  switch (key) {
    case "cycles":
      return "calendar";
    case "bbt_shift":
    case "bbt":
      return "thermo";
    case "lh":
      return "flaskLh";
    case "mucus":
      return "drop";
    case "history":
      return "chart";
    default:
      return "info";
  }
}
