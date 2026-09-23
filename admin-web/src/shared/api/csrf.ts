import { CSRF_COOKIE_NAMES } from '@/shared/config';

/**
 * The CSRF token echoed in X-CSRF-Token on every mutating request. The source
 * of truth is the value from the last login / GET /auth/me; the readable CSRF
 * cookie is the fallback after a full page load.
 */
let token: string | null = null;

export function setCsrfToken(value: string | null): void {
  token = value;
}

export function readCookie(cookie: string, names: readonly string[]): string | null {
  for (const part of cookie.split(';')) {
    const eq = part.indexOf('=');
    if (eq < 0) continue;
    const name = part.slice(0, eq).trim();
    if (names.includes(name)) return decodeURIComponent(part.slice(eq + 1).trim());
  }
  return null;
}

export function getCsrfToken(): string | null {
  if (token) return token;
  if (typeof document === 'undefined') return null;
  return readCookie(document.cookie, CSRF_COOKIE_NAMES);
}
