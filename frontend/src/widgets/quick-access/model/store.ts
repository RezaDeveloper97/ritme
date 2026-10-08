import { create } from 'zustand';

import { onSessionEnd } from '@/shared/session';

/**
 * Today «دسترسی سریع» picks — on this device only (docs/canvas-build/README.md §8 Q3: local first; a server
 * preference can replace this module later without touching the UI).
 *
 * Shortcut ids are not health data, but which tools she pinned can hint at it (a pain diary), so the list
 * is stamped with the account that saved it and dropped for another one, and wiped at sign-out — the same
 * rules as the search recents (CB-NAV-02). Storage can be missing or throw (private mode): then the defaults
 * show and an edit lasts for this session only.
 */
export const QUICK_ACCESS_KEY = 'ritme_quick_access';

interface Stored {
  owner: string;
  ids: string[];
}

function storage(): Storage | null {
  try {
    return typeof window === 'undefined' ? null : window.localStorage;
  } catch {
    return null;
  }
}

/** `owner`'s saved picks, `null` when she never edited them (or they belong to another account). */
export function readQuickAccess(owner: string): string[] | null {
  try {
    const raw = storage()?.getItem(QUICK_ACCESS_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<Stored> | null;
    if (!parsed || typeof parsed.owner !== 'string' || !Array.isArray(parsed.ids)) return null;
    if (parsed.owner !== owner) {
      clearQuickAccess();
      return null;
    }
    return parsed.ids.filter((v): v is string => typeof v === 'string' && /^[a-zA-Z]{1,32}$/.test(v)).slice(0, 32);
  } catch {
    return null;
  }
}

export function writeQuickAccess(owner: string, ids: readonly string[]): void {
  try {
    storage()?.setItem(QUICK_ACCESS_KEY, JSON.stringify({ owner, ids: [...ids] } satisfies Stored));
  } catch {
    // no storage: the pick lives in memory for this session
  }
}

export function clearQuickAccess(): void {
  try {
    storage()?.removeItem(QUICK_ACCESS_KEY);
  } catch {
    // nothing persisted
  }
}

interface QuickAccessState {
  /** Whose picks are loaded (`null` = not loaded yet). */
  owner: string | null;
  /** The picks; `null` = never edited → the mode's defaults. */
  ids: string[] | null;
  load: (owner: string) => void;
  save: (owner: string, ids: string[] | null) => void;
}

/** Shared by the Today row and the edit sheet (a separate tree under `AppSheet`), so a save repaints the row. */
export const useQuickAccessStore = create<QuickAccessState>((set, get) => ({
  owner: null,
  ids: null,
  load: (owner) => {
    if (get().owner === owner) return;
    set({ owner, ids: readQuickAccess(owner) });
  },
  save: (owner, ids) => {
    if (ids === null) clearQuickAccess();
    else writeQuickAccess(owner, ids);
    set({ owner, ids });
  },
}));

onSessionEnd(() => {
  clearQuickAccess();
  useQuickAccessStore.setState({ owner: null, ids: null });
});
