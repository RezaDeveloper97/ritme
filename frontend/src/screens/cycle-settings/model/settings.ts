import { z } from 'zod';

/**
 * Cycle settings (B-N1-09) — GET/PUT /profile/cycle-settings.
 * «سیکل تو»: the lengths shown (automatic = the engine's medians; manual = the
 * profile values). «یادآورها»: the timed cycle reminders, stored by the server
 * as notification preferences (the same switches as /profile/notifications,
 * plus the pill, which is set only here).
 */

export type ReminderCode = 'before_period' | 'pms' | 'fertile_window' | 'daily_log' | 'pill';

/** Screen order; unknown codes from a newer server are hidden. */
export const REMINDER_CODES: readonly ReminderCode[] = ['before_period', 'pms', 'fertile_window', 'daily_log', 'pill'];

/** Manual length bounds (the server accepts 15–60 / 1–15; the steppers keep to realistic cycles). */
export const CYCLE_RANGE = { min: 20, max: 45 } as const;
export const PERIOD_RANGE = { min: 2, max: 10 } as const;
export const DAYS_BEFORE_RANGE = { min: 1, max: 7 } as const;

export interface Reminder {
  code: ReminderCode;
  enabled: boolean;
  /** `HH:MM`, Tehran wall-clock. */
  time: string;
  /** before_period only. */
  daysBefore?: number;
  /** pms only: the cycle day it fires on (0 = unknown). */
  cycleDay?: number;
}

export interface CycleLengths {
  auto: boolean;
  cycleLength: number;
  periodLength: number;
  calculated: { cycleLength: number | null; periodLength: number | null; basedOnCycles: number | null };
  manual: { cycleLength: number | null; periodLength: number | null };
}

export interface CycleSettings {
  lengths: CycleLengths;
  reminders: Reminder[];
}

export interface ReminderPatch {
  enabled?: boolean;
  time?: string;
  days_before?: number;
}

/** A partial PUT body — omitted keys keep their stored value. */
export interface CycleSettingsPatch {
  lengths_auto?: boolean;
  cycle_length?: number;
  period_length?: number;
  reminders?: Partial<Record<ReminderCode, ReminderPatch>>;
}

const isReminder = (code: string): code is ReminderCode => (REMINDER_CODES as readonly string[]).includes(code);
const nullableInt = z.number().int().nullable().default(null);

/** Boundary parser (CLAUDE.md §10). */
export const cycleSettingsSchema = z
  .object({
    lengths: z
      .object({
        auto: z.boolean().default(true),
        cycle_length: z.number().int().default(28),
        period_length: z.number().int().default(5),
        calculated: z
          .object({ cycle_length: nullableInt, period_length: nullableInt, based_on_cycles: nullableInt })
          .default({}),
        manual: z.object({ cycle_length: nullableInt, period_length: nullableInt }).default({}),
      })
      .default({}),
    reminders: z
      .array(
        z.object({
          code: z.string(),
          enabled: z.boolean(),
          time: z.string(),
          days_before: z.number().int().optional(),
          cycle_day: z.number().int().optional(),
        }),
      )
      .default([]),
  })
  .transform(
    (raw): CycleSettings => ({
      lengths: {
        auto: raw.lengths.auto,
        cycleLength: raw.lengths.cycle_length,
        periodLength: raw.lengths.period_length,
        calculated: {
          cycleLength: raw.lengths.calculated.cycle_length,
          periodLength: raw.lengths.calculated.period_length,
          basedOnCycles: raw.lengths.calculated.based_on_cycles,
        },
        manual: { cycleLength: raw.lengths.manual.cycle_length, periodLength: raw.lengths.manual.period_length },
      },
      reminders: raw.reminders.flatMap((r) =>
        isReminder(r.code)
          ? [{ code: r.code, enabled: r.enabled, time: r.time, daysBefore: r.days_before, cycleDay: r.cycle_day }]
          : [],
      ),
    }),
  );

/** The settings with `patch` applied — the optimistic cache value while a PUT is in flight. */
export function applyPatch(s: CycleSettings, patch: CycleSettingsPatch): CycleSettings {
  const auto = patch.lengths_auto ?? s.lengths.auto;
  const manualCycle = patch.cycle_length ?? s.lengths.manual.cycleLength;
  const manualPeriod = patch.period_length ?? s.lengths.manual.periodLength;
  // Switching to manual keeps the numbers on screen until the user changes them.
  const cycleLength = auto ? (s.lengths.calculated.cycleLength ?? s.lengths.cycleLength) : (manualCycle ?? s.lengths.cycleLength);
  const periodLength = auto
    ? (s.lengths.calculated.periodLength ?? s.lengths.periodLength)
    : (manualPeriod ?? s.lengths.periodLength);
  const rp = patch.reminders ?? {};
  return {
    lengths: {
      ...s.lengths,
      auto,
      cycleLength,
      periodLength,
      manual: { cycleLength: manualCycle, periodLength: manualPeriod },
    },
    reminders: s.reminders.map((r) => {
      const p = rp[r.code];
      if (!p) return r;
      return {
        ...r,
        enabled: p.enabled ?? r.enabled,
        time: p.time ?? r.time,
        daysBefore: p.days_before ?? r.daysBefore,
      };
    }),
  };
}

const CLOCK = /^([01]\d|2[0-3]):([0-5]\d)$/;

/** Whether `value` is the `HH:MM` the API accepts (what `<input type="time">` yields). */
export function isClock(value: string): boolean {
  return CLOCK.test(value);
}

/** «09:00» → «9:00» as drawn on the board; digits are localized by the caller. */
export function displayClock(value: string): string {
  const m = CLOCK.exec(value);
  return m ? `${Number(m[1])}:${m[2]}` : value;
}

/** Clamp into a stepper range. */
export function clamp(n: number, range: { min: number; max: number }): number {
  return Math.min(range.max, Math.max(range.min, n));
}
