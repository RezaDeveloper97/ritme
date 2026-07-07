/**
 * Given a horizontal drag, decide which slide to settle on. Pure so the swipe
 * threshold logic can be unit-tested without a DOM.
 *
 * `deltaX` is `currentX - startX` in px: dragging left (negative) advances to
 * the next slide, dragging right (positive) goes to the previous one. The move
 * must clear both an absolute floor and a fraction of the slide width to count,
 * so small jitters never flip the slide.
 */
export function resolveSwipeIndex({
  deltaX,
  width,
  index,
  count,
  threshold = 0.15,
  minPx = 40,
}: {
  deltaX: number;
  width: number;
  index: number;
  count: number;
  threshold?: number;
  minPx?: number;
}): number {
  if (count <= 1 || width <= 0) return index;

  const trigger = Math.max(minPx, width * threshold);

  if (deltaX <= -trigger) return Math.min(count - 1, index + 1);
  if (deltaX >= trigger) return Math.max(0, index - 1);
  return index;
}

/** Wrap an index into the valid range, advancing by `step` (for auto-play). */
export function wrapIndex(index: number, count: number, step = 1): number {
  if (count <= 0) return 0;
  return (((index + step) % count) + count) % count;
}
