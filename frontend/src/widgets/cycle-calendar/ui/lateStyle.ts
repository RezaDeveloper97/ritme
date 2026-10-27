import type { CSSProperties } from 'react';

import type { BandEdge } from '../model/tones';

/**
 * The late style (a day past the predicted start, nothing logged): a dashed
 * outline in the period colour on a clear ground — deliberately not the soft or
 * solid period fill, so a late day never reads as bleeding. Tokens only, so it
 * follows light and dark.
 */
export function lateDayStyle(edge?: BandEdge | null): CSSProperties {
  const line = '1.5px dashed var(--period)';
  return {
    background: 'transparent',
    color: 'var(--period-deep)',
    borderBlock: line,
    borderInlineStart: edge?.start ? line : undefined,
    borderInlineEnd: edge?.end ? line : undefined,
  };
}
