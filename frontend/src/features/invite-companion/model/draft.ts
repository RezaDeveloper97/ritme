import {
  type CompanionAction,
  type CompanionGoal,
  type CompanionGrants,
  type CompanionSignal,
  type CompanionSignalScope,
  COMPANION_SIGNALS,
  type CreateCompanionInput,
  emptyGrants,
} from '@/entities/companion';

/*
 * «ریتمی همراه» v2 invite (companion-v2 design): intro + consent → goal
 * (+ «همسرم است» → children) → scope (per-section access) → preferred actions (+ optional name /
 * phone) → generating (POST /companions) → code → preview. Client state only —
 * nothing is sent before the generating step. The invite code that comes back
 * is held by the component, never here.
 */

export type InviteStep = 'intro' | 'goal' | 'children' | 'scope' | 'actions' | 'generating' | 'code' | 'preview';

export interface InviteDraft {
  /** «با به اشتراک‌گذاشتن … موافقم» — the intro CTA stays disabled until ticked. */
  consent: boolean;
  /** null = nothing picked yet; sent as `unknown`. */
  goal: CompanionGoal | null;
  /** «همسرم است»: creates a `spouse` link (family + shared children) instead of a `partner`. */
  spouse: boolean;
  /** Per-section access («نبیند · فقط دیدن · دیدن و ویرایش»), picked on the scope step. */
  grants: CompanionGrants;
  /** Coarse signals: not asked in the wizard (edited on the companion's detail screen) — all off (CLAUDE.md §11). */
  signals: CompanionSignalScope;
  /** Preferred support kinds, in the order she picked them. */
  actionPrefs: CompanionAction[];
  name: string;
  phone: string;
  /** The owner's own children to share (spouse only, `child_ids` on create). */
  childIds: number[];
}

/**
 * The signals the owner is actually offered. `mood` stays out until the product decides otherwise: SPEC §1.7 /
 * family.md §4 keep mood «never shared, any role», the Go member payload omits it, and a switch that changes
 * nothing for the member would be a misleading consent. The API still accepts the key, so nothing breaks when it
 * is allowed again — remove the filter and the switch appears.
 */
export const OFFERED_SIGNALS: readonly CompanionSignal[] = COMPANION_SIGNALS.filter((s) => s !== 'mood');

export function emptySignals(): CompanionSignalScope {
  return Object.fromEntries(COMPANION_SIGNALS.map((s) => [s, false])) as CompanionSignalScope;
}

export function initialDraft(): InviteDraft {
  return {
    consent: false,
    goal: null,
    spouse: false,
    grants: inviteGrants(),
    signals: emptySignals(),
    actionPrefs: [],
    name: '',
    phone: '',
    childIds: [],
  };
}

/**
 * Where the scope step starts: the cycle section at `view` (what «ریتمی همراه» is about), every other
 * section on `none` until she opens it herself.
 */
export function inviteGrants(): CompanionGrants {
  return { ...emptyGrants(), cycle: 'view' };
}

/** Add / remove one child id (order kept, no duplicates). */
export function toggleChildId(ids: readonly number[], id: number): number[] {
  return ids.includes(id) ? ids.filter((x) => x !== id) : [...ids, id];
}

/** Add / remove one preferred action, keeping the order she picked them in. */
export function toggleAction(prefs: readonly CompanionAction[], action: CompanionAction): CompanionAction[] {
  return prefs.includes(action) ? prefs.filter((a) => a !== action) : [...prefs, action];
}

/** Same set of ids, order ignored (children editor «dirty» check). */
export function sameChildIds(a: readonly number[], b: readonly number[]): boolean {
  return a.length === b.length && a.every((id) => b.includes(id));
}

/** Same signal scope (preferences editor «dirty» check). */
export function sameSignals(a: CompanionSignalScope, b: CompanionSignalScope): boolean {
  return COMPANION_SIGNALS.every((s) => a[s] === b[s]);
}

/** Same preferred actions, order included (the companion sees them in this order). */
export function sameActions(a: readonly CompanionAction[], b: readonly CompanionAction[]): boolean {
  return a.length === b.length && a.every((x, i) => b[i] === x);
}

/** The signals she switched on, in the canonical order (preview rows). */
export function sharedSignals(scope: CompanionSignalScope): CompanionSignal[] {
  return OFFERED_SIGNALS.filter((s) => scope[s]);
}

/** The step after `step`; the children step exists only for a spouse. */
export function nextStep(step: InviteStep, draft: Pick<InviteDraft, 'spouse'>): InviteStep {
  switch (step) {
    case 'intro':
      return 'goal';
    case 'goal':
      return draft.spouse ? 'children' : 'scope';
    case 'children':
      return 'scope';
    case 'scope':
      return 'actions';
    case 'actions':
      return 'generating';
    case 'generating':
      return 'code';
    default:
      return 'preview';
  }
}

/**
 * Where the header's back button goes; `null` = leave the wizard. Once the
 * invite exists (code / preview) going back past it would only create a
 * second one, so the code step leaves and the preview returns to the code.
 */
export function previousStep(step: InviteStep, draft: Pick<InviteDraft, 'spouse'>): InviteStep | null {
  switch (step) {
    case 'goal':
      return 'intro';
    case 'children':
      return 'goal';
    case 'scope':
      return draft.spouse ? 'children' : 'goal';
    case 'actions':
      return 'scope';
    case 'generating':
      return 'actions';
    case 'preview':
      return 'code';
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

/**
 * POST /companions body: the per-section access she picked, plus the v2 preferences.
 * `null` until the consent is given or while the typed phone is not a mobile.
 */
export function draftToInput(draft: InviteDraft): CreateCompanionInput | null {
  if (!draft.consent || phoneProblem(draft.phone)) return null;
  const phone = normalizeMobile(draft.phone);
  return {
    type: draft.spouse ? 'spouse' : 'partner',
    grants: { ...draft.grants },
    ...(draft.name.trim() ? { displayName: draft.name.trim().slice(0, 100) } : {}),
    ...(phone ? { phone } : {}),
    // Only a spouse shares children; a draft that switched «همسرم است» off again drops them.
    ...(draft.spouse && draft.childIds.length ? { childIds: [...draft.childIds] } : {}),
    preferences: {
      goal: draft.goal ?? 'unknown',
      signals: { ...draft.signals },
      actionPrefs: [...draft.actionPrefs],
    },
  };
}
