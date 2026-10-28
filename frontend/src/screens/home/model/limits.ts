import { isFertilitySuppressed, type CycleView } from '@/entities/cycle';

/** Which one calm line explains a limit on Home (§2.40, E-24). */
export type LimitNote = 'hormonal' | 'postpartum' | 'age45';

export interface HomeLimits {
  /** No fertile window, ovulation or fertility level anywhere on Home. */
  hideFertility: boolean;
  /** The next-period row is a withdrawal bleed (hormonal contraception), not an ovulation cycle. */
  withdrawalBleed: boolean;
  note: LimitNote | null;
  /** Positive test logged: offer pregnancy mode (the engine decides, the user chooses). */
  promptPregnancy: boolean;
}

const none: HomeLimits = { hideFertility: false, withdrawalBleed: false, note: null, promptPregnancy: false };

/** Reads the engine's `limits` / warnings. A Laravel payload (no limits, no warnings) changes nothing. */
export function homeLimits(view: CycleView | null | undefined): HomeLimits {
  if (!view) return none;
  const warnings = view.warnings ?? [];
  const reason = view.limits?.suppressionReason ?? null;
  const hormonal = reason === 'hormonal_contraception' || warnings.includes('hormonal_contraception_active');
  const postpartum = reason === 'postpartum_not_returned' || warnings.includes('postpartum_cycles_not_returned');
  const age45 = warnings.includes('perimenopause_age_range');
  return {
    hideFertility: isFertilitySuppressed(view),
    withdrawalBleed: hormonal,
    note: hormonal ? 'hormonal' : postpartum ? 'postpartum' : age45 ? 'age45' : null,
    promptPregnancy: view.limits?.promptPregnancyMode === true,
  };
}
