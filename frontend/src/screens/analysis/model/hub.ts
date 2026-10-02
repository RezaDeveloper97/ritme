import type { AnalysisSummary } from '@/entities/analysis';

/** Hub cards, in An_Hub order. */
export const HUB_CARDS = ['cycle', 'recentCycles', 'symptoms', 'moodByPhase', 'sleepMood', 'weight', 'vitals', 'labs'] as const;
export type HubCard = (typeof HUB_CARDS)[number];

/** The filter chips «همه · سیکل · علائم · حال و خواب · بدن · آزمایش». */
export const HUB_CATEGORIES = ['all', 'cycle', 'symptoms', 'moodSleep', 'body', 'labs'] as const;
export type HubCategory = (typeof HUB_CATEGORIES)[number];

/** The headed groups the cards sit in («سیکل», «علائم و حال», «بدن و آزمایش»). */
export const HUB_GROUPS = ['cycle', 'symptomsMood', 'bodyLabs'] as const;
export type HubGroup = (typeof HUB_GROUPS)[number];

const CARD_CATEGORY: Record<HubCard, Exclude<HubCategory, 'all'>> = {
  cycle: 'cycle',
  recentCycles: 'cycle',
  symptoms: 'symptoms',
  moodByPhase: 'moodSleep',
  sleepMood: 'moodSleep',
  weight: 'body',
  vitals: 'body',
  labs: 'labs',
};

const CARD_GROUP: Record<HubCard, HubGroup> = {
  cycle: 'cycle',
  recentCycles: 'cycle',
  symptoms: 'symptomsMood',
  moodByPhase: 'symptomsMood',
  sleepMood: 'symptomsMood',
  weight: 'bodyLabs',
  vitals: 'bodyLabs',
  labs: 'bodyLabs',
};

/**
 * Which life-stage hub `/analysis` shows. cycle / teen / menopause use this
 * hub (An_Hub); ttc gets An_Hub_TTC through AnalysisPage's `ttcHub` slot
 * (B-N3-11), postpartum and the companion fall back to this one until B-N5-07
 * (An_Hub_Post); pregnancy has no
 * cycle data, so it gets a placeholder until B-N3-12 (An_Hub_Preg).
 */
export type HubVariant = 'cycle' | 'teen' | 'menopause' | 'pregnancy';

export function hubVariant(mode: string | null): HubVariant {
  if (mode === 'teen' || mode === 'menopause' || mode === 'pregnancy') return mode;
  return 'cycle';
}

/**
 * Cards a variant shows. Teens get no Plus upsell (B-N2-03 hides the Plus card
 * for them), so locked Plus cards are dropped; menopause drops the cycle cards
 * that only ask for cycles; the rest always render — a card without enough data
 * explains what is missing instead of disappearing.
 */
export function visibleCards(summary: AnalysisSummary, variant: HubVariant): HubCard[] {
  return HUB_CARDS.filter((card) => {
    const section = summary.sections[card];
    if (card === 'recentCycles' && !section.ready) return false;
    if (variant === 'teen' && section.locked) return false;
    // Menopause tracks no cycles (B-N3-14b): no «۲ سیکل کامل لازم است» card, no mood-by-phase.
    if (variant === 'menopause' && (card === 'moodByPhase' || (card === 'cycle' && !section.ready))) return false;
    return true;
  });
}

/** Whether a card's empty state may ask for more cycles (not in a mode without cycle tracking). */
export function asksForCycles(variant: HubVariant): boolean {
  return variant !== 'menopause';
}

/** Chips worth offering: «همه» plus every category that has a visible card. */
export function availableCategories(cards: readonly HubCard[]): HubCategory[] {
  const present = new Set(cards.map((c) => CARD_CATEGORY[c]));
  return HUB_CATEGORIES.filter((c) => c === 'all' || present.has(c));
}

/** The visible cards grouped under their headings for `category`, empty groups dropped. */
export function groupCards(cards: readonly HubCard[], category: HubCategory): Array<{ group: HubGroup; cards: HubCard[] }> {
  const shown = cards.filter((c) => category === 'all' || CARD_CATEGORY[c] === category);
  return HUB_GROUPS.map((group) => ({ group, cards: shown.filter((c) => CARD_GROUP[c] === group) })).filter(
    (g) => g.cards.length > 0,
  );
}

/** «−۰٫۶» style signed delta, one decimal (U+2212 minus, as the artboards print it). */
export function signedDecimal(value: number): { sign: '−' | '+' | ''; abs: string } {
  const rounded = Math.round(value * 10) / 10;
  const abs = Math.abs(rounded).toFixed(1).replace(/\.0$/, '');
  if (rounded === 0) return { sign: '', abs: '0' };
  return { sign: rounded < 0 ? '−' : '+', abs };
}
