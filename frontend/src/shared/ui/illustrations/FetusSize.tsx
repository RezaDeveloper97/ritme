import clsx from 'clsx';
import type { ReactNode } from 'react';

import { type FetusDrawing, type FruitSize, fetusDrawing } from './fetus-keys';
import { type IllustrationProps, svgA11y } from './svg-a11y';

/*
 * «تقریباً هم‌اندازهٔ یک …» — the Week hero drawing (Week artboard) and the
 * Today carousel art. `illustrationKey` comes from the admin-authored week
 * details; see `fetus-keys.ts` for the list and its shape/tint table.
 *
 * Geometry lives in a 176×176 box centred on (88, 88); the halo is r = 84.
 * Colours are `.pg2-illu-*` / `.pg2-fruit-*` classes — never fills — so the
 * drawing flips with the theme.
 */

const C = 88;
const RADIUS: Record<FruitSize, number> = { s: 16, m: 34, l: 50 };

/** Stem + leaf sitting on top of a body whose top edge is at `top`. */
function Stem({ top }: { top: number }) {
  return (
    <>
      <path className="pg2-illu-stem" d={`M${C - 2},${top + 2} q4,-12 14,-10`} strokeWidth="4" strokeLinecap="round" />
      <path className="pg2-illu-leaf" d={`M${C},${top + 4} q-14,-6 -20,4 q12,4 20,-4z`} />
    </>
  );
}

/** The artboard's raspberry drupelets, re-centred and scaled to `r`. */
const DRUPELETS: readonly [number, number, number][] = [
  [-16, -26, 12], [8, -28, 12], [-28, -4, 12], [-4, -6, 12], [20, -8, 12],
  [-22, 18, 12], [2, 16, 12], [24, 14, 11], [-8, 38, 11], [14, 36, 10],
];

function shape(d: FetusDrawing): ReactNode {
  const r = RADIUS[d.size];
  const tint = `pg2-fruit-${d.tint}`;
  switch (d.shape) {
    case 'seed':
      return (
        <g transform={`rotate(-20 ${C} ${C})`}>
          <ellipse className={tint} cx={C} cy={C} rx={r * 0.7} ry={r} />
          <ellipse className="pg2-illu-gloss" cx={C - r * 0.2} cy={C - r * 0.4} rx={r * 0.18} ry={r * 0.3} />
        </g>
      );
    case 'round':
      return (
        <>
          <circle className={tint} cx={C} cy={C + 4} r={r} />
          <circle className="pg2-illu-gloss" cx={C - r * 0.4} cy={C + 4 - r * 0.4} r={r * 0.18} />
          <Stem top={C + 4 - r} />
        </>
      );
    case 'cluster': {
      const k = r / 40;
      return (
        <>
          <g className={tint}>
            {DRUPELETS.map(([x, y, rr]) => (
              <circle key={`${x},${y}`} cx={C + x * k} cy={C + y * k} r={rr * k} />
            ))}
          </g>
          <g className="pg2-illu-gloss">
            {DRUPELETS.slice(0, 6).map(([x, y]) => (
              <circle key={`g${x},${y}`} cx={C + (x - 4) * k} cy={C + (y - 4) * k} r={3 * k} />
            ))}
          </g>
          <Stem top={C - 38 * k} />
        </>
      );
    }
    case 'drop': {
      const top = C - r * 1.1;
      return (
        <>
          <path
            className={tint}
            d={`M${C},${top} C${C + r * 0.9},${C - r * 0.2} ${C + r},${C + r * 0.9} ${C},${C + r} C${C - r},${C + r * 0.9} ${C - r * 0.9},${C - r * 0.2} ${C},${top}z`}
          />
          <ellipse className="pg2-illu-gloss" cx={C - r * 0.35} cy={C + r * 0.1} rx={r * 0.14} ry={r * 0.28} />
          <Stem top={top} />
        </>
      );
    }
    case 'long':
      return (
        <g transform={`rotate(35 ${C} ${C})`}>
          <ellipse className={tint} cx={C} cy={C} rx={r * 0.45} ry={r * 1.3} />
          <ellipse className="pg2-illu-gloss" cx={C - r * 0.15} cy={C - r * 0.4} rx={r * 0.08} ry={r * 0.5} />
          <Stem top={C - r * 1.3} />
        </g>
      );
    case 'leafy': {
      const petals = [0, 60, 120, 180, 240, 300];
      return (
        <>
          <g className={tint}>
            {petals.map((a) => (
              <circle
                key={a}
                cx={C + Math.cos((a * Math.PI) / 180) * r * 0.45}
                cy={C + Math.sin((a * Math.PI) / 180) * r * 0.45}
                r={r * 0.6}
              />
            ))}
          </g>
          <circle className="pg2-illu-gloss" cx={C} cy={C} r={r * 0.35} />
        </>
      );
    }
    case 'striped':
      return (
        <>
          <circle className={tint} cx={C} cy={C + 4} r={r} />
          <g className="pg2-illu-gloss">
            <ellipse cx={C - r * 0.5} cy={C + 4} rx={r * 0.08} ry={r * 0.8} />
            <ellipse cx={C} cy={C + 4} rx={r * 0.08} ry={r * 0.95} />
            <ellipse cx={C + r * 0.5} cy={C + 4} rx={r * 0.08} ry={r * 0.8} />
          </g>
          <Stem top={C + 4 - r} />
        </>
      );
    case 'spiky':
      return (
        <>
          <ellipse className={tint} cx={C} cy={C + 12} rx={r * 0.7} ry={r * 0.9} />
          <g className="pg2-illu-gloss">
            {[-0.4, 0, 0.4].map((dx) => (
              <ellipse key={dx} cx={C + dx * r} cy={C + 12} rx={r * 0.06} ry={r * 0.75} />
            ))}
          </g>
          <g className="pg2-illu-leaf">
            <path d={`M${C},${C + 12 - r * 0.8} l-${r * 0.35},-${r * 0.55} l${r * 0.2},${r * 0.1} z`} />
            <path d={`M${C},${C + 12 - r * 0.8} l0,-${r * 0.7} l${r * 0.12},${r * 0.2} z`} />
            <path d={`M${C},${C + 12 - r * 0.8} l${r * 0.35},-${r * 0.55} l-${r * 0.2},${r * 0.1} z`} />
          </g>
        </>
      );
    case 'fetus':
    default:
      return (
        <g transform={`translate(${C - 75 * 1.05} ${C - 75 * 1.05}) scale(1.05)`}>
          <path className="pg2-illu-baby" d="M58,98 C52,70 70,50 90,56 C108,62 110,86 96,98 C86,106 70,108 58,98z" />
          <circle className="pg2-illu-head" cx="88" cy="66" r="12" />
          <circle className="pg2-illu-shine" cx="92" cy="63" r="3" />
        </g>
      );
  }
}

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
      {shape(fetusDrawing(illustrationKey))}
    </svg>
  );
}
