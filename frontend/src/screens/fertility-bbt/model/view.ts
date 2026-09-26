import {
  BBT_RANGES,
  type BbtCycle,
  type BbtPoint,
  type BbtRange,
} from "@/entities/fertility";
import type { Reminder } from "@/entities/reminder";

/** Fewer current-cycle readings than this → empty state instead of a chart. */
export const MIN_READINGS = 3;
/** «یادآوری ثبت دمای فردا» fires daily at this local time. */
export const REMINDER_TIME = "07:00";

/** `?range=` → a supported range; anything else is the current cycle. */
export function parseRange(raw: string | undefined): BbtRange {
  const n = Number(raw);
  return (BBT_RANGES as readonly number[]).includes(n) ? (n as BbtRange) : 1;
}

/** «امروز صبح» — the current cycle's last point, only if it was logged today. */
export function todayReading(
  cycle: BbtCycle | undefined,
  today: string,
): BbtPoint | null {
  if (!cycle || cycle.points.length === 0) return null;
  const last = cycle.points.reduce((a, b) => (b.cycleDay > a.cycleDay ? b : a));
  return last.date === today ? last : null;
}

export function hasEnoughReadings(cycle: BbtCycle | undefined): boolean {
  return (cycle?.points.length ?? 0) >= MIN_READINGS;
}

/** The daily 07:00 BBT reminder this screen created earlier, matched by its title. */
export function findBbtReminder(
  reminders: readonly Reminder[] | undefined,
  title: string,
): Reminder | null {
  return (
    reminders?.find(
      (r) =>
        r.type === "custom" &&
        r.recurrence === "daily" &&
        r.recurrenceTime === REMINDER_TIME &&
        r.title === title &&
        r.isActive,
    ) ?? null
  );
}
