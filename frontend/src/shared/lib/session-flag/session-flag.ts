'use client';

import { useCallback, useSyncExternalStore } from 'react';

/*
 * Per-tab boolean flags that last until the tab closes (`sessionStorage`), e.g.
 * «don't show this banner again this session». Storage can be missing or throw
 * (private mode, blocked site data, SSR) — every access is guarded and a failure
 * reads as «not set», so the UI still renders.
 */

const PREFIX = 'ritme_flag_';
const listeners = new Set<() => void>();

export function readSessionFlag(key: string): boolean {
  try {
    return typeof window !== 'undefined' && window.sessionStorage.getItem(PREFIX + key) === '1';
  } catch {
    return false;
  }
}

/** Sets (or clears) the flag and notifies every hook reading it. */
export function writeSessionFlag(key: string, on = true): void {
  try {
    if (on) window.sessionStorage.setItem(PREFIX + key, '1');
    else window.sessionStorage.removeItem(PREFIX + key);
  } catch {
    // storage blocked: the flag simply doesn't persist past this render
  }
  listeners.forEach((notify) => notify());
}

function subscribe(notify: () => void): () => void {
  listeners.add(notify);
  return () => {
    listeners.delete(notify);
  };
}

/** `[isSet, set]` for one flag; `false` on the server and wherever storage is unavailable. */
export function useSessionFlag(key: string): [boolean, () => void] {
  const isSet = useSyncExternalStore(
    subscribe,
    () => readSessionFlag(key),
    () => false,
  );
  const set = useCallback(() => writeSessionFlag(key), [key]);
  return [isSet, set];
}
