import { setRequestLocale } from 'next-intl/server';

import { DirectoryMapPage } from '@/screens/city-services';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/city-services/map` — «مادر و کودک» on the map (CB-DIR-07, nbl_Dir_Map). Falls back to a list without a map key. */
export default async function CityServicesMapRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="cityServices">
      <DirectoryMapPage />
    </RouteMessages>
  );
}
