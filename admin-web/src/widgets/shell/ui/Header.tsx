'use client';

import { useTranslations } from 'next-intl';

import { adminInitial, RoleBadge, type Admin } from '@/entities/admin';
import { Button, Icon, ThemeToggle, UiLocaleSelect } from '@/shared/ui';

export function Header({
  title,
  section,
  admin,
  menuOpen,
  onMenu,
  onSearch,
}: {
  title: string;
  /** The sidebar category of the current page, shown above the title. */
  section?: string;
  admin: Admin;
  menuOpen: boolean;
  onMenu: () => void;
  onSearch: () => void;
}) {
  const t = useTranslations('shell');
  return (
    <header className="topbar">
      <Button
        variant="ghost"
        icon
        className="menu-toggle"
        onClick={onMenu}
        aria-label={menuOpen ? t('closeMenu') : t('menu')}
        aria-expanded={menuOpen}
        aria-controls="admin-sidebar"
      >
        <Icon name="menu" />
      </Button>
      <div className="flex min-w-0 flex-1 flex-col leading-tight">
        {section ? <span className="truncate text-xs text-muted">{section}</span> : null}
        <h1 className="m-0 truncate text-[17px] font-bold">{title}</h1>
      </div>
      <button type="button" className="search-trigger" onClick={onSearch} aria-label={t('search')} aria-keyshortcuts="Control+K Meta+K">
        <Icon name="search" />
        <span className="search-trigger-label">{t('search')}</span>
        <kbd dir="ltr" aria-hidden="true">⌘K</kbd>
      </button>
      <UiLocaleSelect />
      <ThemeToggle />
      <div className="flex shrink-0 items-center gap-2 border-s border-line ps-3">
        <span className="avatar" aria-hidden="true">
          {adminInitial(admin.name)}
        </span>
        <span className="hidden flex-col items-start gap-0.5 leading-tight sm:flex">
          <span className="text-[13px] font-bold">{admin.name}</span>
          <RoleBadge role={admin.role} />
        </span>
      </div>
    </header>
  );
}
