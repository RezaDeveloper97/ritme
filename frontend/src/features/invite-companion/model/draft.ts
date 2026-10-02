import {
  type CompanionGrants,
  type CompanionType,
  type CreateCompanionInput,
  emptyGrants,
} from '@/entities/companion';

/*
 * The «افزودن همدم» wizard (Hamdam_Type → Access → Children → Invite → Done).
 * Client state only — nothing is sent until the invite step. The invite code
 * that comes back is held by the component, never here.
 */

export type InviteStep = 'type' | 'access' | 'children' | 'invite' | 'done';

export interface InviteDraft {
  type: CompanionType | null;
  /** Starts with every section on `none` (CLAUDE.md §11: most private by default). */
  grants: CompanionGrants;
  name: string;
  phone: string;
}

export function initialDraft(): InviteDraft {
  return { type: null, grants: emptyGrants(), name: '', phone: '' };
}

/** The numbered steps for a type: the children step exists only for a spouse. */
export function numberedSteps(type: CompanionType | null): InviteStep[] {
  return type === 'partner' ? ['type', 'access'] : ['type', 'access', 'children'];
}

/** The step after `step` (the invite form follows the last numbered step). */
export function nextStep(step: InviteStep, type: CompanionType | null): InviteStep {
  switch (step) {
    case 'type':
      return 'access';
    case 'access':
      return type === 'spouse' ? 'children' : 'invite';
    case 'children':
      return 'invite';
    default:
      return 'done';
  }
}

/** Where the header's back button goes; `null` = leave the wizard. */
export function previousStep(step: InviteStep, type: CompanionType | null): InviteStep | null {
  switch (step) {
    case 'access':
      return 'type';
    case 'children':
      return 'access';
    case 'invite':
      return type === 'spouse' ? 'children' : 'access';
    default:
      return null;
  }
}

const PERSIAN = '۰۱۲۳۴۵۶۷۸۹';
const ARABIC = '٠١٢٣٤٥٦٧٨٩';

/**
 * Mirrors the backend's `NormalizeMobile`: Persian/Arabic digits → Latin,
 * spaces/dashes/brackets/+ dropped, 0098 / 98 / 9xxxxxxxxx → 09xxxxxxxxx.
 * `null` when it cannot be an Iranian mobile.
 */
export function normalizeMobile(input: string): string | null {
  let digits = '';
  for (const ch of input) {
    const p = PERSIAN.indexOf(ch);
    const a = ARABIC.indexOf(ch);
    if (p >= 0) digits += String(p);
    else if (a >= 0) digits += String(a);
    else if (ch >= '0' && ch <= '9') digits += ch;
    else if (' -+()'.includes(ch)) continue;
    else return null;
  }
  if (digits.startsWith('0098')) digits = `0${digits.slice(4)}`;
  else if (digits.startsWith('98') && digits.length === 12) digits = `0${digits.slice(2)}`;
  else if (digits.startsWith('9') && digits.length === 10) digits = `0${digits}`;
  return /^09\d{9}$/.test(digits) ? digits : null;
}

/** A filled phone that cannot be sent; an empty phone is fine (code-only invite). */
export function phoneProblem(phone: string): boolean {
  return phone.trim() !== '' && normalizeMobile(phone) === null;
}

/** POST /companions body; `withPhone: false` = «فقط کد بساز» (no SMS even if a number was typed). */
export function draftToInput(draft: InviteDraft, withPhone: boolean): CreateCompanionInput | null {
  if (!draft.type) return null;
  const phone = withPhone ? normalizeMobile(draft.phone) : null;
  return {
    type: draft.type,
    grants: draft.grants,
    ...(draft.name.trim() ? { displayName: draft.name.trim().slice(0, 100) } : {}),
    ...(phone ? { phone } : {}),
  };
}
