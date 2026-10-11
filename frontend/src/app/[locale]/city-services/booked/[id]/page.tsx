import { notFound } from 'next/navigation';
import { setRequestLocale } from 'next-intl/server';

import { BookedPage } from '@/screens/city-services';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/city-services/booked/[id]` — the request just sent / one booking (CB-DIR-09, nbl_Dir_Booked). A flow: no bottom nav. */
export default async function CityServiceBookedRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const n = Number(id);
  if (!Number.isInteger(n) || n <= 0) notFound();
  return (
    <RouteMessages route="cityServices">
      <BookedPage id={n} />
    </RouteMessages>
  );
}
