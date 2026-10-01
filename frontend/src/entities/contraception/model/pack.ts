import type {
  ContraceptionMethod,
  ContraceptionMethodCode,
  MethodPayload,
  PackDay,
} from './types';

/** Pill methods get the pack screen; every other method the long-acting reminders. */
export function isPillMethod(method: ContraceptionMethodCode | null | undefined): boolean {
  return method === 'combined_pill' || method === 'progestin_pill';
}

/** Long-acting methods whose reminders live on `/contraception/other` (CB-CONTRA-03). */
export function hasMethodReminders(method: ContraceptionMethodCode | null | undefined): boolean {
  return method === 'copper_iud' || method === 'hormonal_iud' || method === 'injection' || method === 'implant';
}

/** Typical IUD lifetimes as the stepper's starting value (the user corrects it to her device). */
export const IUD_DEFAULT_YEARS: Record<'copper_iud' | 'hormonal_iud', number> = {
  copper_iud: 10,
  hormonal_iud: 5,
};
export const IUD_YEARS_RANGE = { min: 1, max: 12 } as const;
export const PACKS_LEFT_RANGE = { min: 0, max: 24 } as const;

/** How one circle of the pack grid looks. */
export type PackCellState = 'taken' | 'missed' | 'today' | 'todayTaken' | 'upcoming' | 'untracked' | 'placebo' | 'break';

export function packCellState(day: PackDay, todayDate: string): PackCellState {
  if (day.kind === 'break') return 'break';
  const isToday = day.date === todayDate;
  if (isToday) return day.status === 'taken' ? 'todayTaken' : 'today';
  if (day.status === 'taken') return 'taken';
  if (day.status === 'missed') return 'missed';
  if (day.kind === 'placebo') return 'placebo';
  if (day.status === 'untracked') return 'untracked';
  return 'upcoming';
}

/** The 28 days as rows of a week (the board's 4 × 7 grid), in pack order. */
export function packWeeks(days: readonly PackDay[]): PackDay[][] {
  const sorted = [...days].sort((a, b) => a.day - b.day);
  const weeks: PackDay[][] = [];
  for (let i = 0; i < sorted.length; i += 7) weeks.push(sorted.slice(i, i + 7));
  return weeks;
}

/**
 * The saved method as a `PUT /contraception/method` body — a full replace, so
 * a small edit (packs left) re-sends everything else unchanged.
 */
export function methodToPayload(method: ContraceptionMethod): MethodPayload {
  const base: MethodPayload = { method: method.method };
  if (isPillMethod(method.method)) {
    return {
      ...base,
      pack_type: method.method === 'combined_pill' ? method.packType : null,
      pack_started_on: method.packStartedOn,
      packs_left: method.packsLeft,
      ...(method.reminder ? { reminder_time: method.reminder.time, reminder_enabled: method.reminder.enabled } : {}),
    };
  }
  switch (method.method) {
    case 'copper_iud':
    case 'hormonal_iud':
      return {
        ...base,
        inserted_on: method.insertedOn,
        iud_lifetime_years: method.iudLifetimeYears,
        followup_done: method.followupDone,
      };
    case 'injection':
      return { ...base, injected_on: method.injectedOn };
    case 'implant':
      return { ...base, inserted_on: method.insertedOn, replace_on: method.replaceOn };
    default:
      return base;
  }
}
