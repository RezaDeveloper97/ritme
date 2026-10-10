'use client';

import { useEffect, useRef } from 'react';

/**
 * Runs `onHide` on `pagehide` (tab closed, navigated away, put in the bfcache). The public record pages (CB-REC-05)
 * use it to drop what they show, so a shared clinic PC's back button or bfcache restore never brings a patient's
 * record or emergency card back on screen.
 */
export function usePageHide(onHide: () => void): void {
  const ref = useRef(onHide);
  ref.current = onHide;
  useEffect(() => {
    const handler = () => ref.current();
    window.addEventListener('pagehide', handler);
    return () => window.removeEventListener('pagehide', handler);
  }, []);
}

/**
 * Swaps the address bar (and this history entry) to `path` without a navigation, so a token in the URL does not stay
 * in the visible location / this entry after it has been used. No-op when already there.
 */
export function replaceUrlPath(path: string): void {
  if (typeof window === 'undefined' || window.location.pathname === path) return;
  try {
    window.history.replaceState(window.history.state, '', path);
  } catch {
    // a sandboxed frame may refuse; the page still works
  }
}

/**
 * Seconds to wait after a 429: the body's `retry_after` (D-71 breaker), else the `Retry-After` header (seconds), else
 * `fallback`. Clamped to 1 s … 1 h.
 */
export function retryAfterSeconds(body: unknown, headers: Record<string, string> | undefined, fallback = 60): number {
  const fromBody = typeof body === 'object' && body !== null ? Number((body as { retry_after?: unknown }).retry_after) : NaN;
  const fromHeader = headers?.['retry-after'] !== undefined ? Number(headers['retry-after']) : NaN;
  const raw = Number.isFinite(fromBody) && fromBody > 0 ? fromBody : Number.isFinite(fromHeader) && fromHeader > 0 ? fromHeader : fallback;
  return Math.min(3600, Math.max(1, Math.ceil(raw)));
}
