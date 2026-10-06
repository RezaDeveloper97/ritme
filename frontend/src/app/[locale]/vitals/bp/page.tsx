import { setRequestLocale } from 'next-intl/server';

import { VitalsReportPage } from '@/screens/vitals-report';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/vitals/bp` — blood pressure report (nbl_Vitals_BPReport). Back header, no bottom nav. */
export default async function VitalsBpReportRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="vitalsReport">
      <VitalsReportPage type="bp" />
    </RouteMessages>
  );
}
