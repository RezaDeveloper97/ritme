import { notFound } from 'next/navigation';
import { setRequestLocale } from 'next-intl/server';

import { CityServiceRequestPage } from '@/screens/city-services';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
  searchParams: Promise<{ service?: string; slot?: string; reschedule?: string }>;
}

/** A positive integer query param, else null (CB-DIR-09). */
const posInt = (v: string | undefined): number | null => {
  const n = Number(v);
  return Number.isInteger(n) && n > 0 ? n : null;
};

/** `/city-services/[id]/request` — the booking request form of a request-mode venue (B-N10-05). */
export default async function CityServiceRequestRoute({ params, searchParams }: Props) {
  const { locale, id } = await params;
  const q = await searchParams;
  setRequestLocale(locale);
  const n = Number(id);
  if (!Number.isInteger(n) || n <= 0) notFound();
  return (
    <RouteMessages route="cityServices">
      <CityServiceRequestPage id={n} service={posInt(q.service)} slot={posInt(q.slot)} reschedule={posInt(q.reschedule)} />
    </RouteMessages>
  );
}
