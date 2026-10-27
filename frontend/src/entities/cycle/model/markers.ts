import type { CycleDayMarker, MarkerIntensity } from './types';

/**
 * The one palette for cycle day markers, shared by every calendar surface (the
 * full calendar screen and the home mini calendar) so a period day is never
 * pink in one place and something else in another. Follicular/luteal days carry
 * no marker and read as neutral. `bg` is the medium intensity step — day cells
 * pick their exact step from {@link cycleMarkerBg}.
 */
export const cycleMarkerStyle: Record<CycleDayMarker, { bg: string; color: string }> = {
  // Period stays red — bleeding days read as red to users, and that convention
  // outranks the brand palette. It is the one sanctioned red (§10.2).
  period: { bg: 'var(--period-soft)', color: 'var(--period)' },
  fertile: { bg: 'var(--amber-soft)', color: 'var(--amber)' },
  // Ovulation is algorithmic data — it carries the turquoise data accent (§10.2).
  ovulation: { bg: 'var(--data-soft)', color: 'var(--data-deep)' },
  pms: { bg: 'var(--violet-soft)', color: 'var(--violet)' },
};

/**
 * Cell background per marker at each intensity step.
 */
export const cycleMarkerBg: Record<CycleDayMarker, Record<MarkerIntensity, string>> = {
  period: {
    faint: 'var(--period-soft-faint)',
    medium: 'var(--period-soft)',
    strong: 'var(--period-soft-strong)',
  },
  fertile: {
    faint: 'var(--amber-soft-faint)',
    medium: 'var(--amber-soft)',
    strong: 'var(--amber-soft-strong)',
  },
  ovulation: {
    faint: 'var(--data-soft-faint)',
    medium: 'var(--data-soft)',
    strong: 'var(--data-soft-strong)',
  },
  pms: {
    faint: 'var(--violet-soft-faint)',
    medium: 'var(--violet-soft)',
    strong: 'var(--violet-soft-strong)',
  },
};
