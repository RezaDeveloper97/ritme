import type { Metadata } from 'next';
import { setRequestLocale } from 'next-intl/server';

import { PublicEmergencyPage } from '@/screens/record-emergency';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; token: string }>;
}

/** Never indexed or cached; no referrer leaves the page (the token is in the URL). */
export const metadata: Metadata = {
  robots: { index: false, follow: false, nocache: true, googleBot: { index: false, follow: false } },
  referrer: 'no-referrer',
};

export const dynamic = 'force-dynamic';

/** `/emergency/[token]` — the public minimal emergency card (CB-REC-05, D-72). No session, no nav. */
export default async function PublicEmergencyRoute({ params }: Props) {
  const { locale, token } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="publicEmergency">
      <PublicEmergencyPage token={token} scrubTo={`/${locale}/emergency`} />
    </RouteMessages>
  );
}
