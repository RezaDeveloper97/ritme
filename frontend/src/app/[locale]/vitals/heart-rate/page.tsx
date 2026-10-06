import { setRequestLocale } from 'next-intl/server';

import { VitalsReportPage } from '@/screens/vitals-report';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/vitals/heart-rate` — heart rate report (no board; follows nbl_Vitals_BPReport). Back header, no bottom nav. */
export default async function VitalsHrReportRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="vitalsReport">
      <VitalsReportPage type="hr" />
    </RouteMessages>
  );
}
