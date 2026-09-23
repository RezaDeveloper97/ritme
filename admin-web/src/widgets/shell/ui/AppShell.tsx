'use client';

import { useTranslations } from 'next-intl';
import { usePathname } from 'next/navigation';
import { useEffect, useState, type ReactNode } from 'react';

import { AuthGate, useCurrentAdmin } from '@/features/auth';

import { activeItem, navFor } from '../model/nav';
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
  const pathname = usePathname();
  const t = useTranslations('nav');
  const [open, setOpen] = useState(false);

  // Escape closes the mobile drawer — never trap the admin inside the menu.
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(false);
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open]);

  if (!admin) return null;
  const groups = navFor(admin.role, { dev: IS_DEV });
  const active = activeItem(pathname, groups);

  return (
    <div className="shell">
      <Sidebar groups={groups} active={active} open={open} onNavigate={() => setOpen(false)} />
      <div className="scrim" data-open={open || undefined} onClick={() => setOpen(false)} aria-hidden="true" />
      <div className="flex min-w-0 flex-col">
        <Header
          title={active ? t(active.key as 'dashboard') : ''}
          admin={admin}
          menuOpen={open}
          onMenu={() => setOpen((v) => !v)}
        />
        <main className="mx-auto w-full max-w-[1200px] flex-1 p-4 sm:p-6">{children}</main>
      </div>
    </div>
  );
}
