import { setRequestLocale } from 'next-intl/server';

import { DevicesPage } from '@/screens/devices';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/profile/devices` — B-N10-07 (nbl_Me_Hub). */
export default async function ProfileDevicesRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="profileDevices">
      <DevicesPage />
    </RouteMessages>
  );
}
