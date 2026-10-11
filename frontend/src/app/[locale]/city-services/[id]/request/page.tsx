import { notFound } from 'next/navigation';
import { setRequestLocale } from 'next-intl/server';

import { CityServiceRequestPage } from '@/screens/city-services';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/city-services/[id]/request` — the booking request form of a request-mode venue (B-N10-05). */
export default async function CityServiceRequestRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const n = Number(id);
  if (!Number.isInteger(n) || n <= 0) notFound();
  return (
    <RouteMessages route="cityServices">
      <CityServiceRequestPage id={n} />
    </RouteMessages>
  );
}
