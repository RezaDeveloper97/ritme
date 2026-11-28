'use client';

import { useEffect, useRef } from 'react';

import { getBackStack } from './client';

/**
 * While `open`, the overlay owns one history entry, so the system Back closes
 * it (and only it) instead of leaving the screen.
 *
 * `onClose` runs when Back pops the entry; it should just flip `open` off. Any
 * other way of closing (×, swipe, backdrop, a save) flips `open` off directly,
 * and the hook pops the entry itself — never both, so one close can never take
 * the screen with it.
 */
export function useBackLayer(open: boolean, onClose: () => void): void {
  const latest = useRef(onClose);
  latest.current = onClose;

  useEffect(() => {
    if (!open) return;
    const layer = getBackStack().push(() => latest.current());
    return () => layer.release();
  }, [open]);
}
