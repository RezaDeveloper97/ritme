import { type OnboardingDating, onboardingDatingSchema } from '@/entities/pregnancy';

/*
 * The `dating` block of the pregnancy status (T-M10-18, deviation D-169):
 * the basis in effect and the ACOG re-dating comparison the server made
 * between the LMP and a scan. The server decides; this only maps the
 * decision to the short line shown after a scan is saved. The schema is the
 * entity's (one parser for every `dating` block, T-M10-18c).
 */
export const pregnancyDatingSchema = onboardingDatingSchema;

export type PregnancyDating = OnboardingDating;

/** Key under `pregnancyV2.common.datingResult`. */
export type DatingResultKey = 'replaced' | 'kept';

/**
 * - the scan replaced the LMP dating → «… بر اساس سونوگرافی به‌روز شد»;
 * - an LMP and a scan were compared and the LMP stayed → «اختلاف کمتر از حد لازم بود؛ …»;
 * - nothing was compared (no LMP, no scan) or the user chose another basis → no line.
 */
export function datingResultKey(dating: PregnancyDating | null | undefined): DatingResultKey | null {
  if (!dating || dating.discrepancy_days == null) return null;
  if (dating.replaced && dating.basis === 'ultrasound') return 'replaced';
  if (!dating.replaced && dating.basis === 'lmp') return 'kept';
  return null;
}
