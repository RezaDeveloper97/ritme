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
  const dark = mounted && theme === 'dark';
  const label = dark ? t('themeLight') : t('themeDark');
  return (
    <Button variant="ghost" icon onClick={toggle} aria-label={label} title={label}>
      <Icon name={dark ? 'sun' : 'darkMode'} />
    </Button>
  );
}
