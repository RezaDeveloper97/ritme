import { setRequestLocale } from 'next-intl/server';

import { JoinPage } from '@/screens/city-services-join';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/city-services/join` — a business applies to be listed in «مادر و کودک» (CB-DIR-10). No bottom nav. */
export default async function CityServicesJoinRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="cityJoin">
      <JoinPage />
    </RouteMessages>
  );
}
