import type { Metadata } from 'next';
import { setRequestLocale } from 'next-intl/server';

import { PublicEmergencyPage } from '@/screens/record-emergency';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

export const metadata: Metadata = {
  robots: { index: false, follow: false, nocache: true, googleBot: { index: false, follow: false } },
  referrer: 'no-referrer',
};

export const dynamic = 'force-dynamic';

/**
 * `/emergency` — where `/emergency/[token]` leaves the address bar after the first open (CB-REC-05 L3): a refresh
 * shows «open the link again» instead of the card, and no token stays in the URL.
 */
export default async function PublicEmergencyClosedRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="publicEmergency">
      <PublicEmergencyPage token={null} />
    </RouteMessages>
  );
}
