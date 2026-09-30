import { useId, type ReactNode } from 'react';
import { clsx } from 'clsx';

import { toneClass, type Tone } from './tone';

/** Fill percentage (0–100) of `value` out of `max`; empty when `max` ≤ 0. */
export function barPercent(value: number, max: number): number {
  if (!(max > 0)) return 0;
  return Math.max(0, Math.min(100, (value / max) * 100));
}

interface ProgressBarProps {
  value: number;
  max: number;
  /** Row title («پاراکلینیک», «پیاده‌روی تند»); names the progressbar. */
  label: ReactNode;
  /** Visible «X از Y» / «۱۴ از ۲۰ میلیون مانده» at the row end, in locale digits. */
  valueLabel?: ReactNode;
  /** Spoken value when the visible one is not a sentence, e.g. «۶ میلیون مصرف شده از ۲۰». */
  valueText?: string;
  /** Caption under the bar («فقط بعد از ۹ ماه از شروع بیمه‌نامه»). */
  hint?: ReactNode;
  tone?: Tone;
  className?: string;
}

/**
 * Labelled bar on `--track` (8px, pill): caps and usage (nbl_Ins_Coverage),
 * weekly goals (nbl_Meno_Treatment «۹۰ از ۱۵۰»), checklist groups. The fill
 * grows from the inline start, so it follows RTL.
 */
export function ProgressBar({ value, max, label, valueLabel, valueText, hint, tone = 'brand', className }: ProgressBarProps) {
  const labelId = useId();
  const pct = barPercent(value, max);
  return (
    <div className={clsx('nb-pbar', toneClass(tone), className)}>
      <div className="nb-pbar-head">
        <span id={labelId} className="nb-pbar-label">
          {label}
        </span>
        {valueLabel ? (
          <span className="nb-pbar-value" aria-hidden={valueText ? true : undefined}>
            {valueLabel}
          </span>
        ) : null}
      </div>
      <div
        role="progressbar"
        aria-labelledby={labelId}
        aria-valuemin={0}
        aria-valuemax={max}
        aria-valuenow={Math.max(0, Math.min(max, value))}
        aria-valuetext={valueText}
        className="nb-pbar-track"
      >
        <span className="nb-pbar-fill" style={{ inlineSize: `${pct}%` }} />
      </div>
      {hint ? <div className="nb-pbar-hint">{hint}</div> : null}
    </div>
  );
}
