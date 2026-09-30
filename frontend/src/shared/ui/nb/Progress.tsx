import type { ReactNode } from 'react';
import { clsx } from 'clsx';

import { ringDash } from './chart-geometry';
import { toneClass, type Tone } from './tone';

interface ProgressRingProps {
  /** 0 – 1. */
  value: number;
  /** Accessible name, e.g. «پیشرفت سیکل». */
  label: string;
  /** Spoken value, e.g. «روز ۱۴ از ۲۹»; defaults to the percentage. */
  valueText?: string;
  /** Outer diameter in px. */
  size?: number;
  thickness?: number;
  tone?: Tone;
  /** Centre content — typically a Lalezar number + caption. */
  children?: ReactNode;
  className?: string;
}

/** SVG progress ring with centred content (`role=progressbar`). */
export function ProgressRing({
  value,
  label,
  valueText,
  size = 160,
  thickness = 12,
  tone = 'brand',
  children,
  className,
}: ProgressRingProps) {
  const radius = (size - thickness) / 2;
  const { dash } = ringDash(value, radius);
  const c = size / 2;
  return (
    <div
      role="progressbar"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(Math.max(0, Math.min(1, value)) * 100)}
      aria-valuetext={valueText}
      className={clsx('nb-ring', toneClass(tone), className)}
    >
      <svg viewBox={`0 0 ${size} ${size}`} width={size} height={size} aria-hidden>
        <circle className="nb-ring-track" cx={c} cy={c} r={radius} strokeWidth={thickness} />
        <circle
          className="nb-ring-arc"
          cx={c}
          cy={c}
          r={radius}
          strokeWidth={thickness}
          strokeDasharray={dash}
          transform={`rotate(-90 ${c} ${c})`}
        />
      </svg>
      {children ? <div className="nb-ring-center">{children}</div> : null}
    </div>
  );
}

interface ProgressStepsProps {
  total: number;
  /** Steps completed, including the current one (onboarding step 3 of 8 → 3). */
  current: number;
  /** Accessible name, e.g. «مرحله ۳ از ۸». */
  label: string;
  className?: string;
}

/** Row of 4px bars in a header: done = `--brand-fill`, to do = `--line`. */
export function ProgressSteps({ total, current, label, className }: ProgressStepsProps) {
  return (
    <div
      role="progressbar"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={total}
      aria-valuenow={current}
      className={clsx('nb-steps', className)}
    >
      {Array.from({ length: total }, (_, i) => (
        <span key={i} className={clsx('nb-step', i < current && 'is-done')} />
      ))}
    </div>
  );
}
