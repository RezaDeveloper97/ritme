'use client';

import { useId, type ReactNode } from 'react';
import { clsx } from 'clsx';

import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';

import { useRovingRadio } from './radio-group';
import { toneClass, type Tone } from './tone';

/** Integers `min … max` inclusive. */
export function scaleValues(min: number, max: number): number[] {
  return Array.from({ length: Math.max(0, max - min + 1) }, (_, i) => min + i);
}

interface NumericScaleProps {
  min: number;
  max: number;
  value: number | null;
  onChange: (value: number) => void;
  /** Digits follow the locale (۷ in fa). */
  locale: Locale;
  /** Visible question («چقدر شدید؟», «غمگینی یا ناامیدی»); names the radiogroup. */
  label?: ReactNode;
  ariaLabel?: string;
  /** Captions under the first / last cell («بدون درد» · «بدترین درد ممکن»). */
  minLabel?: ReactNode;
  maxLabel?: ReactNode;
  /**
   * `solid` = selected cell filled in the tone (pain 0–10, nbl_Cond_Endo);
   * `soft` = 15% tint + tone border (PMDD 1–6, nbl_Cond_PMDD).
   */
  variant?: 'solid' | 'soft';
  tone?: Tone;
  className?: string;
}

/**
 * Row of numbered radio cells with optional end labels. Cells are 44px tall
 * and share the row edge to edge (no dead gaps between hit areas); an 11-step
 * 0–10 scale is narrower than 44px per cell at phone width by design.
 */
export function NumericScale({
  min,
  max,
  value,
  onChange,
  locale,
  label,
  ariaLabel,
  minLabel,
  maxLabel,
  variant = 'soft',
  tone = 'brand',
  className,
}: NumericScaleProps) {
  const labelId = useId();
  const endsId = useId();
  const values = scaleValues(min, max);
  const roving = useRovingRadio(values, value, onChange);
  const hasEnds = Boolean(minLabel || maxLabel);
  return (
    <div className={clsx('nb-nscale', `is-${variant}`, toneClass(tone), values.length > 7 && 'is-dense', className)}>
      {label ? (
        <div id={labelId} className="nb-scale-label">
          {label}
        </div>
      ) : null}
      <div
        role="radiogroup"
        aria-labelledby={label ? labelId : undefined}
        aria-label={label ? undefined : ariaLabel}
        aria-describedby={hasEnds ? endsId : undefined}
        className="nb-nscale-row"
      >
        {values.map((n, index) => (
          <button
            key={n}
            ref={roving.refOf(index)}
            type="button"
            role="radio"
            aria-checked={n === value}
            tabIndex={roving.tabIndexOf(index)}
            className="nb-nscale-cell"
            onClick={() => onChange(n)}
            onKeyDown={(event) => roving.onKeyDown(event, index)}
          >
            <span className="nb-nscale-num">{formatNumber(n, locale)}</span>
          </button>
        ))}
      </div>
      {hasEnds ? (
        <div id={endsId} className="nb-nscale-ends">
          <span>{minLabel}</span>
          <span>{maxLabel}</span>
        </div>
      ) : null}
    </div>
  );
}
