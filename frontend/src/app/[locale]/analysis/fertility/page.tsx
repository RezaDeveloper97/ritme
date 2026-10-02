import { setRequestLocale } from 'next-intl/server';

import { AnalysisFertilityPage } from '@/screens/analysis-ttc';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/analysis/fertility` — BBT and ovulation of one cycle (B-N3-11, An_Fertility). */
export default async function AnalysisFertilityRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="analysis">
      <AnalysisFertilityPage />
    </RouteMessages>
  );
}
