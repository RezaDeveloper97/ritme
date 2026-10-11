'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

export type SaveState = 'idle' | 'saving' | 'saved' | 'error';

/**
 * Debounced auto-save of one step form. Saves run one after another (never out of order); `flush` saves right away
 * and resolves `false` when the save failed, so «continue» can stay on the step. Nothing is kept on the device.
 */
export function useAutosave<T>(value: T, save: (v: T) => Promise<unknown>, delay = 900) {
  const [state, setState] = useState<SaveState>('idle');
  const latest = useRef(value);
  const saved = useRef(JSON.stringify(value));
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const chain = useRef<Promise<boolean>>(Promise.resolve(true));
  const saveRef = useRef(save);
  saveRef.current = save;
  latest.current = value;

  const run = useCallback((): Promise<boolean> => {
    const snapshot = latest.current;
    const key = JSON.stringify(snapshot);
    if (key === saved.current) return chain.current;
    chain.current = chain.current.then(async () => {
      setState('saving');
      try {
        await saveRef.current(snapshot);
        saved.current = key;
        setState('saved');
        return true;
      } catch {
        setState('error');
        return false;
      }
    });
    return chain.current;
  }, []);

  useEffect(() => {
    if (JSON.stringify(value) === saved.current) return;
    timer.current = setTimeout(() => void run(), delay);
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, [value, delay, run]);

  // Leaving the step (back arrow, route change) still saves what was typed.
  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
      void run();
    },
    [run],
  );

  const flush = useCallback(async (): Promise<boolean> => {
    if (timer.current) clearTimeout(timer.current);
    return run();
  }, [run]);

  return { state, flush };
}
