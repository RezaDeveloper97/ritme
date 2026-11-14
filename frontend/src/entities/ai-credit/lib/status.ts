import type { AiCredits } from '../model/types';

export type AiGateStatus = 'loading' | 'ready' | 'no_subscription' | 'no_credits' | 'disabled';

export interface AiGateState {
  status: AiGateStatus;
  /** Credits one use costs; null while unknown. */
  cost: number | null;
  /** Spendable credits; null while unknown. */
  balance: number | null;
  canUse: boolean;
}

/**
 * The gate status of one feature from the balance read. It FAILS OPEN: with no balance (loading, failed read,
 * feature missing from the catalogue) the user is never blocked — the server is the authority. Only a positive
 * statement from the balance closes the door.
 */
export function gateStateOf(credits: AiCredits | undefined, featureKey: string, loading: boolean): AiGateState {
  if (!credits) return { status: loading ? 'loading' : 'ready', cost: null, balance: null, canUse: true };
  const feature = credits.features.find((f) => f.key === featureKey);
  const cost = feature?.creditCost ?? null;
  const balance = credits.spendable;
  if (!credits.canUse) return { status: 'no_subscription', cost, balance, canUse: false };
  if (feature && !feature.available) return { status: 'disabled', cost, balance, canUse: false };
  if (feature && !feature.affordable) return { status: 'no_credits', cost, balance, canUse: false };
  if (cost !== null && balance < cost) return { status: 'no_credits', cost, balance, canUse: false };
  return { status: 'ready', cost, balance, canUse: true };
}

/** Share (0–1) of the period's credits still left; 0 for an empty plan. */
export function planShare(credits: Pick<AiCredits, 'plan'>): number {
  return credits.plan.total > 0 ? Math.max(0, Math.min(1, credits.plan.remaining / credits.plan.total)) : 0;
}
