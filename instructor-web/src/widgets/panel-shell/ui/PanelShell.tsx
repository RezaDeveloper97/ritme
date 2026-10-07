'use client';

import { useTranslations } from 'next-intl';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import type { ReactNode } from 'react';

import { initialOf, type Instructor } from '@/entities/instructor';
import { LogoutButton } from '@/features/logout';
import { cn } from '@/shared/lib';
import { Icon, ThemeToggle } from '@/shared/ui';

import { PANEL_NAV, activeNav } from '../model/nav';

/**
 * The panel frame: header (avatar, «پنل مدرس», name · title, theme + logout) and
 * the four tabs — a floating bottom nav on phones (nbl_Ins_Dashboard), a side
 * rail from 1024px.
 */
export function PanelShell({ instructor, children }: { instructor: Instructor; children: ReactNode }) {
  const t = useTranslations('shell');
  const active = activeNav(usePathname());
  const subtitle = instructor.title ? `${instructor.display_name} · ${instructor.title}` : instructor.display_name;

  return (
    <div className="shell">
      <nav className="shell__nav" aria-label={t('navLabel')}>
        <div className="shell__rail-brand">
          <span className="brand-mark brand-mark--sm" aria-hidden="true">
            <Icon name="sparkle" size={18} />
          </span>
          <span>{t('brand')}</span>
        </div>
        <ul className="shell__tabs">
          {PANEL_NAV.map((item) => (
            <li key={item.key}>
              <Link
                href={item.href}
                className={cn('shell__tab', active === item.key && 'is-active')}
                aria-current={active === item.key ? 'page' : undefined}
              >
                <Icon name={item.icon} />
                <span>{t(`nav.${item.key}`)}</span>
              </Link>
            </li>
          ))}
        </ul>
      </nav>
      <div className="shell__main">
        <header className="shell__header">
          <span className="avatar" aria-hidden="true">
            {initialOf(instructor.display_name)}
          </span>
          <div className="shell__who">
            <span className="shell__eyebrow">{t('panel')}</span>
            <span className="shell__name">{subtitle}</span>
          </div>
          <div className="shell__actions">
            <ThemeToggle />
            <LogoutButton />
          </div>
        </header>
        <main className="shell__content">{children}</main>
      </div>
    </div>
  );
}
