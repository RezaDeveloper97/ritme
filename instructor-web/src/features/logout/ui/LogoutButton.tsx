'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { Button, Icon, IconButton } from '@/shared/ui';

import { logout } from '../api/logout';

/** `icon` = 44px round header button; `text` = a ghost pill for state screens. */
export function LogoutButton({ variant = 'icon' }: { variant?: 'icon' | 'text' }) {
  const t = useTranslations('common');
  const queryClient = useQueryClient();
  const [busy, setBusy] = useState(false);
  const run = async () => {
    setBusy(true);
    await logout();
    queryClient.clear();
    // Full document replace: nothing of the old session survives in memory.
    window.location.replace('/login');
  };
  if (variant === 'text') {
    return (
      <Button variant="ghost" block loading={busy} onClick={run}>
        {t('logout')}
      </Button>
    );
  }
  return (
    <IconButton label={t('logout')} onClick={run} disabled={busy}>
      <Icon name="logout" />
    </IconButton>
  );
}
