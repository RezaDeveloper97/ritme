'use client';

import { useEffect } from 'react';

/**
 * Kills the last zoom gestures iOS Safari still honours even with
 * `user-scalable=no` and `touch-action: pan-x pan-y`:
 * pinch (`gesturestart`) and double-tap-to-zoom.
 *
 * Everything else (native app feel: no selection, no callout, no tap flash)
 * is handled in globals.css.
 */
export function NoZoom() {
  useEffect(() => {
    const blockGesture = (e: Event) => e.preventDefault();

    let lastTouchEnd = 0;
    const blockDoubleTap = (e: TouchEvent) => {
      const now = Date.now();
      if (now - lastTouchEnd <= 300) e.preventDefault();
      lastTouchEnd = now;
    };

    document.addEventListener('gesturestart', blockGesture);
    document.addEventListener('gesturechange', blockGesture);
    document.addEventListener('gestureend', blockGesture);
    document.addEventListener('touchend', blockDoubleTap, { passive: false });

    return () => {
      document.removeEventListener('gesturestart', blockGesture);
      document.removeEventListener('gesturechange', blockGesture);
      document.removeEventListener('gestureend', blockGesture);
      document.removeEventListener('touchend', blockDoubleTap);
    };
  }, []);

  return null;
}
