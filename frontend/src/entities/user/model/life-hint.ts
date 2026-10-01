import { onSessionEnd } from '@/shared/session';

import { isLifeMode, type LifeMode } from './life-stage';

/**
 * The last effective life mode this device saw, so a cold start can pick the
 * right home before `/profile/life-stage` answers (same idea as the home's TTC
 * layout hint). A hint only — the fresh API value always wins.
 *
 * Privacy (CLAUDE.md §11): only the mode word is kept, and it is wiped with the
 * session (`onSessionEnd`), so the next account on the device starts without it.
 * Storage can be missing or throw (private mode); then there is simply no hint.
 */
const KEY = 'ritme_life_mode';

function storage(): Storage | null {
  try {
    return typeof window === 'undefined' ? null : window.localStorage;
  } catch {
    return null;
  }
}

export function readLifeModeHint(): LifeMode | null {
  try {
    const value = storage()?.getItem(KEY);
    return isLifeMode(value) ? value : null;
  } catch {
    return null;
  }
}

export function writeLifeModeHint(mode: LifeMode | null): void {
  try {
    if (mode) storage()?.setItem(KEY, mode);
    else storage()?.removeItem(KEY);
  } catch {
    // no storage: nothing to remember
  }
}

onSessionEnd(() => writeLifeModeHint(null));
