'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { resolveTheme, useThemeStore } from '@/shared/theme';

import { IconButton } from './Button';
import { Icon } from './Icon';

/** Round header button: flips light ↔ dark (what is painted now). */
export function ThemeToggle() {
  const t = useTranslations('common');
  const preference = useThemeStore((s) => s.preference);
  const toggle = useThemeStore((s) => s.toggle);
  // The server cannot know the theme; render the moon until mounted.
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  const dark = mounted && resolveTheme(preference) === 'dark';
  return (
    <IconButton label={dark ? t('themeLight') : t('themeDark')} onClick={toggle}>
      <Icon name={dark ? 'sun' : 'moon'} />
    </IconButton>
  );
}
