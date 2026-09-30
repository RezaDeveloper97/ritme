import clsx from 'clsx';

import { V2_TERM_WEEKS } from '@/entities/pregnancy';

const SIZE = 260;
const RADIUS = 118;
const DOT = 3.6;
const DOT_NOW = 6;

/** One dot per week of the term, clockwise from 12 o'clock (PregFull_Main). */
const DOTS = Array.from({ length: V2_TERM_WEEKS }, (_, i) => {
  const angle = ((-90 + (360 / V2_TERM_WEEKS) * i) * Math.PI) / 180;
  return {
    cx: +(SIZE / 2 + RADIUS * Math.cos(angle)).toFixed(1),
    cy: +(SIZE / 2 + RADIUS * Math.sin(angle)).toFixed(1),
  };
});

interface Props {
  /** Current gestational week (1-based); weeks before it are "walked", it is the big dot. */
  week: number;
  /** Accessible summary, e.g. «هفتهٔ ۸ از ۴۰، ۲۲۱ روز تا زایمان». */
  label: string;
  children: React.ReactNode;
}

/** «حلقه هفته‌های بارداری» — 40 dots with the centre copy laid over it. */
export function WeekRing({ week, label, children }: Props) {
  const current = Math.min(V2_TERM_WEEKS, Math.max(1, week)) - 1;
  return (
    <div className="pgn-ring" role="img" aria-label={label}>
      <svg width={SIZE} height={SIZE} viewBox={`0 0 ${SIZE} ${SIZE}`} aria-hidden focusable="false">
        {DOTS.map((d, i) => (
          <circle
            key={i}
            cx={d.cx}
            cy={d.cy}
            r={i === current ? DOT_NOW : DOT}
            className={clsx('pgn-ring-dot', i < current && 'is-past', i === current && 'is-now')}
          />
        ))}
      </svg>
      <div className="pgn-ring-center" aria-hidden>
        {children}
      </div>
    </div>
  );
}
