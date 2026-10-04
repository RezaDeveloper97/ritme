import {
  IVF_FOLLICLE_BINS,
  IVF_OVARIES,
  IVF_SCAN_LIMITS,
  type IvfE2Unit,
  type IvfGrowthPoint,
  type IvfOvaryCounts,
  type IvfScan,
  type IvfScanInput,
} from '@/entities/ivf';
import { diffInDays, fromApiDate } from '@/shared/lib/date';
import { toAsciiDigits } from '@/shared/lib/phone';

/*
 * Pure helpers of the scan log (nbl_IVF_Scan, CB-IVF-04). No React, no copy:
 * the UI turns the keys into text.
 */

/** Editable state of one day's scan. Decimal fields stay text while typed («۸٫۵»). */
export interface ScanForm {
  right: IvfOvaryCounts;
  left: IvfOvaryCounts;
  endometrium: string;
  e2: string;
  e2Unit: IvfE2Unit;
}

/** How many scan days the growth chart shows (the most recent ones). */
export const GROWTH_POINTS = 6;

export function emptyCounts(): IvfOvaryCounts {
  return Object.fromEntries(IVF_FOLLICLE_BINS.map((bin) => [bin, 0])) as IvfOvaryCounts;
}

/** «8.50» → «8.5», «1250.00» → «1250» (the API pads decimals), then shown with `format` (locale digits). */
function trimDecimal(value: string | null, format: (v: string) => string): string {
  if (value === null) return '';
  const n = Number(value);
  return Number.isFinite(n) ? format(String(n)) : '';
}

/**
 * The form of a day: its saved scan, or zeros and empty fields. `format`
 * renders a saved decimal in the locale's digits («۸٫۵»); `parseDecimal` reads it back.
 */
export function formFrom(scan: IvfScan | undefined, format: (v: string) => string = (v) => v): ScanForm {
  return {
    right: scan ? { ...scan.right } : emptyCounts(),
    left: scan ? { ...scan.left } : emptyCounts(),
    endometrium: trimDecimal(scan?.endometriumMm ?? null, format),
    e2: trimDecimal(scan?.e2 ?? null, format),
    e2Unit: scan?.e2Unit ?? 'pg_ml',
  };
}

export type DecimalResult = { ok: true; value: number | null } | { ok: false };

/**
 * Typed decimal → number: any digit script, «٫» / «,» as the separator;
 * empty = null; anything else non-numeric or outside 0..max is invalid.
 */
export function parseDecimal(text: string, max: number): DecimalResult {
  const normalized = toAsciiDigits(text.trim()).replace(/[٫,]/g, '.');
  if (normalized === '') return { ok: true, value: null };
  if (!/^\d+(\.\d+)?$|^\.\d+$/.test(normalized)) return { ok: false };
  const value = Number(normalized);
  return Number.isFinite(value) && value >= 0 && value <= max ? { ok: true, value } : { ok: false };
}

export interface FormCheck {
  input: IvfScanInput | null;
  errors: { endometrium: boolean; e2: boolean };
}

/** Validate the form and build the PUT body; `input` is null while a field is invalid. */
export function checkForm(form: ScanForm, date: string, notes: string | null): FormCheck {
  const endometrium = parseDecimal(form.endometrium, IVF_SCAN_LIMITS.endometriumMm);
  const e2 = parseDecimal(form.e2, IVF_SCAN_LIMITS.e2);
  const errors = { endometrium: !endometrium.ok, e2: !e2.ok };
  if (!endometrium.ok || !e2.ok) return { input: null, errors };
  return {
    errors,
    input: {
      date,
      right: { ...form.right },
      left: { ...form.left },
      endometriumMm: endometrium.value,
      e2: e2.value,
      e2Unit: e2.value === null ? null : form.e2Unit,
      notes,
    },
  };
}

/** Whether the form differs from the saved one (numbers compared as numbers, «8.50» = «8.5»). */
export function isDirty(base: ScanForm, form: ScanForm): boolean {
  for (const ovary of IVF_OVARIES) {
    for (const bin of IVF_FOLLICLE_BINS) if (base[ovary][bin] !== form[ovary][bin]) return true;
  }
  const same = (a: string, b: string, max: number) => {
    const x = parseDecimal(a, max);
    const y = parseDecimal(b, max);
    return x.ok && y.ok ? x.value === y.value : a.trim() === b.trim();
  };
  if (!same(base.endometrium, form.endometrium, IVF_SCAN_LIMITS.endometriumMm)) return true;
  if (!same(base.e2, form.e2, IVF_SCAN_LIMITS.e2)) return true;
  return form.e2.trim() !== '' && base.e2Unit !== form.e2Unit;
}

/** Stimulation day on `date` («روز ۷ تحریک»): day 1 is the start; null before it or without one. */
export function stimDayOn(date: string, stimStartedOn: string | null): number | null {
  if (!stimStartedOn) return null;
  const day = diffInDays(fromApiDate(date), fromApiDate(stimStartedOn)) + 1;
  return day >= 1 ? day : null;
}

/** A `?date=` value the screen accepts: a real `Y-m-d` not after today; else today. */
export function scanDate(param: string | undefined, today: string): string {
  if (!param || !/^\d{4}-\d{2}-\d{2}$/.test(param)) return today;
  const parsed = fromApiDate(param);
  if (Number.isNaN(parsed.getTime())) return today;
  return param > today ? today : param;
}

/** The chart's points: the most recent `GROWTH_POINTS` scan days, oldest first. */
export function growthWindow(growth: readonly IvfGrowthPoint[]): IvfGrowthPoint[] {
  return [...growth].sort((a, b) => a.date.localeCompare(b.date)).slice(-GROWTH_POINTS);
}
