'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useCallback, useMemo, useState } from 'react';

import {
  aiCreditKeys,
  aiDenialOf,
  gateStateOf,
  isGateDenial,
  useAiCredits,
  type AiCredits,
  type AiDenial,
  type AiGateDenial,
  type AiGateHooks,
  type AiGateStatus,
} from '@/entities/ai-credit';
import { plusKeys } from '@/entities/plus';

import { openAiGate } from './store';

export interface AiGateControls extends AiGateHooks {
  status: AiGateStatus;
  /** Credits one use costs; null while unknown. */
  cost: number | null;
  /** Spendable credits; null while unknown. */
  balance: number | null;
  canUse: boolean;
  /** Call before starting an AI action: opens the gate sheet and returns false when the user cannot use it. */
  guard: () => boolean;
  /**
   * Call in EVERY catch / onError of an AI request. Subscription / credits / disabled denials open the gate sheet
   * (and refresh the balance) and return true — show nothing else. Anything else returns false and is exposed as
   * `denial` (consent / busy / unavailable) for the surface's own state.
   */
  handleError: (error: unknown) => boolean;
  /** The last non-gate denial `handleError` saw (consent / busy / unavailable), else null. */
  denial: AiDenial | null;
  /** Call after every successful AI request so the meter stays right. */
  refresh: () => void;
}

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
 * The ONLY way an AI surface handles access (CLAUDE.md «AI gate»). Fails open while the balance is unknown —
 * the server is the authority, so a failed balance read never blocks a user.
 */
export function useAiGate(featureKey: string): AiGateControls {
  const queryClient = useQueryClient();
  const credits = useAiCredits();
  const [denial, setDenial] = useState<AiDenial | null>(null);
  const state = useMemo(() => gateStateOf(credits.data, featureKey, credits.isLoading), [credits.data, credits.isLoading, featureKey]);

  const refresh = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: aiCreditKeys.all });
  }, [queryClient]);

  const guard = useCallback((): boolean => {
    if (state.canUse || !credits.data) return true;
    const d = denialFromBalance(credits.data, featureKey, state.status);
    if (d) openAiGate(d);
    return d === null;
  }, [state.canUse, state.status, credits.data, featureKey]);

  const handleError = useCallback(
    (error: unknown): boolean => {
      const d = aiDenialOf(error);
      if (isGateDenial(d)) {
        setDenial(null);
        openAiGate(d);
        void queryClient.invalidateQueries({ queryKey: aiCreditKeys.all });
        void queryClient.invalidateQueries({ queryKey: plusKeys.status() });
        return true;
      }
      setDenial(d);
      return false;
    },
    [queryClient],
  );

  return { ...state, guard, handleError, denial, refresh };
}
