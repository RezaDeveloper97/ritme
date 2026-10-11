import { onSessionEnd } from '@/shared/session';

/**
 * «Last server sync» (bloom B-N10-07, QUESTIONS Q11): the app has no explicit backup concept (data lives in her
 * account on the server), so the moment of the latest successful authenticated API write stands in for it. The
 * timestamp is the only thing stored (localStorage, this device); it is cleared when the session ends.
 */
const KEY = 'ritme_last_sync_at';

/** Paths whose writes say nothing about her data being saved (analytics, auth). */
const IGNORED = /^\/?(analytics|auth|otp)(\/|$)/;

export function markSynced(method: string, path: string, now: number = Date.now()): void {
  if (method === 'GET' || IGNORED.test(path)) return;
  try {
    window.localStorage.setItem(KEY, String(now));
  } catch {
    // storage blocked: the screen just shows «no sync yet»
  }
}

/** Epoch ms of the last successful write from this device, or null. */
export function readLastSync(): number | null {
  try {
    const raw = window.localStorage.getItem(KEY);
    const n = raw ? Number(raw) : NaN;
    return Number.isFinite(n) && n > 0 ? n : null;
  } catch {
    return null;
  }
}

export function clearLastSync(): void {
  try {
    window.localStorage.removeItem(KEY);
  } catch {
    // ignore
  }
}

if (typeof window !== 'undefined') onSessionEnd(clearLastSync);
