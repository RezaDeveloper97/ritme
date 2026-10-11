import { setRequestLocale } from 'next-intl/server';

import { HomeLabPage } from '@/screens/home-lab';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/services/home-lab` — home lab test and checkup requests (B-N10-04). */
export default async function HomeLabRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="homeLab">
      <HomeLabPage />
    </RouteMessages>
  );
}
