'use client';

import type { ReactNode } from 'react';
import { clsx } from 'clsx';

import { Icon } from '../Icon';

interface CheckboxProps {
  checked: boolean;
  onCheckedChange: (next: boolean) => void;
  /** Visible label — the whole row is the hit target (≥ 44px). */
  label: ReactNode;
  description?: ReactNode;
  disabled?: boolean;
  className?: string;
}

/**
 * Native checkbox (visually a 22px rounded box, `--brand-fill` + check when
 * on) inside its label row — checklist items (Teen kit, Shop_Checklist),
 * terms (Dir_Join). Native input keeps Space, form semantics and SR state.
 */
export function Checkbox({ checked, onCheckedChange, label, description, disabled, className }: CheckboxProps) {
  return (
    <label className={clsx('nb-check', disabled && 'is-disabled', className)}>
      <input
        type="checkbox"
        className="nb-check-input"
        checked={checked}
        disabled={disabled}
        onChange={(event) => onCheckedChange(event.target.checked)}
      />
      <span className="nb-check-box" aria-hidden>
        <Icon name="check" size={14} strokeWidth={3} />
      </span>
      <span className="nb-check-text">
        <span className="nb-check-label">{label}</span>
        {description ? <span className="nb-check-desc">{description}</span> : null}
      </span>
    </label>
  );
}
