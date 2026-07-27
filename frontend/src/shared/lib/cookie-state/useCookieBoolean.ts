'use client';

import { useCallback, useEffect, useState } from 'react';

import { readCookie, writeCookie } from './cookie';

/**
 * A boolean UI preference that survives reloads, stored in a cookie.
 *
 * The cookie is read **after mount**, not in the state initializer: these routes
 * are prerendered, so reading `document.cookie` during the first render would
 * make the client pass differ from the server HTML and break hydration
 * (same reasoning as `useMounted`).
 */
export function useCookieBoolean(
  name: string,
  defaultValue: boolean,
): [boolean, (next: boolean | ((prev: boolean) => boolean)) => void] {
  const [value, setValue] = useState(defaultValue);

  useEffect(() => {
    const stored = readCookie(name);
    if (stored === '1' || stored === '0') setValue(stored === '1');
  }, [name]);

  const set = useCallback(
    (next: boolean | ((prev: boolean) => boolean)) => {
      setValue(prev => {
        const resolved = typeof next === 'function' ? next(prev) : next;
        writeCookie(name, resolved ? '1' : '0');
        return resolved;
      });
    },
    [name],
  );

  return [value, set];
}
