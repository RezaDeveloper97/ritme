import { setRequestLocale } from 'next-intl/server';

import { VitalsHubPage } from '@/screens/vitals-hub';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/vitals` — «علائم حیاتی» hub (nbl_Vitals_Hub). A hub: bottom nav (خدمات). */
export default async function VitalsHubRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="vitals">
      <VitalsHubPage />
    </RouteMessages>
  );
}
