'use client';

import { useEffect, useState } from 'react';

/**
 * Wall-clock `Date.now()` that re-renders once a second while `running` — the
 * live part of a server-side session. Derived from the clock on every tick,
 * so a throttled background tab never drifts.
 */
export function useNow(running: boolean): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!running) return;
    setNow(Date.now());
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [running]);
  return now;
}
