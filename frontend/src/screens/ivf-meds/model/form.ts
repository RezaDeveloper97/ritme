import {
  IVF_MAX_TIMES,
  type IvfMed,
  type IvfMedInput,
  type IvfMedPreset,
  type IvfRole,
  type IvfRoute,
  type IvfStockUnit,
} from '@/entities/ivf';

/*
 * The add / edit medicine form («افزودن دارو از روی نسخه», CB-IVF-03): a
 * controlled draft, its problems and the API body. Pure — no React, no locale.
 */

/** Dose units the form offers as chips (a saved unknown unit is kept as an extra chip). */
export const DOSE_UNITS = ['iu', 'mg', 'mcg', 'ml'] as const;

export const STOCK_RANGE = { min: 0, max: 999 } as const;
export const DOSES_PER_UNIT_RANGE = { min: 1, max: 100 } as const;

export interface MedDraft {
  presetCode: string | null;
  name: string;
  role: IvfRole;
  route: IvfRoute;
  /** ASCII digits / dot; empty = no amount. */
  dose: string;
  unit: string | null;
  /** Daily `HH:MM` (not for the trigger). */
  times: string[];
  /** The trigger's `Y-m-d` + `HH:MM`. */
  triggerDate: string | null;
  triggerTime: string;
  startsOn: string | null;
  endsOn: string | null;
  notes: string;
  trackStock: boolean;
  stockUnits: number;
  stockUnit: IvfStockUnit;
  dosesPerUnit: number;
}

export type DraftField = 'name' | 'times' | 'trigger' | 'endsOn' | 'dose';
export type DraftProblems = Partial<Record<DraftField, 'missing' | 'order' | 'format'>>;

export function emptyDraft(todayIso: string): MedDraft {
  return {
    presetCode: null,
    name: '',
    role: 'stimulation',
    route: 'subcutaneous',
    dose: '',
    unit: 'iu',
    times: ['20:00'],
    triggerDate: null,
    triggerTime: '22:00',
    startsOn: todayIso,
    endsOn: null,
    notes: '',
    trackStock: false,
    stockUnits: 1,
    stockUnit: 'pen',
    dosesPerUnit: 1,
  };
}

/**
 * Fill the draft from a prescription preset: the class name (unless she typed
 * her own), role, route, unit, times and stock unit. Amounts stay hers.
 */
export function applyPreset(draft: MedDraft, preset: IvfMedPreset, title: string, previousTitle: string | null): MedDraft {
  const typed = draft.name.trim();
  const keepName = typed !== '' && typed !== previousTitle;
  return {
    ...draft,
    presetCode: preset.code,
    name: keepName ? draft.name : title,
    role: preset.role,
    route: preset.route,
    unit: preset.unit ?? draft.unit,
    times: preset.role === 'trigger' ? draft.times : preset.times.length ? preset.times.slice(0, IVF_MAX_TIMES) : draft.times,
    stockUnit: preset.stockUnit ?? draft.stockUnit,
  };
}

/** An existing medicine → the edit draft (the stock as left today, so an untouched save keeps the count). */
export function draftFromMed(med: IvfMed, todayIso: string): MedDraft {
  const base = emptyDraft(todayIso);
  const trigger = med.triggerAt ? med.triggerAt.trim() : null;
  return {
    ...base,
    name: med.name,
    role: med.role,
    route: med.route,
    dose: med.dose ?? '',
    unit: med.unit,
    times: med.times.length ? med.times : base.times,
    triggerDate: trigger ? trigger.slice(0, 10) : null,
    triggerTime: trigger ? (/\d{2}:\d{2}/.exec(trigger.slice(10))?.[0] ?? base.triggerTime) : base.triggerTime,
    startsOn: med.startsOn,
    endsOn: med.endsOn,
    notes: med.notes ?? '',
    trackStock: med.inventory !== null,
    stockUnits: med.inventory?.unitsLeft ?? base.stockUnits,
    stockUnit: med.inventory?.stockUnit ?? base.stockUnit,
    dosesPerUnit: med.inventory?.dosesPerUnit ?? base.dosesPerUnit,
  };
}

export function isTriggerDraft(draft: Pick<MedDraft, 'role'>): boolean {
  return draft.role === 'trigger';
}

/** Add a dose time (one hour after the last, wrapping), at most {@link IVF_MAX_TIMES}. */
export function addTime(times: readonly string[]): string[] {
  if (times.length >= IVF_MAX_TIMES) return [...times];
  const last = times[times.length - 1];
  const hour = last ? (Number(last.slice(0, 2)) + 12) % 24 : 8;
  let next = `${String(hour).padStart(2, '0')}:00`;
  while (times.includes(next)) next = `${String((Number(next.slice(0, 2)) + 1) % 24).padStart(2, '0')}:00`;
  return sortTimes([...times, next]);
}

export function sortTimes(times: readonly string[]): string[] {
  return [...new Set(times)].sort();
}

/** `HH:MM` → wheel indexes, and back. */
export function parseClock(value: string): { hour: number; minute: number } {
  const m = /^(\d{2}):(\d{2})$/.exec(value);
  return m ? { hour: Math.min(23, Number(m[1])), minute: Math.min(59, Number(m[2])) } : { hour: 8, minute: 0 };
}

export function toClock(hour: number, minute: number): string {
  return `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`;
}

export function draftProblems(draft: MedDraft): DraftProblems {
  const problems: DraftProblems = {};
  if (!draft.name.trim()) problems.name = 'missing';
  if (draft.dose.trim() && !/^\d+(\.\d+)?$/.test(draft.dose.trim())) problems.dose = 'format';
  if (isTriggerDraft(draft)) {
    if (!draft.triggerDate) problems.trigger = 'missing';
  } else if (draft.times.length === 0) {
    problems.times = 'missing';
  }
  if (draft.startsOn && draft.endsOn && draft.endsOn < draft.startsOn) problems.endsOn = 'order';
  return problems;
}

/** The draft → `POST|PUT /ivf/meds` input. */
export function draftToInput(draft: MedDraft): IvfMedInput {
  const trigger = isTriggerDraft(draft);
  const dose = draft.dose.trim();
  const notes = draft.notes.trim();
  return {
    name: draft.name.trim(),
    role: draft.role,
    route: draft.route,
    dose: dose || null,
    unit: dose ? draft.unit : null,
    times: trigger ? null : sortTimes(draft.times),
    triggerAt: trigger && draft.triggerDate ? `${draft.triggerDate} ${draft.triggerTime}` : null,
    startsOn: draft.startsOn,
    endsOn: draft.endsOn,
    notes: notes || null,
    stockUnits: draft.trackStock ? draft.stockUnits : null,
    stockUnit: draft.trackStock ? draft.stockUnit : null,
    dosesPerUnit: draft.trackStock ? draft.dosesPerUnit : null,
  };
}
