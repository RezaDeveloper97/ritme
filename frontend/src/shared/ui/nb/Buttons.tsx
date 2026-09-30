import type { ButtonHTMLAttributes, ReactNode } from 'react';
import { clsx } from 'clsx';

import { Icon, type IconName } from '../Icon';
import { toneClass, type Tone } from './tone';

interface PillButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  icon?: IconName;
  /** Shows a busy state and blocks clicks; the label stays for screen readers. */
  loading?: boolean;
  /** Full width (the default in forms and sheets). */
  block?: boolean;
  children: ReactNode;
}

/** 54px solid pill: `--brand-fill`, 15/800 `--on-brand`, `--shadow-cta`. */
export function PrimaryButton({
  icon,
  loading,
  block = true,
  className,
  type = 'button',
  disabled,
  children,
  ...rest
}: PillButtonProps) {
  return (
    <button
      type={type}
      className={clsx('nb-btn', 'is-primary', block && 'is-block', loading && 'is-loading', className)}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      {...rest}
    >
      {icon ? <Icon name={icon} size={20} /> : null}
      {children}
    </button>
  );
}

interface SecondaryButtonProps extends PillButtonProps {
  /** `outline` = 54px ghost with 1.5px `--line-strong`; `text` = link-style «هنوز نه». */
  variant?: 'outline' | 'text';
}

/** 54px ghost pill (or the text-link variant) in `--brand-strong`. */
export function SecondaryButton({
  icon,
  loading,
  block = true,
  variant = 'outline',
  className,
  type = 'button',
  disabled,
  children,
  ...rest
}: SecondaryButtonProps) {
  return (
    <button
      type={type}
      className={clsx(
        'nb-btn',
        variant === 'text' ? 'is-text' : 'is-outline',
        block && 'is-block',
        loading && 'is-loading',
        className,
      )}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      {...rest}
    >
      {icon ? <Icon name={icon} size={20} /> : null}
      {children}
    </button>
  );
}

interface TileButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> {
  icon: IconName;
  label: ReactNode;
  /** Caption under the label in the tone («باز است», «زودرنج»). `card` layout only. */
  sub?: ReactNode;
  tone?: Tone;
  /**
   * `compact` = 58px tile, icon over label, 1.5px line, radius 16 (Cycle_Home_During);
   * `card` = quick-log card, 40px tinted icon disc, radius 24 (Log_Sheet_Cycle).
   */
  layout?: 'compact' | 'card';
  /** Toggle tiles expose `aria-pressed`; plain action tiles leave it unset. */
  pressed?: boolean;
}

/** Quick-action tile: icon over label; pressed = tone border + tint. */
export function TileButton({
  icon,
  label,
  sub,
  tone = 'brand',
  layout = 'compact',
  pressed,
  className,
  type = 'button',
  ...rest
}: TileButtonProps) {
  return (
    <button
      type={type}
      className={clsx('nb-tile', layout === 'card' && 'is-card', toneClass(tone), className)}
      aria-pressed={pressed}
      {...rest}
    >
      {layout === 'card' ? (
        <span className="nb-tile-disc" aria-hidden>
          <Icon name={icon} size={20} strokeWidth={1.8} />
        </span>
      ) : (
        <Icon name={icon} size={20} strokeWidth={1.8} />
      )}
      <span className="nb-tile-label">{label}</span>
      {sub && layout === 'card' ? <span className="nb-tile-sub">{sub}</span> : null}
    </button>
  );
}
