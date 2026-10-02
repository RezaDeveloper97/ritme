import type { LogDayChanges, LogDayValues } from '@/entities/health-log';

/*
 * «حالت امروز چطوره؟» chips of the postpartum home (nbl_v15_Main). They are a
 * shortcut into the same taxonomy v2 rows the postpartum log sheet writes
 * (B-N3-06), so the sheet and the chips always agree:
 *   خوب → mood.moods happy (calm also counts) · خسته → appetite_energy.energy low
 *   بی‌قرار → mood.moods anxious · غمگین → mood.moods sad
 */

export const MOOD_CHIPS = ['good', 'tired', 'restless', 'sad'] as const;
export type MoodChip = (typeof MOOD_CHIPS)[number];

export type DayValues = LogDayValues;
export type DayChanges = LogDayChanges;

const MOOD_CODES: Record<Exclude<MoodChip, 'tired'>, readonly string[]> = {
  good: ['happy', 'calm'],
  restless: ['anxious'],
  sad: ['sad'],
};
const TIRED_LEVELS = ['low', 'very_low'];

function moods(day: DayValues): string[] {
  const v = day.mood?.moods;
  return Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : [];
}

/** Which chips today's log already holds. */
export function pressedChips(day: DayValues | undefined): Set<MoodChip> {
  const out = new Set<MoodChip>();
  if (!day) return out;
  const list = moods(day);
  for (const chip of ['good', 'restless', 'sad'] as const) {
    if (MOOD_CODES[chip].some((c) => list.includes(c))) out.add(chip);
  }
  const energy = day.appetite_energy?.energy;
  if (typeof energy === 'string' && TIRED_LEVELS.includes(energy)) out.add('tired');
  return out;
}

/** One PUT /logs/days change + the optimistic draft for toggling `chip`. */
export function toggleChip(day: DayValues | undefined, chip: MoodChip): { changes: DayChanges; draft: DayValues } {
  const base: DayValues = day ?? {};
  const on = pressedChips(base).has(chip);
  if (chip === 'tired') {
    const energy = on ? null : 'low';
    const draft: DayValues = { ...base, appetite_energy: { ...base.appetite_energy } };
    if (energy === null) delete draft.appetite_energy.energy;
    else draft.appetite_energy.energy = energy;
    if (Object.keys(draft.appetite_energy).length === 0) delete draft.appetite_energy;
    return { changes: { appetite_energy: { energy } }, draft };
  }
  const current = moods(base);
  const next = on ? current.filter((c) => !MOOD_CODES[chip].includes(c)) : [...current, MOOD_CODES[chip][0]];
  const draft: DayValues = { ...base, mood: { ...base.mood } };
  if (next.length) draft.mood.moods = next;
  else delete draft.mood.moods;
  if (Object.keys(draft.mood).length === 0) delete draft.mood;
  return { changes: { mood: { moods: next.length ? next : null } }, draft };
}
