import type { MenopauseMessage } from '@/entities/menopause';

/*
 * Pure helpers of the menopause home (CB-MENO-05, nbl_Meno_Home / Main).
 */

/**
 * In-app links the API may hand the home that have a screen today. Everything
 * else (/menopause/score, /treatment, /alert, /report until CB-MENO-07…11) is
 * dropped so the home never links to a 404.
 */
const ROUTED = [/^\/checkups$/, /^\/checkups\/\d+$/];

export function routedLink(link: string | null): string | null {
  if (!link) return null;
  return ROUTED.some((re) => re.test(link)) ? link : null;
}

export type ScoreTrend = { kind: 'better' | 'worse'; points: number } | { kind: 'same' } | null;

/** The pill next to the score: delta < 0 = fewer symptoms = better. */
export function scoreTrend(delta: number | null): ScoreTrend {
  if (delta === null) return null;
  if (delta === 0) return { kind: 'same' };
  return { kind: delta < 0 ? 'better' : 'worse', points: Math.abs(delta) };
}

/**
 * The messages the home lists as cards: the bleeding alert is drawn as the
 * urgent card on its own, and the stage tip already sits in the hero.
 */
export function listedMessages(messages: readonly MenopauseMessage[], heroTipCode: string | null): MenopauseMessage[] {
  return messages.filter((m) => m.key !== 'postmenopausal_bleeding' && m.key !== heroTipCode);
}

/** The running timer's length now, capped at the API's hour. */
export function flashElapsedSeconds(serverElapsedS: number, sinceFetchMs: number): number {
  return Math.min(3600, Math.max(0, serverElapsedS + Math.floor(sinceFetchMs / 1000)));
}
