import type { PlusQuote } from '@/entities/plus';

export type CodeState = 'none' | 'accepted' | 'outranked';

/**
 * What a quote priced with an entered code means for the code field (B-N2-11b, B-2).
 * The code is "accepted" only when the code itself priced the quote; during a trial
 * the larger offer wins (no stacking) and the server answers `discount_source:
 * 'trial_offer'` with no code — the code was outranked, not applied.
 */
export function codeState(quote: Pick<PlusQuote, 'discountSource'> | undefined): CodeState {
  if (!quote) return 'none';
  return quote.discountSource === 'code' ? 'accepted' : 'outranked';
}
