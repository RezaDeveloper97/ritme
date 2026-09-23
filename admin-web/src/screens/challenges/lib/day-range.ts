/** Which label describes a challenge's cycle-day window (Challenge::cycleDayLabel). */
export type DayRange =
  | { kind: 'all' }
  | { kind: 'single'; day: number }
  | { kind: 'range'; from: number; to: number }
  | { kind: 'from'; from: number }
  | { kind: 'to'; to: number };

export function dayRange(from: number | null, to: number | null): DayRange {
  if (from !== null && to !== null) return from === to ? { kind: 'single', day: from } : { kind: 'range', from, to };
  if (from !== null) return { kind: 'from', from };
  if (to !== null) return { kind: 'to', to };
  return { kind: 'all' };
}
