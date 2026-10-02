import { setRequestLocale } from 'next-intl/server';

import { PregnancyWeightPage } from '@/screens/analysis-pregnancy';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/analysis/pregnancy-weight` — pregnancy weight gain against the IOM band (B-N3-12, An_PregWeight). */
export default async function AnalysisPregnancyWeightRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="analysisPregnancyWeight">
      <PregnancyWeightPage />
    </RouteMessages>
  );
}
