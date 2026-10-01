'use client';

import { useEffect, useState } from 'react';

/** `value`, settled for `delay` ms — so typing doesn't fire a request per keystroke. */
export function useDebouncedValue<T>(value: T, delay: number): T {
  const [settled, setSettled] = useState(value);
  useEffect(() => {
    const id = window.setTimeout(() => setSettled(value), delay);
    return () => window.clearTimeout(id);
  }, [value, delay]);
  return settled;
}
