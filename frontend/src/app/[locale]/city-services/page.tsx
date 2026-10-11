import { setRequestLocale } from 'next-intl/server';

import { CityServicesPage } from '@/screens/city-services';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/city-services` — «برای مادر و کودک»: classes, pools and playhouses near you (B-N10-05). */
export default async function CityServicesRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="cityServices">
      <CityServicesPage />
    </RouteMessages>
  );
}
