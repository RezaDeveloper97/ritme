'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

/** `mm:ss` (or `h:mm:ss` from an hour up) of a non-negative duration. */
export function formatClock(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const pad = (n: number) => String(n).padStart(2, '0');
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${pad(m)}:${pad(s)}`;
}

/**
 * Ring fill (0–1) of a timer. `countdown` empties as time runs out; `elapsed`
 * fills towards `totalMs` and, when the timer has no cap, laps once a minute.
 */
export function timerProgress(elapsedMs: number, totalMs: number | undefined, mode: 'elapsed' | 'countdown'): number {
  if (mode === 'countdown') {
    if (!totalMs) return 0;
    return Math.max(0, Math.min(1, 1 - elapsedMs / totalMs));
  }
  const lap = totalMs ?? 60_000;
  if (totalMs) return Math.max(0, Math.min(1, elapsedMs / lap));
  return (elapsedMs % lap) / lap;
}

interface UseTimerOptions {
  running: boolean;
  /** Stops at this duration and fires `onComplete` (countdown length, Kegel hold). */
  totalMs?: number;
  onComplete?: () => void;
}

/**
 * Wall-clock stopwatch: elapsed time comes from `Date.now()` deltas, so a
 * throttled background tab never drifts. Ticks 4×/s, or once a second under
 * `prefers-reduced-motion` (the ring then steps instead of gliding).
 */
export function useTimer({ running, totalMs, onComplete }: UseTimerOptions) {
  const [elapsedMs, setElapsedMs] = useState(0);
  const baseRef = useRef(0);
  const startRef = useRef<number | null>(null);
  const completeRef = useRef(onComplete);
  useEffect(() => {
    completeRef.current = onComplete;
  }, [onComplete]);

  useEffect(() => {
    if (!running) {
      if (startRef.current !== null) {
        baseRef.current += Date.now() - startRef.current;
        startRef.current = null;
      }
      return;
    }
    startRef.current = Date.now();
    const reduce = typeof window !== 'undefined' && window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
    const tick = () => {
      const now = baseRef.current + Date.now() - (startRef.current ?? Date.now());
      if (totalMs !== undefined && now >= totalMs) {
        setElapsedMs(totalMs);
        baseRef.current = totalMs;
        startRef.current = null;
        window.clearInterval(id);
        completeRef.current?.();
        return;
      }
      setElapsedMs(now);
    };
    const id = window.setInterval(tick, reduce ? 1000 : 250);
    return () => window.clearInterval(id);
  }, [running, totalMs]);

  const reset = useCallback(() => {
    baseRef.current = 0;
    startRef.current = startRef.current === null ? null : Date.now();
    setElapsedMs(0);
  }, []);

  return { elapsedMs, reset };
}
