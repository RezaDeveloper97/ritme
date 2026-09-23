/**
 * Illustrations are decorative unless the caller passes a (translated) label —
 * then they are announced as one image.
 */
export function svgA11y(label: string | undefined) {
  return label
    ? ({ role: 'img', 'aria-label': label } as const)
    : ({ 'aria-hidden': true, focusable: false } as const);
}

export interface IllustrationProps {
  /** Rendered width/height in px. */
  size?: number;
  /** Translated accessible name; omit for a decorative drawing. */
  label?: string;
  className?: string;
  /** Drawn on the Today hero (a saturated fill): the disc turns translucent white. */
  onHero?: boolean;
}
