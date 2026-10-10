'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { useThemeStore } from '@/shared/theme';

import { Button } from './Button';
import { Icon } from './Icon';

export function ThemeToggle() {
  const t = useTranslations('shell');
  const theme = useThemeStore((s) => s.theme);
  const toggle = useThemeStore((s) => s.toggle);
  // The stored theme is only known on the client; show its state after mount.
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  const current = mounted ? theme : 'light';
  // The button names what the NEXT click selects (light → dark → system → light).
  const label = current === 'light' ? t('themeDark') : current === 'dark' ? t('themeSystem') : t('themeLight');
  const icon = current === 'light' ? 'sun' : current === 'dark' ? 'darkMode' : 'system';
  return (
    <Button variant="ghost" icon onClick={toggle} aria-label={label} title={label}>
      <Icon name={icon} />
    </Button>
  );
}
