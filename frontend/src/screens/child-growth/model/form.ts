import type { ChildMeasurement } from '@/entities/child';

import type { MeasurementForm } from './types';

/** Server limits (backend-go `internal/children/service.go`). */
export const MEASUREMENT_RANGES = {
  weightKg: { min: 0.3, max: 60, decimals: 3 },
  lengthCm: { min: 20, max: 150, decimals: 1 },
  headCm: { min: 15, max: 70, decimals: 1 },
} as const;
export type MeasurementField = keyof typeof MEASUREMENT_RANGES;

const text = (n: number | undefined | null): string => (n == null ? '' : String(n));

export function emptyForm(today: string): MeasurementForm {
  return { measuredOn: today, weightKg: '', lengthCm: '', headCm: '' };
}

export function formFromMeasurement(m: ChildMeasurement): MeasurementForm {
  return {
    measuredOn: m.measuredOn,
    weightKg: text(m.weight?.value),
    lengthCm: text(m.length?.value),
    headCm: text(m.head?.value),
  };
}

const value = (s: string): number | null => {
  if (s.trim() === '') return null;
  const n = Number(s);
  return Number.isFinite(n) ? n : null;
};

export type FormProblem = { field: MeasurementField; kind: 'range' } | { field: 'all'; kind: 'empty' };

/** Client-side check before sending (the server re-validates and its 422 wins). */
export function checkForm(form: MeasurementForm): FormProblem | null {
  for (const field of Object.keys(MEASUREMENT_RANGES) as MeasurementField[]) {
    const v = value(form[field]);
    const r = MEASUREMENT_RANGES[field];
    if (v !== null && (v < r.min || v > r.max)) return { field, kind: 'range' };
  }
  if (value(form.weightKg) === null && value(form.lengthCm) === null && value(form.headCm) === null) {
    return { field: 'all', kind: 'empty' };
  }
  return null;
}

/** Snake-case body; every key is sent so an emptied field clears on an update. */
export function toMeasurementBody(form: MeasurementForm): Record<string, unknown> {
  return {
    measured_on: form.measuredOn,
    weight_kg: value(form.weightKg),
    length_cm: value(form.lengthCm),
    head_cm: value(form.headCm),
  };
}
