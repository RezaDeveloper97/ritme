import type { OnboardingState } from './state';

/**
 * The onboarding «داروها» pick that means hormonal contraception (§2.40): the pill or another hormonal
 * medication. `iud` stays out because onboarding cannot tell copper from hormonal.
 */
export function usesHormonalMethod(state: Pick<OnboardingState, 'conditions'>): boolean {
  const meds = state.conditions.medications ?? [];
  return meds.includes('contraceptive_pill') || meds.includes('hormonal_medication');
}
