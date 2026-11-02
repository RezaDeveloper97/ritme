import type { CycleView } from '@/entities/cycle';

export interface WhyThisDate {
  source: 'sourceRecent' | 'sourceProfile' | 'sourceDefault';
  basisCycles: number | null;
  reasonCount: number;
}

/**
 * The "why this date" breakdown (A-11, Plus): which layer the forecast rests
 * on, how many finished cycles back it and how many reasons shaped the
 * confidence. Null when the engine has no forecast to explain.
 */
export function whyThisDate(view: CycleView | null | undefined): WhyThisDate | null {
  const f = view?.forecast;
  if (!view || !f) return null;
  const source = f.source === 'recent_valid_cycles' ? 'sourceRecent' : f.source === 'profile' ? 'sourceProfile' : 'sourceDefault';
  const reasons = f.confidenceReasons.length ? f.confidenceReasons : view.confidenceReasons;
  return { source, basisCycles: view.calculatedValues?.basedOnCycles ?? null, reasonCount: reasons.length };
}
