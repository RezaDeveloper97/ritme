'use client';

import { useCallback, useEffect, useState } from 'react';

/*
 * CB-DIR-06: «پیشنهاد بر اساس سن کودک» is switchable (Dir_Home note). The choice is a plain device preference: it only
 * decides whether the child's age in months is sent with the search — nothing about the child is stored here.
 */
const KEY = 'ritme.cityServices.ageSuggest';

function read(): boolean {
  try {
    return window.localStorage.getItem(KEY) !== '0';
  } catch {
    return true;
  }
}

/** [on, setOn, ready] — on by default; read after mount so server and client render the same markup. */
export function useAgeSuggest(): [boolean, (on: boolean) => void, boolean] {
  const [on, setOn] = useState(true);
  const [ready, setReady] = useState(false);
  useEffect(() => {
    setOn(read());
    setReady(true);
  }, []);
  const choose = useCallback((next: boolean) => {
    setOn(next);
    try {
      window.localStorage.setItem(KEY, next ? '1' : '0');
    } catch {
      /* storage unavailable: lasts for this visit */
    }
  }, []);
  return [on, choose, ready];
}
