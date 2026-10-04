import type { DatingPreviewInput, DatingSource, OnboardingInput } from '@/entities/pregnancy';
import { datePartsToApiDate } from '@/features/edit-profile';
import type { Locale } from '@/shared/i18n';
import type { DateParts } from '@/shared/lib/date';

/*
 * Pure state + payload builders for the pregnancy Setup v2 flow (T-M7-09).
 * Dates are held in the locale's calendar (DateParts) and converted to
 * Gregorian ISO only when a request body is built (CLAUDE.md §7).
 */

export const SETUP_CONDITION_NONE = 'none';

export interface SetupDating {
  source: DatingSource;
  lmp: DateParts | null;
  ultrasoundDate: DateParts | null;
  ultrasoundWeeks: number | null;
  ultrasoundDays: number;
  manualWeeks: number | null;
  manualDays: number;
}

export interface SetupHistory {
  miscarriage: boolean;
  highRisk: boolean;
  /** Condition keys; `['none']` is the exclusive «هیچ‌کدام» answer. */
  conditions: string[];
  bloodType: string | null;
  rhFactor: string | null;
}

export const EMPTY_DATING: SetupDating = {
  source: 'lmp',
  lmp: null,
  ultrasoundDate: null,
  ultrasoundWeeks: null,
  ultrasoundDays: 0,
  manualWeeks: null,
  manualDays: 0,
};

export const EMPTY_HISTORY: SetupHistory = {
  miscarriage: false,
  highRisk: false,
  conditions: [],
  bloodType: null,
  rhFactor: null,
};

/** Toggle a condition chip; «هیچ‌کدام» is exclusive with every other chip. */
export function toggleCondition(current: readonly string[], value: string): string[] {
  if (value === SETUP_CONDITION_NONE) {
    return current.includes(SETUP_CONDITION_NONE) ? [] : [SETUP_CONDITION_NONE];
  }
  const rest = current.filter((c) => c !== SETUP_CONDITION_NONE);
  return rest.includes(value) ? rest.filter((c) => c !== value) : [...rest, value];
}

/** «هیچ‌کدام» leads the condition chips (Setup artboard); the others keep their order. */
export function noneFirst(conditions: readonly string[]): string[] {
  return [
    ...conditions.filter((c) => c === SETUP_CONDITION_NONE),
    ...conditions.filter((c) => c !== SETUP_CONDITION_NONE),
  ];
}

/** Has the dating step got everything its source needs? */
export function isDatingComplete(d: SetupDating): boolean {
  switch (d.source) {
    case 'lmp':
      return d.lmp != null;
    case 'ultrasound':
      return d.ultrasoundDate != null && d.ultrasoundWeeks != null;
    case 'manual':
      return d.manualWeeks != null;
  }
}

/** Body for `POST /pregnancy/v2/dating-preview`, or null if incomplete. */
export function toPreviewInput(d: SetupDating, locale: Locale): DatingPreviewInput | null {
  if (!isDatingComplete(d)) return null;
  switch (d.source) {
    case 'lmp':
      return { source: 'lmp', lmp_date: datePartsToApiDate(d.lmp!, locale) };
    case 'ultrasound':
      return {
        source: 'ultrasound',
        ultrasound_date: datePartsToApiDate(d.ultrasoundDate!, locale),
        ultrasound_weeks: d.ultrasoundWeeks!,
        ultrasound_days: d.ultrasoundDays,
      };
    case 'manual':
      return { source: 'manual', manual_weeks: d.manualWeeks!, manual_days: d.manualDays };
  }
}

/** Body for v1 `POST /pregnancy/onboarding`; history fields only when given. */
export function toOnboardingInput(
  d: SetupDating,
  h: SetupHistory | null,
  locale: Locale,
): OnboardingInput | null {
  const preview = toPreviewInput(d, locale);
  if (!preview) return null;
  const { source, ...dating } = preview;
  const body: OnboardingInput = { age_source: source, ...dating };
  if (!h) return body;
  if (h.miscarriage) body.has_miscarriage_history = true;
  if (h.highRisk) body.has_high_risk_history = true;
  if (h.conditions.length) body.pre_existing_conditions = h.conditions;
  if (h.bloodType) body.blood_type = h.bloodType;
  if (h.rhFactor) body.rh_factor = h.rhFactor;
  return body;
}

/** The three writes of «تمومه», injected so the order is unit-tested without a server. */
export interface SetupFinishSteps {
  /** POST /pregnancy/activate. */
  activate: () => Promise<unknown>;
  /** POST /pregnancy/onboarding with the dating (+ history) body. */
  onboard: () => Promise<unknown>;
  /** PUT /profile/life-stage {mode: 'pregnancy'} — the stored mode. */
  storeMode: () => Promise<unknown>;
}

/**
 * Finishes the setup: activate → onboarding → store the mode. The mode page no
 * longer stores `pregnancy` before the wizard, so nothing is half-switched when
 * the user leaves it early (stage B-3); the stored mode follows only once the
 * pregnancy profile exists. A failure stops the chain (the caller shows the
 * error and «تمومه» can be pressed again — activate is idempotent).
 */
export async function finishPregnancySetup(steps: SetupFinishSteps): Promise<void> {
  await steps.activate();
  await steps.onboard();
  await steps.storeMode();
}
