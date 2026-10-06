import { clsx } from 'clsx';

import type { Tone } from '@/shared/ui';

import { rangeGeometry } from '../model/range';

interface RangeBarProps {
  value: number | null;
  low: number | null;
  high: number | null;
  tone: Tone;
  /** Spoken summary, e.g. «۹، محدوده مرجع ۱۵ تا ۱۵۰» — the bar itself is decorative. */
  label: string;
  size?: 'md' | 'lg';
  className?: string;
}

/**
 * The value-against-range track of nbl_Lab_Result / nbl_Lab_Marker: a pale
 * track, the reference band in turquoise and a square thumb in the state's
 * tone. Plots low → high left to right in both directions, like the charts.
 * Renders nothing without a number and a range.
 */
export function RangeBar({ value, low, high, tone, label, size = 'md', className }: RangeBarProps) {
  const g = rangeGeometry(value, low, high);
  if (!g) return null;
  return (
    <div className={clsx('lab-range', `is-${size}`, `nb-tone-${tone}`, className)} role="img" aria-label={label}>
      <span className="lab-range-band" style={{ insetInlineStart: `${g.bandStart}%`, inlineSize: `${g.bandEnd - g.bandStart}%` }} />
      <span className="lab-range-thumb" style={{ insetInlineStart: `${g.marker}%` }} />
    </div>
  );
}
