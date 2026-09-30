'use client';

import type { ButtonHTMLAttributes, ReactNode } from 'react';
import { clsx } from 'clsx';

import { Icon, type IconName } from '../Icon';
import { toneClass, type Tone } from './tone';

interface PillChipProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'onChange'> {
  pressed: boolean;
  onPressedChange?: (next: boolean) => void;
  /**
   * `single` = solid `--brand-fill` when on (one-of-many);
   * `multi` = 15% tone tint + 1.5px tone border + check (many-of-many).
   */
  mode?: 'single' | 'multi';
  /** Accent of the multi variant (e.g. `period` for bleeding, `bloom` for pain). */
  tone?: Tone;
  /** Dashed `--period` outline for a suggested day. */
  suggested?: boolean;
  icon?: IconName;
  /** `square` = radius 14 (intensity rows), default pill radius 20. */
  shape?: 'pill' | 'square';
  children: ReactNode;
}

/** 40px toggle chip with `aria-pressed`. */
export function PillChip({
  pressed,
  onPressedChange,
  mode = 'single',
  tone = 'brand',
  suggested,
  icon,
  shape = 'pill',
  className,
  onClick,
  type = 'button',
  children,
  ...rest
}: PillChipProps) {
  return (
    <button
      type={type}
      aria-pressed={pressed}
      className={clsx(
        'nb-chip',
        mode === 'multi' ? 'is-multi' : 'is-single',
        toneClass(tone),
        shape === 'square' && 'is-square',
        suggested && 'is-suggested',
        className,
      )}
      onClick={(event) => {
        onClick?.(event);
        if (!event.defaultPrevented) onPressedChange?.(!pressed);
      }}
      {...rest}
    >
      {icon ? <Icon name={icon} size={16} /> : null}
      {children}
      {mode === 'multi' && pressed ? <Icon name="check" size={12} strokeWidth={3} className="nb-chip-check" /> : null}
    </button>
  );
}

interface ChipGroupProps {
  /** Accessible name of the set, e.g. «محل درد». */
  label: string;
  /** `fill` stretches the chips across the row (intensity: کم · متوسط · شدید). */
  layout?: 'wrap' | 'fill';
  className?: string;
  children: ReactNode;
}

/** A labelled row of {@link PillChip}s. */
export function ChipGroup({ label, layout = 'wrap', className, children }: ChipGroupProps) {
  return (
    <div role="group" aria-label={label} className={clsx('nb-chips', layout === 'fill' && 'is-fill', className)}>
      {children}
    </div>
  );
}
