import { z } from 'zod';

/**
 * The timed cycle reminders (B-N1-09), shown inside the «چرخه» group of
 * `/profile/notifications` since the two screens were merged. They are read
 * and saved through GET/PUT /profile/cycle-settings (the server stores them as
 * notification preferences, so the on/off switch is the same row the
 * notification settings return); this slice only edits their schedule
 * (time, days before) and the pill reminder, which has no category row.
 */

export type ReminderCode = 'before_period' | 'pms' | 'fertile_window' | 'daily_log' | 'pill';

const REMINDER_CODES: readonly ReminderCode[] = ['before_period', 'pms', 'fertile_window', 'daily_log', 'pill'];

/** Reminders with a clock the user can change (pms / fertile window follow the cycle day). */
export const TIMED_REMINDERS: readonly ReminderCode[] = ['before_period', 'daily_log', 'pill'];

export const DAYS_BEFORE_RANGE = { min: 1, max: 7 } as const;

export interface Reminder {
  code: ReminderCode;
  enabled: boolean;
  /** `HH:MM`, the user's wall-clock. */
  time: string;
  /** before_period only. */
  daysBefore?: number;
  /** pms only: the cycle day it fires on (0 / missing = unknown). */
  cycleDay?: number;
}

export interface ReminderPatch {
  enabled?: boolean;
  time?: string;
  days_before?: number;
}

/** The `reminders` part of a PUT /profile/cycle-settings body; omitted keys keep their value. */
export type RemindersPatch = Partial<Record<ReminderCode, ReminderPatch>>;

const isReminder = (code: string): code is ReminderCode => (REMINDER_CODES as readonly string[]).includes(code);

/** Boundary parser (CLAUDE.md §10): only the reminders of the cycle-settings payload; unknown codes dropped. */
export const remindersSchema = z
  .object({
    reminders: z
      .array(
        z.object({
          code: z.string(),
          enabled: z.boolean(),
          time: z.string(),
          days_before: z.number().int().nullable().optional(),
          cycle_day: z.number().int().nullable().optional(),
        }),
      )
      .default([]),
  })
  .transform((raw): Reminder[] =>
    raw.reminders.flatMap((r) =>
      isReminder(r.code)
        ? [
            {
              code: r.code,
              enabled: r.enabled,
              time: r.time,
              daysBefore: r.days_before ?? undefined,
              cycleDay: r.cycle_day ?? undefined,
            },
          ]
        : [],
    ),
  );

/** The reminders with `patch` applied — the optimistic cache value while a PUT is in flight. */
export function applyReminderPatch(list: Reminder[], patch: RemindersPatch): Reminder[] {
  return list.map((r) => {
    const p = patch[r.code];
    if (!p) return r;
    return { ...r, enabled: p.enabled ?? r.enabled, time: p.time ?? r.time, daysBefore: p.days_before ?? r.daysBefore };
  });
}

/** Clamp into a stepper range. */
export function clamp(n: number, range: { min: number; max: number }): number {
  return Math.min(range.max, Math.max(range.min, n));
}
