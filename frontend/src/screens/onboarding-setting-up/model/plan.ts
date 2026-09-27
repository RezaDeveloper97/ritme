import type { OnboardingInput } from '@/entities/pregnancy';
import { onboardingRoute, type OnboardingData, type PregnancyBasis } from '@/entities/user';
import { datePartsToApiDate } from '@/features/edit-profile';
import type { Locale } from '@/shared/i18n';

/*
 * What the setting-up screen does with the collected answers (T-M7-16 bug 7).
 * Pure so the decision is unit-tested: a store that lost its answers (not yet
 * hydrated, or reset by a logout in the same tab) must never be saved as a
 * cycle profile, and a pregnant signup must always be activated and dated.
 */

/** One request the screen sends, in this order. */
export type SetupStep = 'profile' | 'activate' | 'onboarding';

export type SetupPlan =
  /** An answer the save depends on is missing — send the user back to that step. */
  | { kind: 'resume'; route: string }
  | {
      kind: 'save';
      steps: SetupStep[];
      /** The pregnancy onboarding body (only on the pregnant path). */
      pregnancy: OnboardingInput | null;
      /** Where the user lands once every step succeeded. */
      landing: '/pregnancy' | '/home';
    };

/**
 * The collected dating basis as the pregnancy onboarding body, or null when
 * the source or the fields it needs are missing. The stored parts are in
 * `locale`'s calendar; they cross the boundary as Gregorian (§7).
 */
export function buildPregnancyOnboarding(basis: PregnancyBasis, locale: Locale): OnboardingInput | null {
  switch (basis.source) {
    case 'lmp':
      return basis.lmp ? { age_source: 'lmp', lmp_date: datePartsToApiDate(basis.lmp, locale) } : null;
    case 'ultrasound':
      return basis.ultrasoundDate && basis.ultrasoundWeeks != null
        ? {
            age_source: 'ultrasound',
            ultrasound_date: datePartsToApiDate(basis.ultrasoundDate, locale),
            ultrasound_weeks: basis.ultrasoundWeeks,
            ultrasound_days: basis.ultrasoundDays ?? 0,
          }
        : null;
    case 'manual':
      return basis.manualWeeks != null
        ? { age_source: 'manual', manual_weeks: basis.manualWeeks, manual_days: basis.manualDays ?? 0 }
        : null;
    default:
      return null;
  }
}

export function planSetup(o: Pick<OnboardingData, 'intention' | 'pregnancyBasis' | 'locale'>): SetupPlan {
  if (!o.intention) return { kind: 'resume', route: onboardingRoute('intention') };
  if (o.intention !== 'pregnant') return { kind: 'save', steps: ['profile'], pregnancy: null, landing: '/home' };
  const pregnancy = buildPregnancyOnboarding(o.pregnancyBasis, o.locale);
  if (!pregnancy) return { kind: 'resume', route: onboardingRoute('pregnancyBasis') };
  return { kind: 'save', steps: ['profile', 'activate', 'onboarding'], pregnancy, landing: '/pregnancy' };
}

/**
 * Run the plan's steps in order, skipping those already in `done` and adding
 * each one that succeeds — so a retry after a failure resumes where it
 * stopped instead of re-activating. Rejects with the first failure.
 */
export async function runSetup(
  steps: readonly SetupStep[],
  run: (step: SetupStep) => Promise<unknown>,
  done: Set<SetupStep>,
): Promise<void> {
  for (const step of steps) {
    if (done.has(step)) continue;
    await run(step);
    done.add(step);
  }
}
