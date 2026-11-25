'use client';

import { useTranslations } from 'next-intl';
import { usePathname } from 'next/navigation';
import { useEffect, useState, type ReactNode } from 'react';

import { AuthGate, useCurrentAdmin, useCurrentPermissions } from '@/features/auth';

import { activeItem, groupOf, navFor } from '../model/nav';
import { CommandPalette } from './CommandPalette';
import { Header } from './Header';
import { Sidebar } from './Sidebar';

const IS_DEV = process.env.NODE_ENV !== 'production';

/** Sidebar + header around every signed-in screen (mounted by app/(panel)/layout). */
export function AppShell({ children }: { children: ReactNode }) {
  return (
    <AuthGate>
      <ShellFrame>{children}</ShellFrame>
    </AuthGate>
  );
}

function ShellFrame({ children }: { children: ReactNode }) {
  const admin = useCurrentAdmin();
  const permissions = useCurrentPermissions();
  const pathname = usePathname();
  const t = useTranslations('nav');
  const [open, setOpen] = useState(false);
  const [paletteOpen, setPaletteOpen] = useState(false);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setPaletteOpen((v) => !v);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  // Escape closes the mobile drawer — never trap the admin inside the menu.
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(false);
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open]);

  if (!admin) return null;
  const groups = navFor(admin.role, { dev: IS_DEV, permissions });
  const active = activeItem(pathname, groups);
  const group = groupOf(active, groups);

  return (
    <div className="shell">
      <Sidebar groups={groups} active={active} open={open} onNavigate={() => setOpen(false)} />
      <div className="scrim" data-open={open || undefined} onClick={() => setOpen(false)} aria-hidden="true" />
      <div className="flex min-w-0 flex-col">
        <Header
          title={active ? t(active.key as 'dashboard') : ''}
          section={group ? t(group.key as 'dashboard') : ''}
          admin={admin}
          menuOpen={open}
          onMenu={() => setOpen((v) => !v)}
          onSearch={() => setPaletteOpen(true)}
        />
        <main className="mx-auto w-full max-w-[1200px] flex-1 p-4 sm:p-6">{children}</main>
      </div>
      <CommandPalette groups={groups} open={paletteOpen} onClose={() => setPaletteOpen(false)} />
    </div>
  );
}
