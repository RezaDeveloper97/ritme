import { setRequestLocale } from 'next-intl/server';

import { VitalsReportPage } from '@/screens/vitals-report';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/vitals/glucose` — blood glucose report (nbl_Vitals_GlucoseReport). Back header, no bottom nav. */
export default async function VitalsGlucoseReportRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="vitalsReport">
      <VitalsReportPage type="glucose" />
    </RouteMessages>
  );
}
