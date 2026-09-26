import {
  type CervicalMucus,
  type FertilityDay,
  type FertilityDayInput,
  type FertilitySymptom,
  type IntercourseType,
  type LhResult,
  FERTILITY_SYMPTOMS,
  isBbtInRange,
  parseBbt,
  roundBbt,
} from '@/entities/fertility';

/** Sections `?focus=` can scroll to. */
export const LOG_SECTIONS = ['lh', 'bbt', 'mucus', 'intercourse', 'symptoms', 'note'] as const;
export type LogSection = (typeof LOG_SECTIONS)[number];

export function parseFocus(value: string | undefined | null): LogSection | null {
  return (LOG_SECTIONS as readonly string[]).includes(value ?? '') ? (value as LogSection) : null;
}

/** `?date=` → `Y-m-d`, falling back to today for anything malformed or in the future. */
export function parseLogDate(value: string | undefined | null, todayApi: string): string {
  if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return todayApi;
  return value > todayApi ? todayApi : value;
}

export interface LogFormState {
  lh: LhResult | null;
  mucus: CervicalMucus | null;
  /** Raw text of the BBT field ('' = not logged). */
  bbtText: string;
  intercourse: IntercourseType | null;
  symptoms: FertilitySymptom[];
  note: string;
}

export function fromDay(day: FertilityDay, formatBbtText: (v: number) => string): LogFormState {
  return {
    lh: day.lh,
    mucus: day.mucus,
    bbtText: day.bbt === null ? '' : formatBbtText(day.bbt),
    intercourse: day.intercourse,
    symptoms: FERTILITY_SYMPTOMS.filter((s) => day.symptoms.includes(s)),
    note: day.note ?? '',
  };
}

export type BbtCheck = { ok: true; value: number | null } | { ok: false; reason: 'invalid' | 'range' };

export function checkBbt(text: string): BbtCheck {
  if (text.trim() === '') return { ok: true, value: null };
  const value = parseBbt(text);
  if (value === null) return { ok: false, reason: 'invalid' };
  if (!isBbtInRange(value)) return { ok: false, reason: 'range' };
  return { ok: true, value: roundBbt(value) };
}

export function toggleSymptom(list: FertilitySymptom[], s: FertilitySymptom): FertilitySymptom[] {
  const set = new Set(list);
  if (set.has(s)) set.delete(s);
  else set.add(s);
  return FERTILITY_SYMPTOMS.filter((x) => set.has(x));
}

const sameList = (a: readonly string[], b: readonly string[]) =>
  a.length === b.length && a.every((x, i) => x === b[i]);

/**
 * Only the fields that differ from the loaded day (Scope §4). Returns null when
 * the BBT text is not a valid reading.
 */
export function changedInput(initial: FertilityDay, state: LogFormState): FertilityDayInput | null {
  const bbt = checkBbt(state.bbtText);
  if (!bbt.ok) return null;
  const input: FertilityDayInput = {};
  if (state.lh !== initial.lh) input.lh = state.lh;
  if (state.mucus !== initial.mucus) input.mucus = state.mucus;
  if (bbt.value !== (initial.bbt === null ? null : roundBbt(initial.bbt))) input.bbt = bbt.value;
  if (state.intercourse !== initial.intercourse) input.intercourse = state.intercourse;
  const initialSymptoms = FERTILITY_SYMPTOMS.filter((s) => initial.symptoms.includes(s));
  if (!sameList(state.symptoms, initialSymptoms)) input.symptoms = state.symptoms;
  const note = state.note.trim();
  if (note !== (initial.note ?? '').trim()) input.note = note === '' ? null : note;
  return input;
}

/** Dirty = anything differs from what was loaded (an invalid BBT edit counts). */
export function isDirty(initial: FertilityDay, state: LogFormState): boolean {
  const input = changedInput(initial, state);
  if (input === null) return true;
  return Object.keys(input).length > 0;
}
