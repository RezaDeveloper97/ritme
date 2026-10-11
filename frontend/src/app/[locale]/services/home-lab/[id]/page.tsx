import { notFound } from 'next/navigation';
import { setRequestLocale } from 'next-intl/server';

import { HomeLabRequestPage } from '@/screens/home-lab';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/services/home-lab/[id]` — status, cancel and the consented result import of one request (B-N10-04). */
export default async function HomeLabRequestRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const n = Number(id);
  if (!Number.isInteger(n) || n <= 0) notFound();
  return (
    <RouteMessages route="homeLab">
      <HomeLabRequestPage id={n} />
    </RouteMessages>
  );
}
