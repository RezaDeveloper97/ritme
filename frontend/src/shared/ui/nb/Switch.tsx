'use client';

import { clsx } from 'clsx';

interface SwitchProps {
  checked: boolean;
  onCheckedChange: (next: boolean) => void;
  /** Accessible name — or pass `labelledBy` when a visible row title names it. */
  label?: string;
  labelledBy?: string;
  describedBy?: string;
  /** 46×28 instead of 48×30. */
  compact?: boolean;
  disabled?: boolean;
  className?: string;
}

/** `role=switch` toggle, 48×30, on = `--brand-fill`; the knob slides to the inline end. */
export function Switch({
  checked,
  onCheckedChange,
  label,
  labelledBy,
  describedBy,
  compact,
  disabled,
  className,
}: SwitchProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={labelledBy ? undefined : label}
      aria-labelledby={labelledBy}
      aria-describedby={describedBy}
      disabled={disabled}
      className={clsx('nb-switch', compact && 'is-compact', className)}
      onClick={() => onCheckedChange(!checked)}
    >
      <span className="nb-switch-knob" aria-hidden />
    </button>
  );
}
