import type { AiCredits, CreditUsageByDay, CreditUsageByFeature } from '@/entities/ai-credit';

export type UsageState = 'ready' | 'no_subscription' | 'exhausted';

/** What the hero explains: a running subscription with credits left, with none left, or no subscription at all. */
export function usageStateOf(credits: AiCredits): UsageState {
  if (!credits.canUse) return 'no_subscription';
  return credits.spendable <= 0 ? 'exhausted' : 'ready';
}

/** The extra card says its credits are kept (not usable) only when there are some and they are locked. */
export function extraKept(credits: AiCredits): boolean {
  return credits.extra.locked && credits.extra.remaining > 0;
}

/** The features with a cost, cheapest first, hiding switched-off slots. */
export function costRows(credits: AiCredits): AiCredits['features'] {
  return credits.features.filter((f) => f.available && f.creditCost > 0).sort((a, b) => a.creditCost - b.creditCost);
}

/** Features by credits spent, biggest first (zero rows dropped). */
export function featureRows(rows: readonly CreditUsageByFeature[]): CreditUsageByFeature[] {
  return rows.filter((r) => r.credits > 0).sort((a, b) => b.credits - a.credits);
}

/** The by-day series, oldest first; the API already sends one row per day with use. */
export function dayRows(rows: readonly CreditUsageByDay[]): CreditUsageByDay[] {
  return [...rows].sort((a, b) => a.date.localeCompare(b.date));
}
