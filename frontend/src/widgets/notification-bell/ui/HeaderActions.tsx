'use client';

import { useTranslations } from 'next-intl';

import { useMounted } from '@/shared/lib/use-mounted';
import { useThemeStore } from '@/shared/theme';
import { Icon } from '@/shared/ui';

import { NotificationBell } from './NotificationBell';

/** One-tap light/dark switch; the icon waits for mount so the server render can't disagree with the stored theme. */
export function ThemeToggle({ className }: { className: string }) {
  const t = useTranslations('home');
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);
  const mounted = useMounted();
  const dark = mounted && theme === 'dark';
  return (
    <button
      type="button"
      className={className}
      onClick={() => setTheme(dark ? 'light' : 'dark')}
      aria-label={dark ? t('header.themeLight') : t('header.themeDark')}
    >
      <Icon name={dark ? 'sun' : 'moon'} size={20} strokeWidth={1.8} />
    </button>
  );
}

/**
 * The only two actions of every mode's Today header: notifications and the light/dark switch.
 * `className` is the host header's button style (`home-hdr-btn` or `nb-hbtn is-soft`).
 */
export function HeaderActions({ className }: { className: string }) {
  return (
    <>
      <NotificationBell className={className} />
      <ThemeToggle className={className} />
    </>
  );
}
