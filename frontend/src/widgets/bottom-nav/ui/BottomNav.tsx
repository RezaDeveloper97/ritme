'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { isNavRootPath } from '@/shared/config';
import { Link, usePathname, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet } from '@/shared/sheet';

import {
  activeTabKey,
  navConfig,
  type NavKey,
  type NavMode,
  type NavTab,
} from '../model/nav-items';
import { useNavMode } from '../model/use-nav-mode';
import { NavIcon } from './NavIcon';

export interface BottomNavProps {
  /**
   * Unread counts per tab (`true` = a dot without a number). Screens pass what
   * they know; nothing feeds this yet (notifications/services in later tasks).
   */
  badges?: Partial<Record<NavKey, number | boolean>>;
  /** Override the mode (tests, the dev ui-kit); otherwise read from the API. */
  mode?: NavMode;
}

/**
 * Night & Bloom floating glass nav (B-N1-04, docs/night-bloom/nav.md):
 * امروز · <mode tab> · solid FAB (+) · خدمات · من. Rendered only on tab roots
 * and first-level hubs (`shared/config/app-nav`) — every other screen may keep
 * mounting it and it simply stays out. The FAB opens the mode's log sheet
 * (`?sheet=log`) over the current screen instead of navigating.
 */
export function BottomNav({ badges, mode: forcedMode }: BottomNavProps) {
  const t = useTranslations('nav');
  const locale = useLocale() as Locale;
  const pathname = usePathname();
  const navMode = useNavMode();
  // The mode query is gated on the token in localStorage (off during SSR), so
  // the mode tab is a placeholder until hydration — both renders then match.
  const mounted = useMounted();

  if (!isNavRootPath(pathname)) return null;

  // While the mode is unknown the pathname is a safe hint for pregnancy; any
  // other mode tab renders as a placeholder so it never flashes the wrong one.
  const pending = !forcedMode && (!mounted || navMode.pending);
  const hinted: NavMode | null = pathname.startsWith('/pregnancy') ? 'pregnancy' : null;
  const mode: NavMode = forcedMode ?? navMode.mode ?? hinted ?? 'cycle';
  const config = navConfig(mode);
  const active = activeTabKey(config, pathname);
  const placeholderMode = pending && !hinted;

  const renderTab = (tab: NavTab, index: number) => {
    if (placeholderMode && index === 1 && config.before.length > 1) {
      return (
        <span key="mode-placeholder" className="nbnav-tab is-placeholder" aria-hidden>
          <span className="nbnav-ph-ic" />
          <span className="nbnav-ph-label" />
        </span>
      );
    }
    const on = tab.key === active;
    const badge = badges?.[tab.key];
    return (
      <Link
        key={tab.key}
        href={tab.href}
        className={clsx('nbnav-tab', on && 'is-on')}
        aria-current={on ? 'page' : undefined}
      >
        <span className="nbnav-ic-wrap">
          <NavIcon name={tab.icon} />
          {badge ? (
            <span className={clsx('nbnav-badge', badge === true && 'is-dot')} aria-hidden>
              {badge === true ? null : formatNumber(Math.min(badge, 99), locale)}
            </span>
          ) : null}
        </span>
        <span className="nbnav-label">{t(tab.key)}</span>
        {badge ? (
          <span className="sr-only">
            {badge === true ? t('badgeNew') : t('badgeCount', { count: badge })}
          </span>
        ) : null}
      </Link>
    );
  };

  return (
    <nav className="nbnav" aria-label={t('label')}>
      {config.before.map((tab, i) => renderTab(tab, i))}
      {config.fab ? (
        <button type="button" className="nbnav-fab" aria-label={t('log')} onClick={() => openSheet('log')}>
          <svg
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2.4"
            strokeLinecap="round"
            aria-hidden
            focusable="false"
          >
            <path d="M12 5v14M5 12h14" />
          </svg>
        </button>
      ) : null}
      {config.after.map((tab, i) => renderTab(tab, config.before.length + i))}
    </nav>
  );
}
