import type { IconName } from '@/shared/ui';

export type NavKey = 'today' | 'calendar' | 'log' | 'cycle' | 'profile';

export interface NavItem {
  /** Stable key; also the message key under the `nav` i18n namespace. */
  key: NavKey;
  /** Locale-agnostic path — the locale prefix is added by the i18n `Link`. */
  href: string;
  icon: IconName;
  /** The center "+" renders as a raised gradient FAB instead of a labelled tab. */
  fab?: boolean;
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
  { key: 'cycle', href: '/cycle', icon: 'refresh' },
  { key: 'profile', href: '/profile', icon: 'user' },
];
