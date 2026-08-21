'use client';

import { createContext, useContext, type ReactNode } from 'react';

/**
 * The active locale's text direction, made available to Client Components.
 *
 * Direction is a property of the language row in the backend, not something
 * derivable from the locale code (CLAUDE.md §6), so it cannot be looked up in a
 * static map on the client. The server resolves it once per request and passes
 * it down here; components that need it for layout math read `useDirection()`.
 *
 * Prefer CSS logical properties (`ms-`, `pe-`, `text-start`) — reach for this
 * only where the direction has to enter JavaScript, e.g. swipe/transform math.
 */
const DirectionContext = createContext<'rtl' | 'ltr'>('rtl');

export function DirectionProvider({
  direction,
  children,
}: {
  direction: 'rtl' | 'ltr';
  children: ReactNode;
}) {
  return (
    <DirectionContext.Provider value={direction}>{children}</DirectionContext.Provider>
  );
}

export function useDirection(): 'rtl' | 'ltr' {
  return useContext(DirectionContext);
}
