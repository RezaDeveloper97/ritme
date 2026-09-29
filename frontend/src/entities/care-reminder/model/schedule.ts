/** How a medication's weekdays are worded on every reminders surface. */
export type MedicationSchedule =
  | { kind: 'everyDay' }
  | { kind: 'everyOtherDay' }
  | { kind: 'daysPerWeek'; count: number };

/**
 * All seven (or none) = every day; three or more days each two apart
 * (ش/د/چ/ج, ی/س/پ) = «یک روز در میان»; anything else is a count. The one rule
 * for the /reminders screen and the legacy Profile → یادآورها sheet (whose
 * `recurrence` column can only say `daily` / `weekly`).
 */
export function medicationSchedule(weekdays: readonly number[]): MedicationSchedule {
  const days = [...new Set(weekdays)].sort((a, b) => a - b);
  if (days.length === 0 || days.length >= 7) return { kind: 'everyDay' };
  if (days.length >= 3 && days.every((d, i) => i === 0 || d - days[i - 1]! === 2)) {
    return { kind: 'everyOtherDay' };
  }
  return { kind: 'daysPerWeek', count: days.length };
}
