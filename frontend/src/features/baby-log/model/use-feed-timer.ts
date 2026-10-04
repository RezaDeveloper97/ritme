'use client';

import { useState } from 'react';

import { type FeedAction, useFeedAction, useFeedDay } from '../api/queries';
import { feedSeconds, sideSeconds } from './live';
import type { BabyFeed, FeedSide, FeedType } from './types';
import { useNow } from './use-now';

export const DEFAULT_BOTTLE_ML = 90;

/**
 * State of the Log_Feed timer (B-N5-07), shared by its panel and the screen's
 * «پایان و ذخیره» bar. The running feed is the server's (`active` of
 * `GET /children/{id}/feeds`), so it survives a reload; only the chosen tab
 * and the bottle / pump ml are local.
 */
export function useFeedTimer(childId: number, initialType: FeedType = 'breast') {
  const day = useFeedDay(childId);
  const action = useFeedAction(childId);
  const active: BabyFeed | null = day.data?.active ?? null;
  const [picked, setPicked] = useState<FeedType>(initialType);
  const [amountMl, setAmountMl] = useState<number>(DEFAULT_BOTTLE_ML);
  const [typeLocked, setTypeLocked] = useState(false);
  const now = useNow(!!active);

  /** The running feed decides the tab; otherwise the user's pick. */
  const type: FeedType = active?.type ?? picked;

  const run = (a: FeedAction) => {
    if (action.isPending) return;
    action.mutate(a);
  };

  const pickType = (next: FeedType) => {
    if (active && next !== active.type) {
      setTypeLocked(true);
      return;
    }
    setTypeLocked(false);
    setPicked(next);
  };

  /** A side card: start on it, switch to it, or pause it when it is the running one. */
  const tapSide = (side: FeedSide) => {
    if (!active) return run({ kind: 'start', type: 'breast', side });
    if (active.type !== 'breast') return;
    run({ kind: 'side', feedId: active.id, side: active.activeSide === side ? null : side });
  };

  const startTimer = () => {
    if (!active) run({ kind: 'start', type });
  };

  const finish = () => {
    if (!active) return;
    run({ kind: 'stop', feedId: active.id, amountMl: active.type === 'breast' ? null : amountMl });
  };

  const discard = () => {
    if (active) run({ kind: 'discard', feedId: active.id });
  };

  return {
    day,
    action,
    active,
    type,
    pickType,
    typeLocked,
    amountMl,
    setAmountMl,
    tapSide,
    startTimer,
    finish,
    discard,
    run,
    now,
    leftSeconds: active ? sideSeconds(active, 'left', now) : 0,
    rightSeconds: active ? sideSeconds(active, 'right', now) : 0,
    elapsedSeconds: active ? feedSeconds(active, now) : 0,
  };
}

export type FeedTimerState = ReturnType<typeof useFeedTimer>;
