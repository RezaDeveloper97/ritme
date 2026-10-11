'use client';

import { useCallback, useEffect, useState } from 'react';

/*
 * The user's city choice lives on the device only (a plain preference, bloom decision B-N10-05): localStorage can
 * be missing or throw (private window, blocked data), so every access is wrapped and the screen works without it.
 */
const KEY = 'ritme.cityServices.city';

export function readCityChoice(): number | null {
  try {
    const n = Number(window.localStorage.getItem(KEY));
    return Number.isInteger(n) && n > 0 ? n : null;
  } catch {
    return null;
  }
}

function writeCityChoice(id: number | null): void {
  try {
    if (id === null) window.localStorage.removeItem(KEY);
    else window.localStorage.setItem(KEY, String(id));
  } catch {
    /* storage unavailable: the choice just lasts for this visit */
  }
}

/** The remembered city id (null = all cities). Read after mount so server and client render the same markup. */
export function useCityChoice(): [number | null, (id: number | null) => void, boolean] {
  const [city, setCity] = useState<number | null>(null);
  const [ready, setReady] = useState(false);
  useEffect(() => {
    setCity(readCityChoice());
    setReady(true);
  }, []);
  const choose = useCallback((id: number | null) => {
    setCity(id);
    writeCityChoice(id);
  }, []);
  return [city, choose, ready];
}
