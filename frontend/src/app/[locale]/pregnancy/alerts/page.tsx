import { setRequestLocale } from 'next-intl/server';

import { PregnancyAlertsPage } from '@/screens/pregnancy-alerts';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/pregnancy/alerts` — the v2 Alerts screen (last 7 days + level legend). */
export default async function PregnancyAlertsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="pregnancyAlerts">
      <PregnancyAlertsPage />
    </RouteMessages>
  );
}
