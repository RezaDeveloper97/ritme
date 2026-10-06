import { LIMITS, type GlucoseUnit } from '@/entities/vital';

/*
 * Client validation of the add forms, mirroring the server
 * (backend-go/internal/vitals/request.go ValidateReading) so a bad number is
 * caught before the request; the server's 422 still wins when it disagrees.
 * Pure: no React, no locale.
 */

export type FieldError =
  | { key: 'range'; min: number; max: number }
  | { key: 'systolicGtDiastolic' }
  | { key: 'future' }
  | { key: 'tooOld' }
  | { key: 'noteTooLong'; max: number };

export type FormErrors = Partial<Record<'systolic' | 'diastolic' | 'pulse' | 'value' | 'bpm' | 'measured_at' | 'note', FieldError>>;

function range(v: number | null, min: number, max: number, integer: boolean): FieldError | undefined {
  if (v === null || !Number.isFinite(v) || v < min || v > max || (integer && !Number.isInteger(v))) {
    return { key: 'range', min, max };
  }
  return undefined;
}

export function validateBp(systolic: number | null, diastolic: number | null, pulse: number | null): FormErrors {
  const errors: FormErrors = {};
  const s = range(systolic, LIMITS.systolic.min, LIMITS.systolic.max, true);
  const d = range(diastolic, LIMITS.diastolic.min, LIMITS.diastolic.max, true);
  if (s) errors.systolic = s;
  if (d) errors.diastolic = d;
  if (pulse !== null) {
    const p = range(pulse, LIMITS.pulse.min, LIMITS.pulse.max, true);
    if (p) errors.pulse = p;
  }
  if (!s && !d && systolic !== null && diastolic !== null && systolic <= diastolic) {
    errors.systolic = { key: 'systolicGtDiastolic' };
  }
  return errors;
}

/** Glucose in the unit typed: 20–600 mg/dL or 1.1–33.3 mmol/L. */
export function validateGlucose(value: number | null, unit: GlucoseUnit): FormErrors {
  const lim = unit === 'mmol_l' ? LIMITS.mmolL : LIMITS.mgDl;
  const e = range(value, lim.min, lim.max, false);
  return e ? { value: e } : {};
}

export function validateHr(bpm: number | null): FormErrors {
  const e = range(bpm, LIMITS.pulse.min, LIMITS.pulse.max, true);
  return e ? { bpm: e } : {};
}

export function validateNote(note: string): FormErrors {
  return [...note.trim()].length > LIMITS.noteMax ? { note: { key: 'noteTooLong', max: LIMITS.noteMax } } : {};
}

/** `Y-m-d HH:mm` → epoch ms on the Tehran wall clock (+03:30, Iran has no DST since 2022). */
export function tehranMs(date: string, time: string): number {
  return Date.parse(`${date}T${time}:00+03:30`);
}

/**
 * A picked measuring time: not in the future (a minute of slack, like the
 * server), not more than two years back. `null` = «now», always valid.
 */
export function validateMeasuredAt(at: { date: string; time: string } | null, nowMs: number): FormErrors {
  if (!at) return {};
  const ms = tehranMs(at.date, at.time);
  if (!Number.isFinite(ms)) return { measured_at: { key: 'future' } };
  if (ms > nowMs + 60_000) return { measured_at: { key: 'future' } };
  if (ms < nowMs - LIMITS.backfillDays * 86_400_000) return { measured_at: { key: 'tooOld' } };
  return {};
}

/** The API's `measured_at` (`Y-m-d H:i`, Tehran), or null for «now». */
export function measuredAtBody(at: { date: string; time: string } | null): string | null {
  return at ? `${at.date} ${at.time}` : null;
}

/** Today and now on the Tehran wall clock (`Y-m-d`, `HH:mm`) — the API's clock. */
export function tehranNow(now: Date = new Date()): { date: string; time: string } {
  const parts = new Intl.DateTimeFormat('en-GB', {
    timeZone: 'Asia/Tehran',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(now);
  const get = (type: string) => parts.find((p) => p.type === type)?.value ?? '00';
  return { date: `${get('year')}-${get('month')}-${get('day')}`, time: `${get('hour')}:${get('minute')}` };
}

const DIGITS: Record<string, string> = {
  '۰': '0', '۱': '1', '۲': '2', '۳': '3', '۴': '4', '۵': '5', '۶': '6', '۷': '7', '۸': '8', '۹': '9',
  '٠': '0', '١': '1', '٢': '2', '٣': '3', '٤': '4', '٥': '5', '٦': '6', '٧': '7', '٨': '8', '٩': '9',
};

/**
 * A typed number in any digit script («۵٫۲», «5,2», «118») → number, or null
 * when empty / not a number. `decimals` 0 drops anything after a separator.
 */
export function parseLocaleNumber(raw: string, decimals: number): number | null {
  const ascii = raw
    .trim()
    .replace(/[۰-۹٠-٩]/g, (d) => DIGITS[d] ?? d)
    .replace(/[٫,،/]/g, '.');
  if (!ascii) return null;
  if (!/^\d+(\.\d*)?$/.test(ascii)) return null;
  const n = Number(ascii);
  if (!Number.isFinite(n)) return null;
  const f = 10 ** decimals;
  return Math.trunc(n * f) / f;
}
