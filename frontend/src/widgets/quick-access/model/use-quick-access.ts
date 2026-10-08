'use client';

import { useEffect } from 'react';

import { useCurrentUser, useLifeStage } from '@/entities/user';

import { type QuickMode, type Shortcut, visibleShortcuts } from './shortcuts';
import { useQuickAccessStore } from './store';

export interface QuickAccessState {
  /** `null` until the account and its life stage are known (nothing is drawn meanwhile). */
  mode: QuickMode | null;
  owner: string | null;
  /** Stored picks (`null` = never edited). */
  stored: string[] | null;
  visible: Shortcut[];
}

/** The life stage as a quick-access mode — IVF is the TTC sub-mode (CB-IVF-02); a companion has no Today row. */
export function useQuickAccess(): QuickAccessState {
  const me = useCurrentUser();
  const life = useLifeStage();
  const owner = me.data ? String(me.data.id) : null;
  const { owner: loaded, ids, load } = useQuickAccessStore();
  useEffect(() => {
    if (owner) load(owner);
  }, [owner, load]);

  const data = life.data;
  const mode: QuickMode | null =
    !data || data.companion ? null : data.mode === 'ttc' && data.ivfIui ? 'ivf' : data.mode;
  const stored = owner && loaded === owner ? ids : null;
  const ready = mode !== null && owner !== null && loaded === owner;
  return { mode: ready ? mode : null, owner, stored, visible: ready && mode ? visibleShortcuts(mode, stored) : [] };
}
