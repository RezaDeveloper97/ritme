import clsx from 'clsx';

import { type IllustrationProps, svgA11y } from './svg-a11y';

/**
 * Setup welcome drawing (Setup artboard, step 0) — a curled baby on a soft
 * disc, cradled by a brand arc. Every fill is a `.pg2-illu-*` class, so it
 * flips with the theme.
 */
export function WelcomePregnancy({ size = 220, label, className }: Omit<IllustrationProps, 'onHero'>) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 230 230"
      className={clsx('pg2-illu', className)}
      {...svgA11y(label)}
    >
      <circle className="pg2-illu-disc" cx="115" cy="115" r="110" />
      <circle className="pg2-illu-ring" cx="115" cy="115" r="110" strokeWidth="2" strokeDasharray="6 8" />
      <path
        className="pg2-illu-body"
        d="M60,150 C70,110 100,95 130,105 C160,115 165,150 145,170 C120,190 80,185 60,150z"
      />
      <circle className="pg2-illu-baby" cx="128" cy="112" r="26" />
      <circle className="pg2-illu-shine" cx="136" cy="104" r="8" />
      <path className="pg2-illu-arc" d="M70,175 Q115,205 160,175" strokeWidth="5" strokeLinecap="round" />
      <g className="pg2-illu-spark">
        <circle cx="40" cy="70" r="3" />
        <circle cx="195" cy="60" r="2.5" />
        <circle cx="190" cy="185" r="3" />
      </g>
    </svg>
  );
}
