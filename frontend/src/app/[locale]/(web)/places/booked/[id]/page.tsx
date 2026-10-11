import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { setRequestLocale } from 'next-intl/server';

import { WebBookedPage } from '@/screens/web-city-services';

import { RouteMessages } from '../../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** A booking is private: never indexed, no referrer. */
export const metadata: Metadata = {
  robots: { index: false, follow: false, nocache: true },
  referrer: 'no-referrer',
};

/**
 * `/places/booked/[id]` — the booking confirmation on the web (CB-DIR-11). `places` is a public segment, so this route
 * has no middleware guard: the page itself asks a visitor without a session to log in.
 */
export default async function PlaceBookedRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const n = Number(id);
  if (!Number.isInteger(n) || n <= 0) notFound();
  return (
    <RouteMessages route="webDirectory">
      <WebBookedPage id={n} />
    </RouteMessages>
  );
}
