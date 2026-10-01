'use client';

import { type KeyboardEvent, type PointerEvent, useCallback, useRef, useState } from 'react';

/** Pixels from the scroll container's edge where a drag starts scrolling it. */
const EDGE = 64;
const STEP = 12;

export interface ReorderHandle {
  onPointerDown: (e: PointerEvent<HTMLElement>) => void;
  onPointerMove: (e: PointerEvent<HTMLElement>) => void;
  onPointerUp: (e: PointerEvent<HTMLElement>) => void;
  onPointerCancel: (e: PointerEvent<HTMLElement>) => void;
  onKeyDown: (e: KeyboardEvent<HTMLElement>) => void;
}

/**
 * Drag-to-reorder for a vertical list of equal rows, by pointer (mouse, touch, pen — the handle captures
 * the pointer and sets `touch-action: none`) and by keyboard on the same handle (↑ ↓ Home End). The list
 * reorders live as the pointer crosses a neighbour, so there is no ghost element to position; `onDrop`
 * fires once per drag with the final index (for the screen-reader announcement).
 */
export function useReorder(
  count: number,
  onMove: (from: number, to: number, via: 'pointer' | 'key') => void,
  onDrop: (index: number) => void,
) {
  const rows = useRef<(HTMLElement | null)[]>([]);
  const active = useRef<number | null>(null);
  const [dragging, setDragging] = useState<number | null>(null);

  const rowRef = useCallback(
    (index: number) => (el: HTMLElement | null) => {
      rows.current[index] = el;
    },
    [],
  );

  const targetAt = (y: number, from: number): number => {
    const list = rows.current.slice(0, count);
    const first = list[0]?.getBoundingClientRect();
    const last = list[count - 1]?.getBoundingClientRect();
    if (first && y < first.top) return 0;
    if (last && y >= last.bottom) return count - 1;
    for (let i = 0; i < count; i++) {
      const r = list[i]?.getBoundingClientRect();
      if (r && y >= r.top && y < r.bottom) return i;
    }
    return from;
  };

  const autoScroll = (el: HTMLElement, y: number) => {
    const scroller = el.closest<HTMLElement>('.scroll');
    if (!scroller) return;
    const box = scroller.getBoundingClientRect();
    if (y < box.top + EDGE) scroller.scrollTop -= STEP;
    else if (y > box.bottom - EDGE) scroller.scrollTop += STEP;
  };

  const end = (e: PointerEvent<HTMLElement>) => {
    if (active.current === null) return;
    const index = active.current;
    active.current = null;
    setDragging(null);
    if (e.currentTarget.hasPointerCapture(e.pointerId)) e.currentTarget.releasePointerCapture(e.pointerId);
    onDrop(index);
  };

  const handle = (index: number): ReorderHandle => ({
    onPointerDown: (e) => {
      if (e.pointerType === 'mouse' && e.button !== 0) return;
      e.preventDefault();
      e.currentTarget.setPointerCapture(e.pointerId);
      active.current = index;
      setDragging(index);
    },
    onPointerMove: (e) => {
      const from = active.current;
      if (from === null) return;
      autoScroll(e.currentTarget, e.clientY);
      const to = targetAt(e.clientY, from);
      if (to !== from) {
        onMove(from, to, 'pointer');
        active.current = to;
        setDragging(to);
      }
    },
    onPointerUp: end,
    onPointerCancel: end,
    onKeyDown: (e) => {
      const to =
        e.key === 'ArrowUp' ? index - 1
        : e.key === 'ArrowDown' ? index + 1
        : e.key === 'Home' ? 0
        : e.key === 'End' ? count - 1
        : null;
      if (to === null) return;
      e.preventDefault();
      const target = Math.max(0, Math.min(count - 1, to));
      if (target === index) return;
      onMove(index, target, 'key');
    },
  });

  return { rowRef, handle, dragging };
}
