import type { ReactNode } from 'react';
import { clsx } from 'clsx';

import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';

import { ProgressRing } from './Progress';
import { formatClock, timerProgress } from './timer';
import type { Tone } from './tone';

interface CountdownRingProps {
  /** Time run so far — from {@link useTimer} or the caller's own clock. */
  elapsedMs: number;
  /** Countdown length / elapsed cap (Kegel hold 5 s). Uncapped `elapsed` laps once a minute. */
  totalMs?: number;
  /** `elapsed` counts up (hot flash 01:42); `countdown` shows the time left (hold 3 of 5 s). */
  mode?: 'elapsed' | 'countdown';
  /** Accessible name, e.g. «زمان گرگرفتگی». */
  label: string;
  locale: Locale;
  /** Big centre value; defaults to the locale `mm:ss` clock. */
  display?: ReactNode;
  /** Caption above / below the value («الان دارم», «از ۵ ثانیه»). */
  caption?: ReactNode;
  footer?: ReactNode;
  size?: number;
  thickness?: number;
  tone?: Tone;
  /** Soft radial tone disc behind the ring (Meno_HotFlash, Pelvic_Kegel). */
  glow?: boolean;
  className?: string;
}

/**
 * Timer face: a thin wrapper over {@link ProgressRing} (same `role=progressbar`)
 * that turns a duration into ring fill and a Lalezar clock. Not a live region —
 * a ticking clock must not be announced every second.
 */
export function CountdownRing({
  elapsedMs,
  totalMs,
  mode = 'elapsed',
  label,
  locale,
  display,
  caption,
  footer,
  size = 220,
  thickness = 8,
  tone = 'brand',
  glow,
  className,
}: CountdownRingProps) {
  const shown = mode === 'countdown' && totalMs ? Math.max(0, totalMs - elapsedMs) : elapsedMs;
  // Count-down rounds up so «1» stays until the hold is really over.
  const clock = formatNumber(formatClock(mode === 'countdown' ? Math.ceil(shown / 1000) * 1000 : shown), locale);
  return (
    <ProgressRing
      value={timerProgress(elapsedMs, totalMs, mode)}
      label={label}
      valueText={clock}
      size={size}
      thickness={thickness}
      tone={tone}
      className={clsx('nb-countdown', glow && 'is-glow', className)}
    >
      {caption ? <span className="nb-countdown-caption">{caption}</span> : null}
      <span className="nb-countdown-value">{display ?? clock}</span>
      {footer ? <span className="nb-countdown-footer">{footer}</span> : null}
    </ProgressRing>
  );
}
