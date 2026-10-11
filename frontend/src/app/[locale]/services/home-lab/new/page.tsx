import { setRequestLocale } from 'next-intl/server';

import { HomeLabFormPage } from '@/screens/home-lab';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ service?: string }>;
}

/** `/services/home-lab/new?service=ID` — the booking form (B-N10-04). */
export default async function HomeLabFormRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { service } = await searchParams;
  setRequestLocale(locale);
  const id = Number(service);
  return (
    <RouteMessages route="homeLab">
      <HomeLabFormPage serviceId={Number.isInteger(id) && id > 0 ? id : null} />
    </RouteMessages>
  );
}
