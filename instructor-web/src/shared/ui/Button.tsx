import type { ButtonHTMLAttributes, ReactNode } from 'react';

import { cn } from '@/shared/lib';

import { Spinner } from './Spinner';

type Variant = 'primary' | 'ghost' | 'text';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  loading?: boolean;
  block?: boolean;
  children: ReactNode;
}

/** 54px pill (primary solid --brand-fill, ghost outlined) or a text link-button. */
export function Button({
  variant = 'primary',
  loading = false,
  block = false,
  disabled,
  className,
  children,
  type = 'button',
  ...rest
}: ButtonProps) {
  return (
    <button
      type={type}
      className={cn('btn', `btn--${variant}`, block && 'btn--block', className)}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      {...rest}
    >
      {loading ? <Spinner size="sm" /> : children}
    </button>
  );
}

interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  label: string;
  children: ReactNode;
}

/** 44px round header button (Night & Bloom). */
export function IconButton({ label, className, children, type = 'button', ...rest }: IconButtonProps) {
  return (
    <button type={type} className={cn('icon-btn', className)} aria-label={label} title={label} {...rest}>
      {children}
    </button>
  );
}
