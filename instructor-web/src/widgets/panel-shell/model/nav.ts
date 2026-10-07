import type { IconName } from '@/shared/ui';

/** The four panel tabs (nbl_Ins_* bottom nav). Content arrives with B-N8-06 / B-N8-07. */
export const PANEL_NAV = [
  { href: '/', key: 'dashboard', icon: 'dashboard' },
  { href: '/content', key: 'content', icon: 'content' },
  { href: '/groups', key: 'groups', icon: 'groups' },
  { href: '/students', key: 'students', icon: 'students' },
] as const satisfies readonly { href: string; key: string; icon: IconName }[];

export type PanelNavKey = (typeof PANEL_NAV)[number]['key'];

/** Active tab for a pathname: exact for `/`, prefix for the others. */
export function activeNav(pathname: string | null): PanelNavKey {
  const path = pathname ?? '/';
  const hit = PANEL_NAV.find((item) => item.href !== '/' && (path === item.href || path.startsWith(`${item.href}/`)));
  return hit?.key ?? 'dashboard';
}
