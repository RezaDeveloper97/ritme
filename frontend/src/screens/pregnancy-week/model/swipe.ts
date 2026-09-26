import { clampV2Week } from '@/entities/pregnancy';

/*
 * Swipe between weeks (Week artboard). Pure so it can be unit-tested.
 * "Next" follows reading direction: in RTL the next week comes in from the
 * left, so a finger moving right (dx > 0) advances; in LTR it is the reverse.
 */

/** Minimum horizontal travel (px) before a swipe counts. */
export const SWIPE_MIN_PX = 60;

/**
 * The week a swipe lands on, or `null` when the gesture is not a swipe
 * (too short, mostly vertical, or already at the 1 / 42 edge).
 */
export function swipeTarget(week: number, dx: number, dy: number, dir: 'rtl' | 'ltr'): number | null {
  if (Math.abs(dx) < SWIPE_MIN_PX || Math.abs(dx) <= Math.abs(dy) * 1.5) return null;
  const forward = dir === 'rtl' ? dx > 0 : dx < 0;
  const next = clampV2Week(week + (forward ? 1 : -1));
  return next === week ? null : next;
}
