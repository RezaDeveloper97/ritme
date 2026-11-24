'use client';

import { useTranslations } from 'next-intl';
import Link from 'next/link';
import { useState } from 'react';

import { LogoutButton } from '@/features/auth';
import { Icon } from '@/shared/ui';

import { groupOf, type NavGroup, type NavItem } from '../model/nav';

/**
 * Every category is a titled, collapsible block; the one holding the current page opens by itself. The filter on top
 * narrows the whole menu (category names included) and opens every category that still has a match.
 */
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
  const [filter, setFilter] = useState('');
  const activeGroup = groupOf(active, groups);
  const q = filter.trim().toLowerCase();
  const label = (key: string) => t(key as 'dashboard');

  const visible = q
    ? groups
        .map((g) =>
          label(g.key).toLowerCase().includes(q)
            ? g
            : { ...g, items: g.items.filter((i) => label(i.key).toLowerCase().includes(q)) },
        )
        .filter((g) => g.items.length > 0)
    : groups;

  const renderItem = (item: NavItem) =>
    item.ready ? (
      <Link
        key={item.key}
        href={item.href}
        className="nav-link"
        aria-current={active?.key === item.key ? 'page' : undefined}
        onClick={onNavigate}
      >
        <Icon name={item.icon} />
        <span className="truncate">{label(item.key)}</span>
      </Link>
    ) : (
      <span key={item.key} className="nav-link" aria-disabled="true" title={ts('soon')}>
        <Icon name={item.icon} />
        <span className="truncate">{label(item.key)}</span>
        <span className="sr-only">({ts('soon')})</span>
      </span>
    );

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
      <label className="nav-filter">
        <Icon name="search" />
        <span className="sr-only">{ts('menuFilter')}</span>
        <input
          type="search"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder={ts('menuFilter')}
          onKeyDown={(e) => e.key === 'Escape' && setFilter('')}
        />
      </label>
      <nav className="flex flex-1 flex-col gap-0.5">
        {visible.length === 0 ? <p className="nav-empty">{ts('menuFilterEmpty')}</p> : null}
        {visible.map((group) => (
          <details
            // Re-mount on filter / page change so the open state follows them; a manual toggle sticks until then.
            key={`${group.key}:${q ? 'q' : activeGroup?.key ?? ''}`}
            className="nav-details"
            open={Boolean(q) || group.key === activeGroup?.key || undefined}
          >
            <summary className="nav-summary" data-active={group.key === activeGroup?.key || undefined}>
              <span className="flex min-w-0 items-center gap-2">
                <Icon name={group.icon} />
                <span className="truncate">{label(group.key)}</span>
              </span>
              <Icon name="chevronDown" />
            </summary>
            <div className="nav-items">{group.items.map(renderItem)}</div>
          </details>
        ))}
      </nav>
      <div className="mt-3 border-t border-line pt-2">
        <LogoutButton />
      </div>
    </aside>
  );
}
