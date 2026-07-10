/**
 * Given a horizontal drag on the intro carousel, decide which slide to settle
 * on. Pure so the swipe threshold can be unit-tested without a DOM.
 *
 * The track is laid out physically (LTR) so this math is direction-independent:
 * `deltaX = currentX - startX` in px. Dragging left (negative) advances to the
 * next slide; dragging right (positive) goes back. The move must clear both an
 * absolute floor and a fraction of the slide width, so small jitters never flip
 * the slide. Mirrors the banner-slideshow helper, kept slice-local rather than
 * shared until a third carousel earns the abstraction (CLAUDE.md §12).
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
