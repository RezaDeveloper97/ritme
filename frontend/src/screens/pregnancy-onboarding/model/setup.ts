import type { OnboardingInput } from '@/entities/pregnancy';

/*
 * Pure state + payload builders for the in-app pregnancy setup. The dating
 * itself is the sign-up form (`widgets/pregnancy-dating`); this adds the
 * optional history and the order of the finishing writes.
 */

export const SETUP_CONDITION_NONE = 'none';

export interface SetupHistory {
  miscarriage: boolean;
  highRisk: boolean;
  /** Condition keys; `['none']` is the exclusive «هیچ‌کدام» answer. */
  conditions: string[];
  bloodType: string | null;
  rhFactor: string | null;
}

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

/**
 * Adds the optional history to the v1 `POST /pregnancy/onboarding` dating
 * body (built by the shared dating form); only the fields the user gave.
 */
export function withHistory(body: OnboardingInput, h: SetupHistory | null): OnboardingInput {
  if (!h) return body;
  const out: OnboardingInput = { ...body };
  if (h.miscarriage) out.has_miscarriage_history = true;
  if (h.highRisk) out.has_high_risk_history = true;
  if (h.conditions.length) out.pre_existing_conditions = h.conditions;
  if (h.bloodType) out.blood_type = h.bloodType;
  if (h.rhFactor) out.rh_factor = h.rhFactor;
  return out;
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
