/**
 * The bearer token from POST /auth/verify-otp. It lives in localStorage on the
 * instructor host only (its own origin — never shared with the user app), the
 * same model as frontend/ (shared/session/token.ts). The CSP in next.config.ts
 * allows no third-party script, which is what keeps this storage private.
 */
export const TOKEN_KEY = 'ritme_instructor_token';

/** Fired on the window when the session is dropped (logout, 401 from the API). */
export const SESSION_CLEARED_EVENT = 'ritme-instructor:session-cleared';

function storage(): Storage | null {
  if (typeof window === 'undefined') return null;
  try {
    return window.localStorage;
  } catch {
    return null; // storage blocked: the session lasts as long as the tab (memory below)
  }
}

let memoryToken: string | null = null;

export function getAuthToken(): string | null {
  const store = storage();
  if (!store) return memoryToken;
  try {
    return store.getItem(TOKEN_KEY) ?? memoryToken;
  } catch {
    return memoryToken;
  }
}

export function setAuthToken(token: string): void {
  memoryToken = token;
  try {
    storage()?.setItem(TOKEN_KEY, token);
  } catch {
    // private mode / quota: keep the in-memory copy
  }
}

export function clearAuthToken(): void {
  memoryToken = null;
  try {
    storage()?.removeItem(TOKEN_KEY);
  } catch {
    // nothing to clear
  }
  if (typeof window !== 'undefined') window.dispatchEvent(new Event(SESSION_CLEARED_EVENT));
}

export function hasAuthToken(): boolean {
  return Boolean(getAuthToken());
}
