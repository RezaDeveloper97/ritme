/**
 * Accent tones of the Night & Bloom palette (docs/night-bloom/tokens.md §2).
 * A tone paints tinted surfaces (icon discs, status pills, multi-select chips)
 * from one accent: the class `nb-tone-<tone>` sets `--nb-tone` (the accent) and
 * `--nb-tone-ink` (its AA-safe text colour) and the component mixes the tint.
 */
export type Tone = 'brand' | 'data' | 'warm' | 'period' | 'bloom' | 'success' | 'danger' | 'neutral';

export function toneClass(tone: Tone): string {
  return `nb-tone-${tone}`;
}
