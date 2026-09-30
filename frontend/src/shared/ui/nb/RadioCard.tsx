'use client';

import { useId, type ReactNode } from 'react';
import { clsx } from 'clsx';

import type { IconName } from '../Icon';
import { IconCircle } from './IconCircle';
import { useRovingRadio } from './radio-group';
import type { Tone } from './tone';

export interface RadioCardOption<V extends string> {
  value: V;
  title: ReactNode;
  description?: ReactNode;
  icon?: IconName;
  iconTone?: Tone;
  disabled?: boolean;
}

interface RadioCardGroupProps<V extends string> {
  options: readonly RadioCardOption<V>[];
  value: V | null;
  onChange: (value: V) => void;
  /** Accessible name of the set («مرحله یائسگی»). */
  label: string;
  className?: string;
}

/**
 * One-of-many choice as full-width cards (nbl_Meno_Stage, Loss_Start/Next,
 * Teen_Onb, Nav_Mode): tone icon disc, title 15/800 + description, radio dot
 * at the inline end. Selected = `--brand-soft` fill + `--brand` border.
 * `role=radiogroup` with roving tabindex; each card is ≥ 64px.
 */
export function RadioCardGroup<V extends string>({ options, value, onChange, label, className }: RadioCardGroupProps<V>) {
  const baseId = useId();
  const enabled = options.filter((o) => !o.disabled).map((o) => o.value);
  const roving = useRovingRadio(enabled, value, onChange);
  return (
    <div role="radiogroup" aria-label={label} className={clsx('nb-rcards', className)}>
      {options.map((option) => {
        const checked = option.value === value;
        const index = enabled.indexOf(option.value);
        const titleId = `${baseId}-${option.value}-t`;
        const descId = `${baseId}-${option.value}-d`;
        return (
          <button
            key={option.value}
            ref={index >= 0 ? roving.refOf(index) : undefined}
            type="button"
            role="radio"
            aria-checked={checked}
            aria-labelledby={titleId}
            aria-describedby={option.description ? descId : undefined}
            disabled={option.disabled}
            tabIndex={index >= 0 ? roving.tabIndexOf(index) : -1}
            className="nb-rcard"
            onClick={() => onChange(option.value)}
            onKeyDown={(event) => index >= 0 && roving.onKeyDown(event, index)}
          >
            {option.icon ? <IconCircle icon={option.icon} tone={option.iconTone ?? 'brand'} size="lg" /> : null}
            <span className="nb-rcard-text">
              <span id={titleId} className="nb-rcard-title">
                {option.title}
              </span>
              {option.description ? (
                <span id={descId} className="nb-rcard-desc">
                  {option.description}
                </span>
              ) : null}
            </span>
            <span className="nb-rcard-dot" aria-hidden />
          </button>
        );
      })}
    </div>
  );
}
