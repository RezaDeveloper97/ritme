import type { AdminRole } from '@/entities/admin';
import type { IconName } from '@/shared/ui';

/**
 * The sidebar, mirroring the Blade panel's sections (backend/resources/views/
 * admin/layouts/app.blade.php). `ready: true` items render disabled until
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
      { key: 'users', href: '/users', icon: 'users', ready: true },
      // Any active admin, like /users (B-N1-12b).
      { key: 'supportReports', href: '/support-reports', icon: 'inbox', ready: true },
    ],
  },
  {
    key: 'groupContent',
    items: [
      { key: 'articles', href: '/articles', icon: 'article', ready: true },
      { key: 'affirmations', href: '/affirmations', icon: 'sparkle', ready: true },
      { key: 'challenges', href: '/challenges', icon: 'flag', ready: true },
      { key: 'challengeCompletions', href: '/challenge-completions', icon: 'check', ready: true },
      { key: 'taskTemplates', href: '/task-templates', icon: 'task', ready: true },
      { key: 'phaseContents', href: '/phase-contents', icon: 'moon', ready: true },
      { key: 'recommendations', href: '/recommendations', icon: 'idea', ready: true },
      { key: 'checkupTypes', href: '/checkup-types', icon: 'shieldCheck', ready: true },
      { key: 'banners', href: '/banners', icon: 'banner', ready: true },
      { key: 'infoSections', href: '/info-sections', icon: 'lifeRing', ready: true },
      // Editor + super (the content permission); switch to B-N9-02's content role once it lands.
      { key: 'catalog', href: '/catalog', icon: 'listBullet', ready: true },
    ],
  },
  {
    key: 'groupPregnancy',
    items: [
      { key: 'pregnancyWeeks', href: '/pregnancy-weeks', icon: 'baby', ready: true },
      { key: 'pregnancyCarePlan', href: '/pregnancy-care-plan', icon: 'calendar', ready: true },
      { key: 'pregnancyAlertRules', href: '/pregnancy-alert-rules', icon: 'alert', ready: true },
    ],
  },
  {
    key: 'groupMessages',
    items: [{ key: 'messages', href: '/messages', icon: 'message', ready: true }],
  },
  {
    key: 'groupSettings',
    items: [
      { key: 'languages', href: '/languages', icon: 'language', role: 'super', ready: true },
      { key: 'uiKit', href: '/ui-kit', icon: 'sparkle', ready: true, devOnly: true },
    ],
  },
  {
    key: 'groupAccount',
    items: [
      { key: 'password', href: '/account/password', icon: 'key', ready: true },
      { key: 'admins', href: '/admins', icon: 'shield', role: 'super', ready: true },
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
