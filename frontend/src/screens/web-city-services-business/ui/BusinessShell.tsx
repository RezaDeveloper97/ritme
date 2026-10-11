'use client';

import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { WebShell } from '@/widgets/web-shell';

/** The for-businesses configuration of the generic web shell (same header nav as the directory, the CTA starts the join). */
export function BusinessShell({ appHref, children }: { appHref: string; children: ReactNode }) {
  const t = useTranslations('webShell');
  return (
    <WebShell
      nav={[
        { href: '/places', label: t('nav.services') },
        { href: '/business', label: t('nav.business') },
      ]}
      cta={{ href: '/business/join', label: t('registerPlace') }}
      appHref={appHref}
      homeHref="/places"
      footerLinks={[{ href: '/business', label: t('nav.business') }]}
    >
      {children}
    </WebShell>
  );
}
