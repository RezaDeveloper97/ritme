import type { FertilityLevel } from './types';

export interface FertilityBadgeStyle {
  bg: string;
  fg: string;
}

/**
 * Brand-token colour per calendar fertility level (spec §17 read-out). Purely a
 * presentation choice; the level itself is informational, never diagnostic (§11).
 * Foregrounds are darkened to clear WCAG AA (≥4.5:1) on their pale backgrounds.
 */
const FERTILITY_COLOR: Record<FertilityLevel, FertilityBadgeStyle> = {
  very_high: { bg: 'var(--pink-bg)', fg: 'var(--brand-strong)' },
  high: { bg: 'var(--surface-2)', fg: 'var(--brand-strong)' },
  medium: { bg: 'var(--amber-tint)', fg: 'var(--amber-deep)' },
  low: { bg: 'var(--line)', fg: 'var(--ink-3)' },
  very_low: { bg: 'var(--line)', fg: 'var(--ink-3)' },
};

/** Neutral style for a fertility level the backend added that this build doesn't know. */
const FERTILITY_FALLBACK = FERTILITY_COLOR.low;

/** Badge colours for a level, never crashing on a value a newer backend added. */
export function fertilityBadgeStyle(level: FertilityLevel | string | null): FertilityBadgeStyle {
  if (!level) return FERTILITY_FALLBACK;
  return FERTILITY_COLOR[level as FertilityLevel] ?? FERTILITY_FALLBACK;
}
