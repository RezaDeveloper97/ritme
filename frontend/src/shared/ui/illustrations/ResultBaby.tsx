import clsx from 'clsx';

import { type IllustrationProps, svgA11y } from './svg-a11y';

/**
 * The fetus-in-a-disc drawing of the Setup result (step 3/3) and the Today
 * hero (Main artboard, `onHero`). Fills are `.pg2-illu-*` classes.
 */
export function ResultBaby({ size = 150, label, className, onHero = false }: IllustrationProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 150 150"
      className={clsx('pg2-illu', onHero && 'on-hero', className)}
      {...svgA11y(label)}
    >
      <circle className="pg2-illu-plate" cx="75" cy="75" r="70" />
      <circle className="pg2-illu-body" cx="75" cy="75" r="56" />
      <circle
        className={onHero ? 'pg2-illu-ring-pk' : 'pg2-illu-ring'}
        cx="75"
        cy="75"
        r="56"
        strokeWidth="3"
        strokeDasharray="4 7"
      />
      <path className="pg2-illu-baby" d="M58,98 C52,70 70,50 90,56 C108,62 110,86 96,98 C86,106 70,108 58,98z" />
      <circle className="pg2-illu-head" cx="88" cy="66" r="12" />
      <circle className="pg2-illu-shine" cx="92" cy="63" r="3" />
    </svg>
  );
}
