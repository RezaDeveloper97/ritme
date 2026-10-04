'use client';

import { useEffect, useState } from 'react';

/**
 * Client time, re-read once a second while `running` — enough for `m:ss`
 * timers. The value itself is never trusted as elapsed time: screens derive
 * durations from server timestamps (see `timing.ts`), so a throttled
 * background tab catches up on its next tick.
 */
export function useNow(running: boolean): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!running) return;
    setNow(Date.now());
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    const wake = () => setNow(Date.now());
    document.addEventListener('visibilitychange', wake);
    return () => {
      window.clearInterval(id);
      document.removeEventListener('visibilitychange', wake);
    };
  }, [running]);
  return now;
}
