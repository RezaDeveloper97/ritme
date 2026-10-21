import type { CycleView } from './types';

/** Engine warnings (§2.32) that mean the fertile window / ovulation must not be shown. */
const SUPPRESSING_WARNINGS = ['hormonal_contraception_active', 'postpartum_cycles_not_returned'];

/**
 * Whether fertility (ovulation, fertile window, chance) is clinically suppressed
 * for this view. The engine's `limits.fertility_suppressed` is authoritative; an
 * older v6 build without `limits` is read from its warnings instead.
 */
export function isFertilitySuppressed(view: CycleView | null | undefined): boolean {
  if (!view) return false;
  if (view.limits?.fertilitySuppressed) return true;
  return (view.warnings ?? []).some((w) => SUPPRESSING_WARNINGS.includes(w));
}
