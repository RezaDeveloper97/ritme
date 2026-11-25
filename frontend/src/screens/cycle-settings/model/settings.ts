import { z } from 'zod';

/**
 * Cycle settings (B-N1-09) — GET/PUT /profile/cycle-settings.
 * «چرخهٔ تو»: the lengths shown (automatic = the engine's medians; manual = the
 * profile values). The same payload carries the cycle reminders; since the
 * merge they are edited on `/profile/notifications`, so this screen ignores them.
 */

/** Manual length bounds (the server accepts 15–60 / 1–15; the steppers keep to realistic cycles). */
export const CYCLE_RANGE = { min: 20, max: 45 } as const;
export const PERIOD_RANGE = { min: 2, max: 10 } as const;

export interface CycleLengths {
  auto: boolean;
  cycleLength: number;
  periodLength: number;
  calculated: { cycleLength: number | null; periodLength: number | null; basedOnCycles: number | null };
  manual: { cycleLength: number | null; periodLength: number | null };
}

export interface CycleSettings {
  lengths: CycleLengths;
}

/** A partial PUT body — omitted keys keep their stored value. */
export interface CycleSettingsPatch {
  lengths_auto?: boolean;
  cycle_length?: number;
  period_length?: number;
}

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
          .prefault({}),
        manual: z.object({ cycle_length: nullableInt, period_length: nullableInt }).prefault({}),
      })
      .prefault({}),
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
  return {
    lengths: {
      ...s.lengths,
      auto,
      cycleLength,
      periodLength,
      manual: { cycleLength: manualCycle, periodLength: manualPeriod },
    },
  };
}

/** Clamp into a stepper range. */
export function clamp(n: number, range: { min: number; max: number }): number {
  return Math.min(range.max, Math.max(range.min, n));
}
