'use client';

import { useTranslations } from 'next-intl';

import { useUserMode } from '@/entities/message';
import { Link, usePathname } from '@/shared/i18n';
import { Icon } from '@/shared/ui';

import { navItemsForMode } from '../model/nav-items';

/**
 * Floating bottom navigation shared across the main tabs. Highlights the tab
 * whose href matches the current path; the center "+" is always a gradient FAB.
 * Labels come from the `nav` namespace and the locale prefix is handled by the
 * i18n `Link`, so the same component works in both `fa` (RTL) and `en` (LTR).
 */
export function BottomNav() {
  const t = useTranslations('nav');
  const pathname = usePathname();
  const modeQuery = useUserMode();
  const items = navItemsForMode(modeQuery.data?.mode);

  return (
    <nav className="tabbar" aria-label={t('label')}>
      {items.map((item) => {
        if (item.fab) {
          return (
            <Link
              key={item.key}
              href={item.href}
              className="fab"
              aria-label={t(item.key)}
            >
              <Icon name={item.icon} size={26} className="ic" />
            </Link>
          );
        }

        const active = pathname === item.href;

        return (
          <Link
            key={item.key}
            href={item.href}
            className={active ? 'tab on' : 'tab'}
            aria-current={active ? 'page' : undefined}
          >
            <Icon name={item.icon} size={24} className="ic" />
            <span>{t(item.key)}</span>
          </Link>
        );
      })}
    </nav>
  );
}
