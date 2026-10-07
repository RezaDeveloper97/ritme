'use client';

import { useCallback, useEffect, useState } from 'react';

/** Seconds left until `start(n)` runs out (0 = done). */
export function useCountdown(): { left: number; start: (seconds: number) => void } {
  const [until, setUntil] = useState<number | null>(null);
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (until === null) return;
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [until]);
  const start = useCallback((seconds: number) => {
    const t = Date.now();
    setNow(t);
    setUntil(t + seconds * 1000);
  }, []);
  const left = until === null ? 0 : Math.max(0, Math.ceil((until - now) / 1000));
  return { left, start };
}
