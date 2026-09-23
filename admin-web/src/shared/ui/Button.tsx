import type { ButtonHTMLAttributes } from 'react';

import { cn } from '@/shared/lib';

import { Spinner } from './Spinner';

export type ButtonVariant = 'default' | 'primary' | 'gradient' | 'danger' | 'ghost';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: 'md' | 'sm';
  icon?: boolean;
  loading?: boolean;
}

/**
 * `primary` is the everyday save button; `gradient` is reserved for the one
 * brand moment on a screen (login) — the gradient is scarce by design.
 */
export function Button({
  variant = 'default',
  size = 'md',
  icon = false,
  loading = false,
  disabled,
  className,
  children,
  type = 'button',
  ...rest
}: ButtonProps) {
  return (
    <button
      type={type}
      className={cn(
        'btn',
        variant !== 'default' && `btn-${variant}`,
        size === 'sm' && 'btn-sm',
        icon && 'btn-icon',
        className,
      )}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      {...rest}
    >
      {loading ? <Spinner /> : null}
      {children}
    </button>
  );
}
