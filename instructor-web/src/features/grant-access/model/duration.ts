export type DurationChoice = 'unlimited' | '30' | '90' | 'until';
export const DURATIONS: DurationChoice[] = ['unlimited', '30', '90', 'until'];

const pad = (n: number) => String(n).padStart(2, '0');
export const isoDate = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;

/** `until` allowed range: today … 5 years ahead (inclusive). */
export function untilRange(now = new Date()): { min: string; max: string } {
  const max = new Date(now);
  max.setFullYear(max.getFullYear() + 5);
  return { min: isoDate(now), max: isoDate(max) };
}

export function untilValid(value: string, now = new Date()): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  const { min, max } = untilRange(now);
  return value >= min && value <= max;
}
