'use client';

import { useEffect, useRef, useState } from 'react';

export interface CountdownParts {
  days: number;
  hours: number;
  minutes: number;
}

/** «۰۵ روز ۱۴ ساعت ۲۲ دقیقه»: whole days, hours and minutes of a non-negative duration (the backend's split). */
export function splitSeconds(seconds: number): CountdownParts {
  const s = Math.max(0, Math.floor(seconds));
  return { days: Math.floor(s / 86_400), hours: Math.floor((s % 86_400) / 3_600), minutes: Math.floor((s % 3_600) / 60) };
}

/** Seconds still left of `secondsLeft` (read at `since`, epoch ms) at `now`. */
export function secondsRemaining(secondsLeft: number, since: number, now: number): number {
  return Math.max(0, secondsLeft - Math.floor(Math.max(0, now - since) / 1000));
}

/**
 * A live countdown that starts from the server's `seconds_left` (B-N2-06), so
 * a wrong device clock can't stretch or end the offer early: only the time
 * elapsed since the response (`since` = the query's `dataUpdatedAt`) is
 * subtracted. Re-renders once a minute (the display has minute resolution);
 * `onExpire` fires once when it reaches zero.
 */
export function useServerCountdown(secondsLeft: number, since: number, onExpire?: () => void): CountdownParts & { seconds: number } {
  const [seconds, setSeconds] = useState(() => secondsRemaining(secondsLeft, since, Date.now()));
  const expireRef = useRef(onExpire);
  useEffect(() => {
    expireRef.current = onExpire;
  }, [onExpire]);

  useEffect(() => {
    let fired = false;
    const tick = () => {
      const left = secondsRemaining(secondsLeft, since, Date.now());
      // Minute resolution: equal minute values bail out of the re-render.
      setSeconds((prev) => (Math.floor(prev / 60) === Math.floor(left / 60) && left > 0 ? prev : left));
      if (left === 0 && !fired) {
        fired = true;
        expireRef.current?.();
      }
    };
    tick();
    const id = window.setInterval(tick, 1000);
    return () => window.clearInterval(id);
  }, [secondsLeft, since]);

  return { seconds, ...splitSeconds(seconds) };
}
