import {
  CUSTOM_INTERVAL_MONTHS,
  type CheckupDetail,
  type CheckupPerformer,
} from '@/entities/checkup';

/** Local state of the custom-checkup form. */
export interface CustomFormState {
  title: string;
  intervalMonths: number;
  performedBy: CheckupPerformer;
  /** `Y-m-d` or null. */
  lastDoneOn: string | null;
  note: string;
}

export const TITLE_MAX = 255;
export const NOTE_MAX = 2000;

export function emptyCustomForm(): CustomFormState {
  return { title: '', intervalMonths: 12, performedBy: 'doctor', lastDoneOn: null, note: '' };
}

/** Prefill from a custom checkup's detail (the API has no separate note field on the detail). */
export function fromDetail(detail: CheckupDetail): CustomFormState {
  return {
    title: detail.title,
    intervalMonths: detail.intervalMonths ?? 12,
    performedBy: detail.performedBy,
    lastDoneOn: detail.lastDoneOn,
    note: '',
  };
}

/** The chip key under `checkups.custom.intervals` (`m12`), or null for a non-chip value. */
export function intervalKey(months: number): `m${number}` | null {
  return (CUSTOM_INTERVAL_MONTHS as readonly number[]).includes(months) ? `m${months}` : null;
}

/** Null when valid, otherwise the field that failed. */
export function validateCustomForm(state: CustomFormState): 'title' | null {
  const title = state.title.trim();
  if (!title || title.length > TITLE_MAX) return 'title';
  return null;
}
