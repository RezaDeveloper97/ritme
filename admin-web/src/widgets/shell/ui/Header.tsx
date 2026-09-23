'use client';

import { useTranslations } from 'next-intl';

import { adminInitial, RoleBadge, type Admin } from '@/entities/admin';
import { Button, Icon, ThemeToggle, UiLocaleSelect } from '@/shared/ui';

export function Header({
  title,
  admin,
  menuOpen,
  onMenu,
}: {
  title: string;
  admin: Admin;
  menuOpen: boolean;
  onMenu: () => void;
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
      <h1 className="m-0 min-w-0 flex-1 truncate text-[17px] font-bold">{title}</h1>
      <UiLocaleSelect />
      <ThemeToggle />
      <div className="flex shrink-0 items-center gap-2 border-s border-line ps-3">
        <span className="avatar" aria-hidden="true">
          {adminInitial(admin.name)}
        </span>
        <span className="hidden flex-col leading-tight sm:flex">
          <span className="text-[13px] font-semibold">{admin.name}</span>
          <span className="text-[11.5px] text-muted" dir="ltr">
            {admin.email}
          </span>
        </span>
        <span className="hidden md:inline-flex">
          <RoleBadge role={admin.role} />
        </span>
      </div>
    </header>
  );
}
