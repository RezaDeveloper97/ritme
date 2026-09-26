import type { AppMode } from '@/entities/message';
import type { IconName } from '@/shared/ui';

export type NavKey = 'today' | 'calendar' | 'log' | 'cycle' | 'pregnancy' | 'profile';

export interface NavItem {
  /** Stable key; also the message key under the `nav` i18n namespace. */
  key: NavKey;
  /** Locale-agnostic path — the locale prefix is added by the i18n `Link`. */
  href: string;
  icon: IconName;
  /** The center "+" renders as a raised gradient FAB instead of a labelled tab. */
  fab?: boolean;
  /** Also active on sub-paths (`/pregnancy/weeks/12` for `/pregnancy/weeks`). */
  prefix?: boolean;
}

/** Is `item` the tab for `pathname`? */
export function isNavItemActive(item: NavItem, pathname: string): boolean {
  return pathname === item.href || (!!item.prefix && pathname.startsWith(`${item.href}/`));
}

/**
 * DOM order maps to the right-to-left visual order in RTL (`fa`): «امروز» sits
 * on the right and «پروفایل» on the left, with the FAB always in the middle.
 * In LTR (`en`) it reads left-to-right with the FAB still centered.
 */
export const NAV_ITEMS: NavItem[] = [
  { key: 'today', href: '/home', icon: 'home' },
  { key: 'calendar', href: '/calendar', icon: 'calendar' },
  { key: 'log', href: '/log', icon: 'plus', fab: true },
  { key: 'cycle', href: '/cycle', icon: 'chart' },
  { key: 'profile', href: '/profile', icon: 'user' },
];

/**
 * Pregnancy-mode tabs (docs/pregnancy-v2/README.md): امروز · تقویم · gradient
 * FAB → today's log · بارداری (week by week) · پروفایل.
 */
export const PREGNANCY_NAV_ITEMS: NavItem[] = [
  { key: 'today', href: '/pregnancy', icon: 'home' },
  { key: 'calendar', href: '/pregnancy/calendar', icon: 'calendar' },
  { key: 'log', href: '/pregnancy/log', icon: 'plus', fab: true },
  { key: 'pregnancy', href: '/pregnancy/weeks', icon: 'heart', prefix: true },
  { key: 'profile', href: '/profile', icon: 'user' },
];

/**
 * Mode is a first-class concept (CLAUDE.md §1), so the nav adapts: in pregnancy
 * mode every tab points into the pregnancy tracker (except profile).
 */
export function navItemsForMode(mode: AppMode | undefined): NavItem[] {
  return mode === 'pregnancy' ? PREGNANCY_NAV_ITEMS : NAV_ITEMS;
}
