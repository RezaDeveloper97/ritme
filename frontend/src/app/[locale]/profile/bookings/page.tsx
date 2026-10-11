import { setRequestLocale } from 'next-intl/server';

import { BookingsPage } from '@/screens/city-services';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/profile/bookings` — «رزروها»: my city-service requests (B-N10-05; CB-DIR-09 supersedes it later). */
export default async function ProfileBookingsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="cityServices">
      <BookingsPage />
    </RouteMessages>
  );
}
