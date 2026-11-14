'use client';

import { useCallback, useSyncExternalStore } from 'react';

/*
 * «بی‌صدا کردن پیشنهادها» (compCards bell): a per-device preference of the viewer that hides the suggestion
 * sections on the companion home. Browser storage is a convenience here — it can be missing (private mode,
 * cleared data), so every access is guarded and a failure simply reads as «not muted».
 */
const KEY = 'ritme_companion_suggestions_muted';
const EVENT = 'ritme:companion-suggestions-muted';
/** In-memory copy for when storage throws, so the choice still holds until the page reloads. */
let memory = false;

function read(): boolean {
  try {
    return window.localStorage.getItem(KEY) === '1';
  } catch {
    return memory;
  }
}

function subscribe(onChange: () => void): () => void {
  window.addEventListener('storage', onChange);
  window.addEventListener(EVENT, onChange);
  return () => {
    window.removeEventListener('storage', onChange);
    window.removeEventListener(EVENT, onChange);
  };
}

export function useSuggestionsMuted(): [boolean, (muted: boolean) => void] {
  const muted = useSyncExternalStore(subscribe, read, () => false);
  const set = useCallback((next: boolean) => {
    memory = next;
    try {
      if (next) window.localStorage.setItem(KEY, '1');
      else window.localStorage.removeItem(KEY);
    } catch {
      // Storage unavailable: the choice lasts until the page reloads at most.
    }
    window.dispatchEvent(new Event(EVENT));
  }, []);
  return [muted, set];
}
