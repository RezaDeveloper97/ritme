import { clsx } from 'clsx';
import { useId, type ReactNode } from 'react';

import {
  CYCLE_DAYS,
  CYCLE_DOT_R,
  PREGNANCY_WEEKS,
  RING_VIEW,
  cycleDotKind,
  isCycleDotFilled,
  ringPoint,
  trimesterOf,
} from '../lib/ring';

interface RingFrameProps {
  /** Accessible name of the illustration; omit when the ring is decorative. */
  label?: string;
  className?: string;
  /** Text centred inside the ring. */
  children?: ReactNode;
  svg: (glowId: string) => ReactNode;
}

function RingFrame({ label, className, children, svg }: RingFrameProps) {
  const glowId = `ib-glow-${useId().replace(/:/g, '')}`;
  return (
    <div className={clsx('ib-ring', className)}>
      <svg
        viewBox={`0 0 ${RING_VIEW} ${RING_VIEW}`}
        className="ib-ring-svg"
        role={label ? 'img' : undefined}
        aria-label={label}
        aria-hidden={label ? undefined : true}
        focusable="false"
      >
        <defs>
          <filter id={glowId}>
            <feGaussianBlur stdDeviation="3" />
          </filter>
        </defs>
        <circle cx={RING_VIEW / 2} cy={RING_VIEW / 2} r="128" className="ib-ring-track" />
        {svg(glowId)}
      </svg>
      {children ? <div className="ib-ring-center">{children}</div> : null}
    </div>
  );
}

/** The glowing "today" marker: blurred halo + a ringed dot. */
function Marker({ x, y, r, halo, glowId, tone }: {
  x: number; y: number; r: number; halo: number; glowId: string; tone: string;
}) {
  return (
    <g className={clsx('ib-marker', tone)}>
      <circle cx={x} cy={y} r={halo} className="ib-marker-halo" filter={`url(#${glowId})`} />
      <circle cx={x} cy={y} r={r} className="ib-marker-dot" />
    </g>
  );
}

interface CycleDotRingProps {
  /** 0-based day that carries the glowing marker; earlier days are filled. */
  today: number;
  label?: string;
  className?: string;
  children?: ReactNode;
}

/**
 * A 29-dot illustrative cycle (period · quiet · fertile + ovulation · PMS),
 * the brand mark of the splash, intro slide 1 and the welcome card. Colours
 * come from the cycle tokens so light/dark both follow the palette.
 */
export function CycleDotRing({ today, label, className, children }: CycleDotRingProps) {
  return (
    <RingFrame
      label={label}
      className={className}
      svg={(glowId) => {
        const dots = Array.from({ length: CYCLE_DAYS }, (_, i) => {
          const kind = cycleDotKind(i);
          const { x, y } = ringPoint(i, CYCLE_DAYS);
          return (
            <circle
              key={i}
              cx={x}
              cy={y}
              r={CYCLE_DOT_R[kind]}
              className={clsx('ib-dot', `is-${kind}`, !isCycleDotFilled(i, today) && 'is-outline')}
            />
          );
        });
        const p = ringPoint(today, CYCLE_DAYS);
        return (
          <>
            {dots}
            <Marker x={p.x} y={p.y} r={10} halo={16} glowId={glowId} tone={`is-${cycleDotKind(today)}`} />
          </>
        );
      }}
    >
      {children}
    </RingFrame>
  );
}

interface PregnancyDotRingProps {
  /** Current week, 1-based. */
  week: number;
  label?: string;
  className?: string;
  children?: ReactNode;
}

/** 40 week dots coloured by trimester, with the current week glowing. */
export function PregnancyDotRing({ week, label, className, children }: PregnancyDotRingProps) {
  const current = Math.min(PREGNANCY_WEEKS, Math.max(1, week)) - 1;
  return (
    <RingFrame
      label={label}
      className={clsx('is-pregnancy', className)}
      svg={(glowId) => {
        const dots = Array.from({ length: PREGNANCY_WEEKS }, (_, i) => {
          const { x, y } = ringPoint(i, PREGNANCY_WEEKS);
          const future = i > current;
          return (
            <circle
              key={i}
              cx={x}
              cy={y}
              r={future ? 4 : 5.5}
              className={clsx('ib-dot', `is-tri${trimesterOf(i)}`, future && 'is-future')}
            />
          );
        });
        const p = ringPoint(current, PREGNANCY_WEEKS);
        return (
          <>
            {dots}
            <Marker x={p.x} y={p.y} r={9} halo={15} glowId={glowId} tone={`is-tri${trimesterOf(current)}`} />
          </>
        );
      }}
    >
      {children}
    </RingFrame>
  );
}
