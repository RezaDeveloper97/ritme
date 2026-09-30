'use client';

import { useId, type ReactNode } from 'react';
import { clsx } from 'clsx';

import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';

import { Icon } from '../Icon';

interface NumberStepperProps {
  value: number;
  onChange: (value: number) => void;
  min: number;
  max: number;
  step?: number;
  /** Visible label, e.g. «طول پریود». Also names the group. */
  label: ReactNode;
  description?: ReactNode;
  /** Caption under the number, e.g. «روز». */
  unit?: ReactNode;
  /** Accessible names of the − / + buttons («کم کردن» / «زیاد کردن»). */
  decrementLabel: string;
  incrementLabel: string;
  /** Digits follow the locale (۵ in fa). */
  locale: Locale;
  /** Render a flat card around the row (the onboarding layout). */
  boxed?: boolean;
  className?: string;
}

/** Label + 44px round −/+ around a big Lalezar value and its unit. */
export function NumberStepper({
  value,
  onChange,
  min,
  max,
  step = 1,
  label,
  description,
  unit,
  decrementLabel,
  incrementLabel,
  locale,
  boxed,
  className,
}: NumberStepperProps) {
  const labelId = useId();
  const clamp = (n: number) => Math.min(max, Math.max(min, n));
  return (
    <div
      role="group"
      aria-labelledby={labelId}
      className={clsx('nb-stepper', boxed && 'nb-card pad-md', className)}
    >
      <div className="nb-stepper-text">
        <div id={labelId} className="nb-stepper-label">
          {label}
        </div>
        {description ? <div className="nb-stepper-desc">{description}</div> : null}
      </div>
      <button
        type="button"
        className="nb-stepper-btn"
        aria-label={decrementLabel}
        disabled={value <= min}
        onClick={() => onChange(clamp(value - step))}
      >
        <Icon name="minus" size={20} strokeWidth={2.4} />
      </button>
      <output className="nb-stepper-value" aria-live="polite">
        <span className="nb-stepper-num">{formatNumber(value, locale)}</span>
        {unit ? <span className="nb-stepper-unit">{unit}</span> : null}
      </output>
      <button
        type="button"
        className="nb-stepper-btn"
        aria-label={incrementLabel}
        disabled={value >= max}
        onClick={() => onChange(clamp(value + step))}
      >
        <Icon name="plus" size={20} strokeWidth={2.4} />
      </button>
    </div>
  );
}
