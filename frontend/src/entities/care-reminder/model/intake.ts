import type { CareToday } from './types';

export interface IntakeChange {
  reminderId: number;
  slot: string;
  taken: boolean;
}

/**
 * The day after ticking/unticking one dose — the optimistic cache update for
 * `careKeys.today` (features/log-intake). Pure and immutable; an unknown
 * dose or a no-op change returns the same object.
 */
export function applyIntake(today: CareToday, change: IntakeChange): CareToday {
  let changed = false;
  const doses = today.doses.map((dose) => {
    if (dose.reminderId !== change.reminderId || dose.slot !== change.slot) return dose;
    if (dose.taken === change.taken) return dose;
    changed = true;
    return { ...dose, taken: change.taken };
  });
  if (!changed) return today;
  return {
    ...today,
    doses,
    takenCount: Math.max(0, Math.min(today.total, today.takenCount + (change.taken ? 1 : -1))),
  };
}
