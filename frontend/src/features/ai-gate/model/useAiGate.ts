'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useCallback, useMemo, useState } from 'react';

import {
  aiCreditKeys,
  aiDenialOf,
  gateStateOf,
  isGateDenial,
  useAiCredits,
  type AiDenial,
  type AiGateHooks,
  type AiGateStatus,
} from '@/entities/ai-credit';
import { plusKeys } from '@/entities/plus';

import { guardDecision } from './gateDecision';
import { openAiGate } from './store';

export { denialFromBalance } from './gateDecision';

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
    const { allowed, denial: d } = guardDecision(state, credits.data, featureKey);
    if (d) openAiGate(d);
    return allowed;
  }, [state, credits.data, featureKey]);

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
