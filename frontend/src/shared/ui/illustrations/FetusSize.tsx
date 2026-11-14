import clsx from 'clsx';

import { isFetusIllustrationKey } from './fetus-keys';
import { type IllustrationProps, svgA11y } from './svg-a11y';

/*
 * «تقریباً هم‌اندازهٔ یک …» — the Week hero art (Week artboard) and the
 * Today carousel art. `illustrationKey` comes from the admin-authored week
 * details; see `fetus-keys.ts` for the list.
 *
 * Each key has a glossy 3D render in `public/art3d/fetus/<key>.webp` (480px,
 * transparent, made with Canva in the Art3d style), drawn over the themed
 * halo + dashed ring. Unknown / missing keys show the neutral `fetus` render.
 */

const C = 88;
/** The 3D art fills most of the halo (r = 84) and leaves the dashed ring visible. */
const ART = 132;

export interface FetusSizeProps extends IllustrationProps {
  /** `pregnancy_week_details.illustration_key`; unknown / null → the neutral fetus. */
  illustrationKey: string | null | undefined;
}

export function FetusSize({ illustrationKey, size = 168, label, className, onHero = false }: FetusSizeProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 176 176"
      className={clsx('pg2-illu', onHero && 'on-hero', className)}
      {...svgA11y(label)}
    >
      <circle className="pg2-illu-halo" cx={C} cy={C} r="84" />
      <circle className="pg2-illu-ring-pk" cx={C} cy={C} r="84" strokeWidth="2" strokeDasharray="5 8" />
      <image
        href={`/art3d/fetus/${isFetusIllustrationKey(illustrationKey) ? illustrationKey : 'fetus'}.webp`}
        x={C - ART / 2}
        y={C - ART / 2}
        width={ART}
        height={ART}
        preserveAspectRatio="xMidYMid meet"
      />
    </svg>
  );
}
