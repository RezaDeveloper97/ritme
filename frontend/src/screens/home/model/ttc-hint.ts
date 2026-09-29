import { onSessionEnd } from '@/shared/session';

/**
 * The last known home layout (TTC or not), so a cold load can render the home
 * without waiting on `GET /profile` (T-M5-12). The home needs exactly one bit
 * of the profile to pick its layout — `pregnancy_intention === 'trying'` — and
 * holding the whole page on that request made a slow `/profile` block the ring,
 * the phase card and the tiles.
 *
 * Privacy (CLAUDE.md §11): only this one boolean is kept, never the profile
 * itself, and it is wiped with the session (`onSessionEnd`), so the next account
 * on the device starts without it. The fresh profile always wins once it lands.
 */
const KEY = 'ritme_home_ttc';

function storage(): Storage | null {
  try {
    return typeof window === 'undefined' ? null : window.localStorage;
  } catch {
    return null; // blocked storage (private mode, sandboxed frame)
  }
}

/** The remembered layout, or `null` when nothing is known yet. */
export function readTtcHint(): boolean | null {
  try {
    const value = storage()?.getItem(KEY);
    if (value === '1') return true;
    if (value === '0') return false;
  } catch {
    // unreadable storage: behave as a first load
  }
  return null;
}

/** Remember the layout the fresh profile decided. */
export function writeTtcHint(isTtc: boolean): void {
  try {
    storage()?.setItem(KEY, isTtc ? '1' : '0');
  } catch {
    // quota / blocked storage: the next cold load just waits for the profile
  }
}

/** Forget the layout (session end). */
export function clearTtcHint(): void {
  try {
    storage()?.removeItem(KEY);
  } catch {
    // nothing stored we could reach
  }
}

onSessionEnd(clearTtcHint);
