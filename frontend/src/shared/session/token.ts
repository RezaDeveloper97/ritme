'use client';

/**
 * Session-token plumbing. This is domain-agnostic session state (not health
 * data), so it lives in `shared` where the HTTP client can read it — `shared`
 * may not import from `entities`/`features`, so the token can't live upward
 * (CLAUDE.md §3, §8).
 *
 * The JWT is kept in `localStorage` for attaching the bearer header, and that
 * copy is the client's source of truth. A separate, value-less cookie flag lets
 * the (edge) middleware gate routes without ever putting the token itself in a
 * cookie or URL (§11 — nothing sensitive leaves where it isn't needed).
 */

import {
  AUTH_COOKIE,
  AUTH_FLAG_MAX_AGE,
  ONBOARDING_COOKIE,
  SESSION_CLEARED_EVENT,
  SESSION_FLAG_ROUTE,
} from './cookie';
import { runSessionCleanups } from './cleanup';

const TOKEN_KEY = 'ritme_token';

/** Re-issue the server-set flag at most this often (start + resume both ask). */
const SERVER_FLAG_INTERVAL_MS = 60_000;

// In-memory mirror so reads work before/without localStorage and stay fast.
let inMemoryToken: string | null = null;
let lastServerFlagAt = 0;

const isBrowser = (): boolean => typeof window !== 'undefined';

export function getAuthToken(): string | null {
  if (inMemoryToken) return inMemoryToken;
  if (!isBrowser()) return null;
  try {
    inMemoryToken = window.localStorage.getItem(TOKEN_KEY);
  } catch {
    // Storage blocked (private mode, disabled site data): no persisted session.
    return null;
  }
  return inMemoryToken;
}

function writeScriptFlag(): void {
  // Flag only — never the token value.
  document.cookie = `${AUTH_COOKIE}=1; path=/; max-age=${AUTH_FLAG_MAX_AGE}; SameSite=Lax`;
}

/**
 * Ask the same-origin route to set the flag with an HTTP `Set-Cookie`, which
 * replaces the script-written copy and escapes WebKit's 7-day cap on those.
 * Fire-and-forget: offline, or on a host that routes `/api` elsewhere, the
 * script-written flag stays and nothing else changes.
 */
function requestServerFlag(force: boolean): void {
  if (typeof fetch !== 'function') return;
  const now = Date.now();
  if (!force && now - lastServerFlagAt < SERVER_FLAG_INTERVAL_MS) return;
  lastServerFlagAt = now;
  void fetch(SESSION_FLAG_ROUTE, {
    method: 'POST',
    credentials: 'same-origin',
    cache: 'no-store',
    keepalive: true,
  }).catch(() => undefined);
}

/**
 * Make sure the middleware can see the session. Writes the flag from script
 * only when it is missing (a script write would re-arm WebKit's cap on a
 * server-set flag), and asks the server to re-issue it on every start/resume so
 * its expiry keeps sliding forward.
 */
export function assertAuthFlag(): void {
  if (!isBrowser()) return;
  if (!hasAuthCookie()) writeScriptFlag();
  requestServerFlag(false);
}

export function setAuthToken(token: string): void {
  inMemoryToken = token;
  if (!isBrowser()) return;
  try {
    window.localStorage.setItem(TOKEN_KEY, token);
  } catch {
    // Keep the in-memory session for this page even if it can't persist.
  }
  // Written synchronously so the very next navigation passes the middleware;
  // the server copy then replaces it.
  writeScriptFlag();
  requestServerFlag(true);
}

export function clearAuthToken(): void {
  inMemoryToken = null;
  if (!isBrowser()) return;
  // Per-user device data (persisted answers, queued writes) ends with the
  // session, whichever path ended it: logout, account deletion or a 401.
  runSessionCleanups();
  try {
    window.localStorage.removeItem(TOKEN_KEY);
  } catch {
    // Nothing persisted to remove.
  }
  // The server-set flag is not HttpOnly precisely so this line can drop it.
  document.cookie = `${AUTH_COOKIE}=; path=/; max-age=0; SameSite=Lax`;
  // The resume marker belongs to the session that just ended; leaving it would
  // trap the next sign-in on this device in a stranger's half-done onboarding.
  document.cookie = `${ONBOARDING_COOKIE}=; path=/; max-age=0; SameSite=Lax`;
  // Guarded screens are still mounted at this point; tell them to leave.
  window.dispatchEvent(new Event(SESSION_CLEARED_EVENT));
}

/**
 * Is the middleware's auth flag present? The flag is host-scoped while the token
 * is origin-scoped in `localStorage`, so the two can diverge — most visibly when
 * the site moved from `http://` to `https://`, which kept every visitor's cookie
 * but hid their token behind a new origin. `SessionGuard` reconciles the two.
 */
export function hasAuthCookie(): boolean {
  if (!isBrowser()) return false;
  return document.cookie
    .split(';')
    .some((entry) => entry.trim().startsWith(`${AUTH_COOKIE}=1`));
}

export function isAuthenticated(): boolean {
  return getAuthToken() !== null;
}
