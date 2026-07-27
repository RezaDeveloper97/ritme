'use client';

import { useEffect, useRef, useState, type ReactNode } from 'react';

interface SheetProps {
  /** Whether the sheet is shown. When false, nothing renders. */
  open: boolean;
  /** Called when the user dismisses (tap outside, grip swipe-down, Escape). */
  onClose: () => void;
  children: ReactNode;
  /** `id` of the sheet's heading, wired to `aria-labelledby` for a11y. */
  labelledBy?: string;
}

// How far the user must drag the sheet down before releasing dismisses it.
const DISMISS_THRESHOLD_PX = 80;

/**
 * Generic bottom sheet — dimmed backdrop + a panel that rises from the bottom,
 * with a grip the user can drag down to dismiss. Reuses the global `.sheet`
 * classes so it matches the log screen's sheets. Presentation only; callers own
 * the content and the open/close state (CLAUDE.md §10). RTL-safe by construction
 * (full-width panel, no directional offsets).
 */
export function Sheet({ open, onClose, children, labelledBy }: SheetProps) {
  const startY = useRef<number | null>(null);
  const [dragY, setDragY] = useState(0);

  // Close on Escape while open, without stealing focus handling from callers.
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  if (!open) return null;

  const onTouchStart = (e: React.TouchEvent) => {
    startY.current = e.touches[0]?.clientY ?? null;
  };
  const onTouchMove = (e: React.TouchEvent) => {
    if (startY.current === null) return;
    const delta = (e.touches[0]?.clientY ?? 0) - startY.current;
    // Only track downward drags; ignore upward movement.
    setDragY(Math.max(0, delta));
  };
  const onTouchEnd = () => {
    if (dragY > DISMISS_THRESHOLD_PX) onClose();
    startY.current = null;
    setDragY(0);
  };

  return (
    <div className="sheet-backdrop" onClick={onClose} role="dialog" aria-modal="true" aria-labelledby={labelledBy}>
      <div
        className="sheet"
        onClick={(e) => e.stopPropagation()}
        style={dragY ? { transform: `translateY(${dragY}px)` } : undefined}
      >
        <div
          className="sheet-grip"
          onTouchStart={onTouchStart}
          onTouchMove={onTouchMove}
          onTouchEnd={onTouchEnd}
        />
        {/* Cap the height so a tall sheet scrolls internally instead of
            overflowing past the top of the screen. */}
        <div className="sheet-scroll">{children}</div>
      </div>
    </div>
  );
}
