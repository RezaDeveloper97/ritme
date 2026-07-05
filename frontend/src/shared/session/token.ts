'use client';

/**
 * Session-token plumbing. This is domain-agnostic session state (not health
 * data), so it lives in `shared` where the HTTP client can read it — `shared`
 * may not import from `entities`/`features`, so the token can't live upward
 * (CLAUDE.md §3, §8).
 *
 * The JWT is kept in `localStorage` for attaching the bearer header. A separate,
 * value-less cookie flag lets the (edge) middleware gate routes without ever
 * putting the token itself in a cookie or URL (§11 — nothing sensitive leaves
 * where it isn't needed).
 */

import { AUTH_COOKIE } from './cookie';

const TOKEN_KEY = 'ritme_token';

// In-memory mirror so reads work before/without localStorage and stay fast.
let inMemoryToken: string | null = null;

const isBrowser = (): boolean => typeof window !== 'undefined';

export function getAuthToken(): string | null {
  if (inMemoryToken) return inMemoryToken;
  if (!isBrowser()) return null;
  inMemoryToken = window.localStorage.getItem(TOKEN_KEY);
  return inMemoryToken;
}

export function setAuthToken(token: string): void {
  inMemoryToken = token;
  if (!isBrowser()) return;
  window.localStorage.setItem(TOKEN_KEY, token);
  // Flag only — never the token value.
  document.cookie = `${AUTH_COOKIE}=1; path=/; max-age=31536000; SameSite=Lax`;
}

export function clearAuthToken(): void {
  inMemoryToken = null;
  if (!isBrowser()) return;
  window.localStorage.removeItem(TOKEN_KEY);
  document.cookie = `${AUTH_COOKIE}=; path=/; max-age=0; SameSite=Lax`;
}

export function isAuthenticated(): boolean {
  return getAuthToken() !== null;
}
