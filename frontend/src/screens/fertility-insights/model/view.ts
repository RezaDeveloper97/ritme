import { PMS_WINDOW_DAYS } from "@/entities/cycle";
import type {
  EvidenceStrength,
  FertilityInsights,
  InsightConfidence,
  OvulationHistoryRow,
} from "@/entities/fertility";
import type { Locale } from "@/shared/i18n";
import {
  addDays,
  diffInDays,
  fromApiDate,
  shiftMonth,
  toParts,
  weekOf,
} from "@/shared/lib/date";

/** Fewer than two logged cycles → the low-data state (Scope §5). */
export const MIN_CYCLES = 2;

export function isLowData(data: FertilityInsights): boolean {
  return data.cyclesUsed < MIN_CYCLES || data.window === null;
}

export type DayMark = "ovulation" | "window" | "today" | null;

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

/**
 * The calendar's mark for one day: today wins (a filled ink disc, as in the
 * artboard), then ovulation, then the window.
 */
export function calendarMark(
  day: string,
  window: FertilityInsights["window"],
  today: string,
): DayMark {
  if (day === today) return "today";
  return markFor(day, window);
}

/** Days of padding either side of the window (audit #19: ± 1 week). */
const CALENDAR_PAD_DAYS = 7;

/**
 * The calendar rows: every locale-week from one week before the window to one
 * week after it, so a window that crosses a month boundary is one continuous
 * block instead of two full months (audit #19). `null` → no rows.
 */
export function windowWeeks(
  window: FertilityInsights["window"],
  locale: Locale,
): Date[][] {
  if (!window) return [];
  const last = addDays(fromApiDate(window.end), CALENDAR_PAD_DAYS);
  const weeks: Date[][] = [];
  let week = weekOf(addDays(fromApiDate(window.start), -CALENDAR_PAD_DAYS), locale);
  // Bounded: a window is days long, so this is a handful of weeks.
  for (let i = 0; i < 10 && diffInDays(week[0], last) <= 0; i++) {
    weeks.push(week);
    week = weekOf(addDays(week[0], 7), locale);
  }
  return weeks;
}

/** Evidence row tone (audit #15): cycles green, BBT teal, LH amber. */
export function evidenceTone(key: string): "green" | "teal" | "amber" {
  switch (key) {
    case "cycles":
      return "green";
    case "lh":
      return "amber";
    default:
      return "teal";
  }
}

/** Strength pill tone (audit #15): قوی green, متوسط turquoise, ندارد amber. */
export const STRENGTH_TONE: Record<EvidenceStrength, "green" | "teal" | "amber"> = {
  strong: "green",
  medium: "teal",
  none: "amber",
};

/**
 * Confidence pill tone (audit #16): the level is the algorithm's output, so
 * medium/high read as data (turquoise); low stays neutral.
 */
export const CONFIDENCE_TONE: Record<InsightConfidence, "muted" | "teal"> = {
  low: "muted",
  medium: "teal",
  high: "teal",
};

export type StripDay = "period" | "fertile" | "ovulation" | "pms" | "none";

/** Longest strip drawn — anything longer is not a cycle we can trust to draw. */
const MAX_STRIP_DAYS = 60;
/** Fertile days before ovulation (matches the backend window, ovulation − 5). */
const FERTILE_DAYS_BEFORE = 5;

/**
 * One finished cycle as a day strip (`v19_TTC_Insights` «تخمک‌گذاری در
 * سیکل‌های قبل»): period days red, the five days before ovulation amber,
 * ovulation turquoise, the last {@link PMS_WINDOW_DAYS} days violet.
 *
 * The cycle's length is the gap to the next cycle's start (`nextStart`: the
 * newer history row, or the current cycle's start for the newest row). The API
 * sends no per-cycle period length, so `periodLength` is the engine's
 * effective one. `null` when the row can't be placed.
 */
export function historyStrip(
  row: OvulationHistoryRow,
  nextStart: string | null,
  periodLength: number,
): StripDay[] | null {
  if (!row.cycleStart || !nextStart) return null;
  const length = diffInDays(fromApiDate(nextStart), fromApiDate(row.cycleStart));
  if (length < 1 || length > MAX_STRIP_DAYS) return null;
  const ov = row.ovulationDay;
  return Array.from({ length }, (_, i): StripDay => {
    const day = i + 1;
    if (day <= periodLength) return "period";
    if (ov !== null && day === ov) return "ovulation";
    if (ov !== null && day >= ov - FERTILE_DAYS_BEFORE && day < ov) return "fertile";
    if (day > length - PMS_WINDOW_DAYS) return "pms";
    return "none";
  });
}

/** `nextStart` for each history row (newest first): the row above it, else the current cycle. */
export function nextStarts(
  history: readonly OvulationHistoryRow[],
  currentStart: string | null,
): (string | null)[] {
  return history.map((_, i) => (i === 0 ? currentStart : history[i - 1].cycleStart));
}

/** Evidence row icon, picked by the server key. */
export function evidenceIcon(
  key: string,
): "chart" | "thermo" | "flaskLh" | "drop" | "check" | "info" {
  switch (key) {
    case "cycles":
      return "check";
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
