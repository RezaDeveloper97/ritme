import { useId } from 'react';

import type { Tone } from '@/shared/ui';

import { stackSegments } from '../lib/geometry';
import { ChartFigure } from './ChartFigure';

export interface PhasePart {
  key: string;
  /** Legend text, e.g. «پریود ۵». */
  label: string;
  days: number;
  tone: Tone | 'track';
}

const W = 320;
const H = 18;

/**
 * The typical cycle as one stacked bar (An_Cycle «فازهای سیکل معمول تو»):
 * period → follicular → fertile → luteal, day 1 at the left, with a legend.
 */
export function PhaseBar({ label, parts }: { label: string; parts: readonly PhasePart[] }) {
  const clip = `axc-phase-${useId().replace(/:/g, '')}`;
  const shown = parts.filter((p) => p.days > 0);
  const segs = stackSegments(
    shown.map((p) => p.days),
    W,
  );
  return (
    <div className="axc-phase">
      <ChartFigure label={label}>
        <svg viewBox={`0 0 ${W} ${H}`} width="100%" className="axc-svg" aria-hidden focusable="false">
          <defs>
            <clipPath id={clip}>
              <rect x={0} y={0} width={W} height={H} rx={H / 2} />
            </clipPath>
          </defs>
          <g clipPath={`url(#${clip})`}>
            {shown.map((p, i) => (
              <rect
                key={p.key}
                className={p.tone === 'track' ? 'axc-phase-track' : `axc-phase-seg nb-tone-${p.tone}`}
                x={segs[i].x}
                y={0}
                width={Math.max(0, segs[i].w - (i < shown.length - 1 ? 2 : 0))}
                height={H}
              />
            ))}
          </g>
        </svg>
      </ChartFigure>
      <ul className="axc-legend" aria-hidden>
        {shown.map((p) => (
          <li key={p.key} className={p.tone === 'track' ? 'is-track' : `nb-tone-${p.tone}`}>
            <span className="axc-legend-dot" />
            <span className="axc-legend-text">{p.label}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
