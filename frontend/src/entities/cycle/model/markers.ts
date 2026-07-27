import type { CycleDayMarker } from './types';

/**
 * The one palette for cycle day markers, shared by every calendar surface (the
 * full calendar screen and the home mini calendar) so a period day is never
 * pink in one place and something else in another. Follicular/luteal days carry
 * no marker and read as neutral.
 */
export const cycleMarkerStyle: Record<CycleDayMarker, { bg: string; color: string }> = {
  period: { bg: 'var(--pink-bg)', color: 'var(--brand)' },
  fertile: { bg: 'var(--amber-soft)', color: 'var(--amber)' },
  ovulation: { bg: 'var(--green-tint)', color: 'var(--green)' },
  pms: { bg: 'var(--violet-soft)', color: 'var(--violet)' },
};
