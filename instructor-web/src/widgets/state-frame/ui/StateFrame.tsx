'use client';

import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { LogoutButton } from '@/features/logout';
import { Icon, ThemeToggle } from '@/shared/ui';

/** Frame of the pre-panel screens (apply, pending): brand, theme, logout; centred column. */
export function StateFrame({ children }: { children: ReactNode }) {
  const t = useTranslations('shell');
  return (
    <div className="state-frame">
      <header className="state-frame__bar">
        <span className="state-frame__brand">
          <span className="brand-mark brand-mark--sm" aria-hidden="true">
            <Icon name="sparkle" size={18} />
          </span>
          <span>{t('brand')}</span>
        </span>
        <div className="shell__actions">
          <ThemeToggle />
          <LogoutButton />
        </div>
      </header>
      <main className="state-frame__body">{children}</main>
    </div>
  );
}
