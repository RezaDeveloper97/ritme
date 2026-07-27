'use client';

/**
 * Tiny document.cookie helpers. Preferences only — never health data (§11),
 * cookies travel with every request and land in server logs.
 */

export function readCookie(name: string): string | null {
  if (typeof document === 'undefined') return null;
  const prefix = `${encodeURIComponent(name)}=`;
  for (const part of document.cookie.split('; ')) {
    if (part.startsWith(prefix)) return decodeURIComponent(part.slice(prefix.length));
  }
  return null;
}

export function writeCookie(name: string, value: string, maxAgeDays = 365): void {
  if (typeof document === 'undefined') return;
  const maxAge = Math.round(maxAgeDays * 24 * 60 * 60);
  document.cookie =
    `${encodeURIComponent(name)}=${encodeURIComponent(value)}; path=/; max-age=${maxAge}; samesite=lax`;
}
