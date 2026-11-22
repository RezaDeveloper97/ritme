import type { AiCredits, AiGateDenial, AiGateState, AiGateStatus } from '@/entities/ai-credit';

/** What the sheet should say when the balance (not a server answer) already tells the user cannot go on. */
export function denialFromBalance(credits: AiCredits, featureKey: string, status: AiGateStatus): AiGateDenial | null {
  const cost = credits.features.find((f) => f.key === featureKey)?.creditCost ?? 0;
  if (status === 'no_subscription')
    return { kind: 'subscription', code: 'ai_subscription_required', extraCredits: credits.extra.remaining, feature: featureKey, message: null };
  if (status === 'no_credits')
    return {
      kind: 'credits',
      code: 'ai_credits_exhausted',
      required: cost,
      balance: credits.spendable,
      planRemaining: credits.plan.remaining,
      extraCredits: credits.extra.remaining,
      periodEndsAt: credits.periodEndsAt,
      feature: featureKey,
      message: null,
    };
  if (status === 'disabled') return { kind: 'disabled', code: 'ai_feature_disabled', feature: featureKey, message: null };
  return null;
}

/**
 * The `guard()` decision as a pure function: `allowed` = start the action; `denial` = open the sheet with it.
 * Fails open while the balance is unknown (the server is the authority).
 */
export function guardDecision(
  state: Pick<AiGateState, 'canUse' | 'status'>,
  credits: AiCredits | undefined,
  featureKey: string,
): { allowed: boolean; denial: AiGateDenial | null } {
  if (state.canUse || !credits) return { allowed: true, denial: null };
  const denial = denialFromBalance(credits, featureKey, state.status);
  return { allowed: denial === null, denial };
}
