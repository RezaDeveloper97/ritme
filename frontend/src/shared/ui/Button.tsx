import { clsx } from 'clsx';
import type { ButtonHTMLAttributes, ReactNode } from 'react';

type ButtonVariant = 'primary' | 'secondary' | 'ghost';
type ButtonSize = 'sm' | 'md';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  children: ReactNode;
}

const base =
  'inline-flex items-center justify-center gap-2 rounded-xl font-medium ' +
  'transition-colors focus-visible:outline-none focus-visible:ring-2 ' +
  'focus-visible:ring-offset-2 focus-visible:ring-pink-500 ' +
  'disabled:pointer-events-none disabled:opacity-50';

const variants: Record<ButtonVariant, string> = {
  primary: 'bg-pink-600 text-white hover:bg-pink-700',
  secondary: 'bg-pink-50 text-pink-700 hover:bg-pink-100',
  ghost: 'bg-transparent text-pink-700 hover:bg-pink-50',
};

// Logical padding (ps/pe) keeps spacing correct in both RTL and LTR.
const sizes: Record<ButtonSize, string> = {
  sm: 'ps-3 pe-3 py-1.5 text-sm',
  md: 'ps-5 pe-5 py-2.5 text-base',
};

/**
 * Shared button primitive — domain-agnostic and RTL-safe (logical spacing, no
 * hardcoded left/right). Visible text is always passed in by the caller via
 * i18n; this component never hardcodes copy (CLAUDE.md §6, §10, §12).
 */
export function Button({
  variant = 'primary',
  size = 'md',
  className,
  type = 'button',
  children,
  ...rest
}: ButtonProps) {
  return (
    <button
      type={type}
      className={clsx(base, variants[variant], sizes[size], className)}
      {...rest}
    >
      {children}
    </button>
  );
}
