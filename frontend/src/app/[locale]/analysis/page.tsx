import { setRequestLocale } from 'next-intl/server';

import { AnalysisPage } from '@/screens/analysis';
import { AnalysisPregnancyHub } from '@/screens/analysis-pregnancy';
import { AnalysisTtcHub } from '@/screens/analysis-ttc';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/analysis` — the analysis hub (B-N3-08, An_Hub; TTC users get An_Hub_TTC, B-N3-11; pregnancy An_Hub_Preg, B-N3-12); lives under the mode tab. */
export default async function AnalysisRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="analysisHub">
      <AnalysisPage ttcHub={<AnalysisTtcHub />} pregnancyHub={<AnalysisPregnancyHub />} />
    </RouteMessages>
  );
}
