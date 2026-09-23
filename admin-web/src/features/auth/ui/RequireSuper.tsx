'use client';

import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { Icon, Panel } from '@/shared/ui';

import { useCurrentAdmin } from '../api/queries';

/**
 * Super-admin-only screens (admins, languages). The sidebar already hides them
 * from editors and the API answers 403; this covers a typed-in URL.
 */
export function RequireSuper({ children }: { children: ReactNode }) {
  const t = useTranslations('errors');
  const admin = useCurrentAdmin();
  if (admin?.role === 'super') return <>{children}</>;
  return (
    <Panel>
      <div role="alert" className="flex flex-col items-center gap-3 px-4 py-10 text-center text-ink-3">
        <Icon name="shield" size={22} className="text-muted" />
        <p className="m-0">{t('forbidden')}</p>
      </div>
    </Panel>
  );
}
