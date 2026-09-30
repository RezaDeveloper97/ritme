'use client';

import type { ReactNode } from 'react';
import { clsx } from 'clsx';

import { useDirection } from '@/shared/i18n';

import { Icon, type IconName } from '../Icon';

interface HeaderButtonProps {
  /** Accessible name — the button shows only an icon (CLAUDE.md §6: from i18n). */
  label: string;
  icon: IconName;
  onClick?: () => void;
  /** `soft` = the hub variant on `--brand-soft`; default = `--surface-2` + line. */
  variant?: 'default' | 'soft';
  /** Small dot on the corner, e.g. unread notifications. */
  badge?: boolean;
  className?: string;
}

/** The 44px round icon button of every Night & Bloom header. */
export function HeaderButton({ label, icon, onClick, variant = 'default', badge, className }: HeaderButtonProps) {
  return (
    <button
      type="button"
      className={clsx('nb-hbtn', variant === 'soft' && 'is-soft', className)}
      onClick={onClick}
      aria-label={label}
    >
      <Icon name={icon} size={20} strokeWidth={1.8} />
      {badge ? <span className="nb-hbtn-dot" aria-hidden /> : null}
    </button>
  );
}

interface ScreenHeaderProps {
  title: ReactNode;
  subtitle?: ReactNode;
  /** Back (or close) handler. Omit on a root screen — a spacer keeps the title centred. */
  onBack?: () => void;
  /** Accessible name of the back button, e.g. «بازگشت» / «بستن». */
  backLabel?: string;
  /** `close` swaps the arrow for an ×, for full-screen sheets. */
  backIcon?: 'back' | 'close';
  /** One 44px action at the end (share, settings, filter) — usually a {@link HeaderButton}. */
  action?: ReactNode;
  /** Replaces the centred title, e.g. {@link ProgressSteps} on onboarding. */
  center?: ReactNode;
  className?: string;
}

/**
 * Night & Bloom screen header: 44px round back button at the start, centred
 * title 17/800 + optional subtitle, optional 44px action at the end.
 * Back never navigates history (CLAUDE.md §4.2) — the caller routes forward.
 */
export function ScreenHeader({
  title,
  subtitle,
  onBack,
  backLabel,
  backIcon = 'back',
  action,
  center,
  className,
}: ScreenHeaderProps) {
  const rtl = useDirection() === 'rtl';
  const arrow: IconName = backIcon === 'close' ? 'x' : rtl ? 'arrowR' : 'arrowL';
  return (
    <header className={clsx('nb-hdr', className)}>
      {onBack ? (
        <HeaderButton label={backLabel ?? ''} icon={arrow} onClick={onBack} />
      ) : (
        <span className="nb-hdr-spacer" aria-hidden />
      )}
      {center ?? (
        <div className="nb-hdr-titles">
          <h1 className="nb-hdr-title">{title}</h1>
          {subtitle ? <p className="nb-hdr-sub">{subtitle}</p> : null}
        </div>
      )}
      {action ?? <span className="nb-hdr-spacer" aria-hidden />}
    </header>
  );
}

interface HubHeaderProps {
  /** Small date line above the greeting, already formatted by `shared/lib/date`. */
  date: ReactNode;
  greeting: ReactNode;
  /** 44px soft round buttons at the end (notifications, settings). */
  actions?: ReactNode;
  className?: string;
}

/** Header of the hub screens (home, TTC main, pregnancy main): date over greeting. */
export function HubHeader({ date, greeting, actions, className }: HubHeaderProps) {
  return (
    <header className={clsx('nb-hub', className)}>
      <div className="nb-hub-text">
        <p className="nb-hub-date">{date}</p>
        <h1 className="nb-hub-greeting">{greeting}</h1>
      </div>
      {actions ? <div className="nb-hub-actions">{actions}</div> : null}
    </header>
  );
}
