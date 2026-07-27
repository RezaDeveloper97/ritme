'use client';

import { useEffect, useState } from 'react';

/**
 * `false` during SSR and on the very first client render, `true` afterwards.
 *
 * Use it to hold back UI whose output depends on the *current moment* (today's
 * Jalali date, the calendar strip, "is this today?"). Those routes are
 * statically prerendered, so the server's HTML carries the **build date** — the
 * client then renders a different day and React aborts hydration with error
 * #418 (text content mismatch). Rendering that UI only after mount makes the
 * first client pass byte-identical to the server's.
 */
export function useMounted(): boolean {
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  return mounted;
}
