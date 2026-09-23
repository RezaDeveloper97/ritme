import type { AdminRole } from '@/entities/admin';
import type { IconName } from '@/shared/ui';

/**
 * The sidebar, mirroring the Blade panel's sections (backend/resources/views/
 * admin/layouts/app.blade.php). `ready: false` items render disabled until
 * their screen lands (T-M2-23 flips them). `role: 'super'` items are hidden
 * from editors — the API enforces the same with 403 (admin-api.md §4).
 */
export interface NavItem {
  key: string; // messages key under `nav`
  href: string;
  icon: IconName;
  role?: AdminRole;
  ready: boolean;
  devOnly?: boolean;
}

export interface NavGroup {
  key: string; // messages key under `nav`
  items: NavItem[];
}

export const NAV: readonly NavGroup[] = [
  {
    key: 'groupGeneral',
    items: [
      { key: 'dashboard', href: '/', icon: 'dashboard', ready: true },
      { key: 'users', href: '/users', icon: 'users', ready: false },
    ],
  },
  {
    key: 'groupContent',
    items: [
      { key: 'articles', href: '/articles', icon: 'article', ready: false },
      { key: 'affirmations', href: '/affirmations', icon: 'sparkle', ready: false },
      { key: 'challenges', href: '/challenges', icon: 'flag', ready: false },
      { key: 'challengeCompletions', href: '/challenge-completions', icon: 'check', ready: false },
      { key: 'taskTemplates', href: '/task-templates', icon: 'task', ready: false },
      { key: 'pregnancyWeeks', href: '/pregnancy-weeks', icon: 'baby', ready: false },
      { key: 'phaseContents', href: '/phase-contents', icon: 'moon', ready: false },
      { key: 'recommendations', href: '/recommendations', icon: 'idea', ready: false },
      { key: 'banners', href: '/banners', icon: 'banner', ready: false },
      { key: 'infoSections', href: '/info-sections', icon: 'lifeRing', ready: false },
    ],
  },
  {
    key: 'groupMessages',
    items: [{ key: 'messages', href: '/messages', icon: 'message', ready: false }],
  },
  {
    key: 'groupSettings',
    items: [
      { key: 'languages', href: '/languages', icon: 'language', role: 'super', ready: false },
      { key: 'uiKit', href: '/ui-kit', icon: 'sparkle', ready: true, devOnly: true },
    ],
  },
  {
    key: 'groupAccount',
    items: [
      { key: 'password', href: '/account/password', icon: 'key', ready: false },
      { key: 'admins', href: '/admins', icon: 'shield', role: 'super', ready: false },
    ],
  },
];

/** Groups visible to `role`; empty groups are dropped. */
export function navFor(role: AdminRole, opts: { dev: boolean }): NavGroup[] {
  return NAV.map((group) => ({
    ...group,
    items: group.items.filter(
      (item) => (!item.role || item.role === role || role === 'super') && (!item.devOnly || opts.dev),
    ),
  })).filter((group) => group.items.length > 0);
}

/** The item whose section contains `pathname` (longest href wins; `/` only exact). */
export function activeItem(pathname: string, groups: readonly NavGroup[] = NAV): NavItem | null {
  let best: NavItem | null = null;
  for (const item of groups.flatMap((g) => g.items)) {
    const match =
      item.href === '/' ? pathname === '/' : pathname === item.href || pathname.startsWith(`${item.href}/`);
    if (match && (!best || item.href.length > best.href.length)) best = item;
  }
  return best;
}
