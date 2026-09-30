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

/** Most cells one row may hold and still give each ≥ 44px at 390px (CB-CORE-02b). */
const MAX_COLS = 7;

/**
 * Columns per row: one row up to {@link MAX_COLS} cells, otherwise the cells
 * wrap into even rows (0–10 → 6 + 5), capped at 6 columns (the CSS classes).
 */
export function scaleColumns(count: number): number {
  if (count <= MAX_COLS) return Math.max(1, count);
  const rows = Math.ceil(count / 6);
  return Math.ceil(count / rows);
}

/**
 * Tones whose fill carries `--on-brand` at ≥ 4.5:1 in BOTH themes
 * (brand 5.0 / 8.7, danger 5.4 / 8.4, success 5.2 / 11.4). The other
 * accents (period 3.8, data 3.0, warm 2.0, bloom 2.6 under white in light)
 * fail AA, so `solid` never paints them.
 */
export const SOLID_SCALE_TONES = ['brand', 'danger', 'success'] as const;
export type SolidScaleTone = (typeof SOLID_SCALE_TONES)[number];

function isSolidTone(tone: Tone): tone is SolidScaleTone {
  return (SOLID_SCALE_TONES as readonly Tone[]).includes(tone);
}

interface NumericScaleBaseProps {
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
  className?: string;
}

/**
 * `solid` = selected cell filled in the tone (pain 0–10, nbl_Cond_Endo) —
 * only the {@link SOLID_SCALE_TONES} (text on the fill stays AA);
 * `soft` = 15% tint + tone border (PMDD 1–6, nbl_Cond_PMDD), any tone.
 */
type NumericScaleProps = NumericScaleBaseProps &
  ({ variant: 'solid'; tone?: SolidScaleTone } | { variant?: 'soft'; tone?: Tone });

/**
 * Numbered radio cells with optional end labels. Cells are ≥ 44px tall and
 * share the row edge to edge (no dead gaps between hit areas). More than 7
 * steps wrap into two rows (0–10 → 0–5 / 6–10) so every cell stays ≥ 44px
 * wide at 390px — one radiogroup, arrows still walk 0 → 10 in order.
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
  const dense = values.length > MAX_COLS;
  // Defensive: a caller casting past the type still gets an AA-safe fill.
  const paint: Tone = variant === 'solid' && !isSolidTone(tone) ? 'brand' : tone;
  return (
    <div
      className={clsx(
        'nb-nscale',
        `is-${variant}`,
        toneClass(paint),
        dense && 'is-dense',
        dense && `is-cols-${scaleColumns(values.length)}`,
        className,
      )}
    >
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
