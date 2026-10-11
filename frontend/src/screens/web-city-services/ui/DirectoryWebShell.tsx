'use client';

import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { WebShell } from '@/widgets/web-shell';

import { PLACES_HOME } from '../model/place';

/** The directory's configuration of the generic web shell: one nav item, the «ثبت مجموعه» CTA, the app fallback. */
export function DirectoryWebShell({ appHref, children }: { appHref: string; children: ReactNode }) {
  const t = useTranslations('webShell');
  return (
    <WebShell
      nav={[{ href: PLACES_HOME, label: t('nav.services') }]}
      cta={{ href: '/city-services/join', label: t('registerPlace') }}
      appHref={appHref}
      homeHref={PLACES_HOME}
    >
      {children}
    </WebShell>
  );
}
