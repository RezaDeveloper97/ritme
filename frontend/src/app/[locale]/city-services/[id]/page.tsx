import { notFound } from 'next/navigation';
import { setRequestLocale } from 'next-intl/server';

import { CityServiceDetailPage } from '@/screens/city-services';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/city-services/[id]` — one venue and its booking action (B-N10-05). */
export default async function CityServiceDetailRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const n = Number(id);
  if (!Number.isInteger(n) || n <= 0) notFound();
  return (
    <RouteMessages route="cityServices">
      <CityServiceDetailPage id={n} />
    </RouteMessages>
  );
}
