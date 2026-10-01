import {
  type ContraceptionMethod,
  type ContraceptionMethodCode,
  IUD_DEFAULT_YEARS,
  isPillMethod,
  type MethodPayload,
  type PackType,
} from '@/entities/contraception';

/** The setup form (nbl_Contra_Setup): the method plus the fields that method needs. */
export interface SetupDraft {
  method: ContraceptionMethodCode | null;
  packType: PackType;
  packStartedOn: string;
  reminderTime: string;
  /** Kept as saved; the refill row on the pack screen edits it. */
  packsLeft: number | null;
  insertedOn: string | null;
  iudYears: number;
  followupDone: boolean;
  injectedOn: string | null;
  replaceOn: string | null;
}

export const DEFAULT_REMINDER_TIME = '21:00';

/** A fresh form, or the saved method to edit. `today` = `Y-m-d`. */
export function initialDraft(saved: ContraceptionMethod | null, today: string): SetupDraft {
  return {
    method: saved?.method ?? null,
    packType: saved?.packType ?? '21_7',
    packStartedOn: saved?.packStartedOn ?? today,
    reminderTime: saved?.reminder?.time ?? DEFAULT_REMINDER_TIME,
    packsLeft: saved?.packsLeft ?? null,
    insertedOn: saved?.insertedOn ?? null,
    iudYears:
      saved?.iudLifetimeYears ??
      (saved?.method === 'hormonal_iud' ? IUD_DEFAULT_YEARS.hormonal_iud : IUD_DEFAULT_YEARS.copper_iud),
    followupDone: saved?.followupDone ?? false,
    injectedOn: saved?.injectedOn ?? null,
    replaceOn: saved?.replaceOn ?? null,
  };
}

/** Picking another method; an IUD switch moves the lifetime to that device's usual value unless saved. */
export function chooseMethod(draft: SetupDraft, method: ContraceptionMethodCode, saved: ContraceptionMethod | null): SetupDraft {
  const next = { ...draft, method };
  if ((method === 'copper_iud' || method === 'hormonal_iud') && saved?.method !== method) {
    next.iudYears = IUD_DEFAULT_YEARS[method];
  }
  return next;
}

export type DraftField = 'method' | 'packStartedOn' | 'insertedOn' | 'injectedOn';
export type DraftProblem = 'missing' | 'future';

/** What blocks saving, per field (the backend repeats these checks as 422s). */
export function draftProblems(draft: SetupDraft, today: string): Partial<Record<DraftField, DraftProblem>> {
  const out: Partial<Record<DraftField, DraftProblem>> = {};
  const date = (field: DraftField, value: string | null, required: boolean) => {
    if (!value) {
      if (required) out[field] = 'missing';
    } else if (value > today) out[field] = 'future';
  };
  switch (draft.method) {
    case null:
      out.method = 'missing';
      break;
    case 'combined_pill':
    case 'progestin_pill':
      date('packStartedOn', draft.packStartedOn, true);
      break;
    case 'copper_iud':
    case 'hormonal_iud':
      date('insertedOn', draft.insertedOn, true);
      break;
    case 'injection':
      date('injectedOn', draft.injectedOn, true);
      break;
    case 'implant':
      date('insertedOn', draft.insertedOn, false);
      break;
    default:
      break;
  }
  return out;
}

/** The `PUT /contraception/method` body; only the chosen method's fields. */
export function draftToPayload(draft: SetupDraft & { method: ContraceptionMethodCode }): MethodPayload {
  const { method } = draft;
  if (isPillMethod(method)) {
    return {
      method,
      pack_type: method === 'combined_pill' ? draft.packType : null,
      pack_started_on: draft.packStartedOn,
      packs_left: draft.packsLeft,
      reminder_time: draft.reminderTime,
      reminder_enabled: true,
    };
  }
  switch (method) {
    case 'copper_iud':
    case 'hormonal_iud':
      return { method, inserted_on: draft.insertedOn, iud_lifetime_years: draft.iudYears, followup_done: draft.followupDone };
    case 'injection':
      return { method, injected_on: draft.injectedOn };
    case 'implant':
      return { method, inserted_on: draft.insertedOn, replace_on: draft.replaceOn };
    default:
      return { method };
  }
}

const pad = (n: number) => String(n).padStart(2, '0');

/** `HH:MM` ↔ wheel indexes. */
export function parseClock(value: string): { hour: number; minute: number } {
  const m = /^(\d{1,2}):(\d{2})$/.exec(value);
  if (!m) return { hour: 21, minute: 0 };
  return { hour: Math.min(23, Number(m[1])), minute: Math.min(59, Number(m[2])) };
}

export function toClock(hour: number, minute: number): string {
  return `${pad(hour)}:${pad(minute)}`;
}
