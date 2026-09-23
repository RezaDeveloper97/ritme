'use client';

import { useTranslations } from 'next-intl';
import Link from 'next/link';

import { LogoutButton } from '@/features/auth';
import { Icon } from '@/shared/ui';

import type { NavGroup, NavItem } from '../model/nav';

export function Sidebar({
  groups,
  active,
  open,
  onNavigate,
}: {
  groups: readonly NavGroup[];
  active: NavItem | null;
  open: boolean;
  onNavigate: () => void;
}) {
  const t = useTranslations('nav');
  const tc = useTranslations('common');
  const ts = useTranslations('shell');
  return (
    <aside id="admin-sidebar" className="sidebar" data-open={open || undefined} aria-label={tc('adminPanel')}>
      <div className="flex items-center gap-2.5 px-2 pb-2">
        <span className="brand-mark" aria-hidden="true">
          {tc('appName').slice(0, 1)}
        </span>
        <span className="flex flex-col leading-tight">
          <strong className="text-[15px]">{tc('appName')}</strong>
          <span className="text-xs text-muted">{tc('adminPanel')}</span>
        </span>
      </div>
      <nav className="flex flex-1 flex-col">
        {groups.map((group) => (
          <div key={group.key} className="flex flex-col gap-0.5">
            <div className="nav-group">{t(group.key as 'groupGeneral')}</div>
            {group.items.map((item) => {
              const label = t(item.key as 'dashboard');
              return item.ready ? (
                <Link
                  key={item.key}
                  href={item.href}
                  className="nav-link"
                  aria-current={active?.key === item.key ? 'page' : undefined}
                  onClick={onNavigate}
                >
                  <Icon name={item.icon} />
                  <span className="truncate">{label}</span>
                </Link>
              ) : (
                <span key={item.key} className="nav-link" aria-disabled="true" title={ts('soon')}>
                  <Icon name={item.icon} />
                  <span className="truncate">{label}</span>
                  <span className="sr-only">({ts('soon')})</span>
                </span>
              );
            })}
          </div>
        ))}
      </nav>
      <div className="mt-3 border-t border-line pt-2">
        <LogoutButton />
      </div>
    </aside>
  );
}
