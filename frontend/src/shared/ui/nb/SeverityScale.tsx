'use client';

import { useId, type ReactNode } from 'react';
import { clsx } from 'clsx';

import { useRovingRadio } from './radio-group';
import { toneClass, type Tone } from './tone';

export interface SeverityOption<V extends string> {
  value: V;
  label: ReactNode;
  /** Tint of the selected option; defaults to the ramp in {@link severityTone}. */
  tone?: Tone;
}

/**
 * Default tint ramp by position: ندارم = neutral, خفیف = data, متوسط = bloom,
 * شدید = danger (nbl_Meno_Log). A scale without a «none» step
 * (خفیف…خیلی شدید, nbl_Meno_HotFlash) passes `withNone={false}` and starts at data.
 */
export function severityTone(index: number, withNone = true): Tone {
  const ramp: Tone[] = withNone ? ['neutral', 'data', 'bloom', 'danger'] : ['data', 'warm', 'bloom', 'danger'];
  return ramp[Math.min(index, ramp.length - 1)];
}

interface SeverityScaleProps<V extends string> {
  options: readonly SeverityOption<V>[];
  value: V | null;
  onChange: (value: V) => void;
  /** Visible row title («گرگرفتگی»); also names the radiogroup. Omit when `ariaLabel` is given. */
  label?: ReactNode;
  ariaLabel?: string;
  /** First option is a «none» step (ندارم) — picks the neutral-first tint ramp. */
  withNone?: boolean;
  className?: string;
}

/**
 * Segmented 4-level radiogroup on a `--surface-2` track (nbl_Meno_Log rows,
 * nbl_Meno_HotFlash intensity). Selected option = tone tint + 1.5px tone
 * border. 44px options, roving tabindex, arrows follow reading direction.
 */
export function SeverityScale<V extends string>({
  options,
  value,
  onChange,
  label,
  ariaLabel,
  withNone = true,
  className,
}: SeverityScaleProps<V>) {
  const labelId = useId();
  const values = options.map((o) => o.value);
  const roving = useRovingRadio(values, value, onChange);
  return (
    <div className={clsx('nb-sev', className)}>
      {label ? (
        <div id={labelId} className="nb-scale-label">
          {label}
        </div>
      ) : null}
      <div
        role="radiogroup"
        aria-labelledby={label ? labelId : undefined}
        aria-label={label ? undefined : ariaLabel}
        className="nb-sev-track"
      >
        {options.map((option, index) => {
          const checked = option.value === value;
          return (
            <button
              key={option.value}
              ref={roving.refOf(index)}
              type="button"
              role="radio"
              aria-checked={checked}
              tabIndex={roving.tabIndexOf(index)}
              className={clsx('nb-sev-opt', toneClass(option.tone ?? severityTone(index, withNone)))}
              onClick={() => onChange(option.value)}
              onKeyDown={(event) => roving.onKeyDown(event, index)}
            >
              {option.label}
            </button>
          );
        })}
      </div>
    </div>
  );
}
