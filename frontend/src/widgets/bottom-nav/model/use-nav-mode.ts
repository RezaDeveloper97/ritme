'use client';

import { useUserMode } from '@/entities/message';
import { useLifeStage } from '@/entities/user';
import { getApiErrorCode } from '@/shared/api';

import { resolveNavMode, type NavMode } from './nav-items';

export interface NavModeState {
  /** `null` while unknown (loading, or both reads failed → callers treat it as `cycle`). */
  mode: NavMode | null;
  pending: boolean;
  /**
   * CB-IVF-02: a `ttc` user with bloom's «IVF/IUI» switch on — the IVF sub-mode
   * nav (امروز → `/ivf`, «درمان»). Only the life-stage read knows it.
   */
  ivf: boolean;
}

/**
 * The nav mode of the signed-in user: the effective life-stage mode
 * (`GET /profile/life-stage`, B-N2-03) — menopause / teen / postpartum exist
 * only there — else the legacy `/messages/mode` + TTC flag (production still on
 * Laravel has no life-stage route). Both are fetched in parallel.
 */
export function useNavMode(): NavModeState {
  const life = useLifeStage();
  const legacy = useUserMode();
  // B-N4-05: a male companion account gets the companion nav (امروز · خدمات · من, no FAB).
  if (life.data?.companion) return { mode: 'companion', pending: false, ivf: false };
  if (life.data) {
    const mode = resolveNavMode({ lifeMode: life.data.mode });
    return { mode, pending: false, ivf: mode === 'ttc' && life.data.ivfIui };
  }
  const lifePending = life.isPending && life.fetchStatus !== 'idle';
  const legacyPending = legacy.isPending && legacy.fetchStatus !== 'idle';
  if (lifePending) return { mode: null, pending: true, ivf: false };
  if (getApiErrorCode(legacy.error) === 'companion_account') return { mode: 'companion', pending: false, ivf: false };
  if (legacy.data) {
    return { mode: resolveNavMode({ mode: legacy.data.mode, isTtc: legacy.data.isTtc }), pending: false, ivf: false };
  }
  return { mode: null, pending: legacyPending, ivf: false };
}
