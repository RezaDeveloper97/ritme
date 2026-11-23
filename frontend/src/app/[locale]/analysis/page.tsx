import { setRequestLocale } from 'next-intl/server';

import { AnalysisPage, analysisTabOf } from '@/screens/analysis';
import { AnalysisPostpartumHub } from '@/screens/analysis-postpartum';
import { AnalysisPregnancyHub } from '@/screens/analysis-pregnancy';
import { AnalysisTtcHub } from '@/screens/analysis-ttc';
import { CycleHistoryPanel } from '@/screens/cycle';
import { CyclePredictions } from '@/widgets/cycle-predictions';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ tab?: string }>;
}

/**
 * `/analysis` — the «تحلیل» bottom-nav tab: the analysis hub (B-N3-08, An_Hub; TTC An_Hub_TTC, B-N3-11; pregnancy
 * An_Hub_Preg, B-N3-12; postpartum An_Hub_Post, B-N5-07) plus the cycle history as its «تاریخچه» tab (`?tab=history`).
 */
export default async function AnalysisRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { tab } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="analysisHub">
      <AnalysisPage
        ttcHub={<AnalysisTtcHub predictions={<CyclePredictions />} />}
        pregnancyHub={<AnalysisPregnancyHub />}
        postpartumHub={<AnalysisPostpartumHub />}
        historyPanel={<CycleHistoryPanel />}
        predictions={<CyclePredictions />}
        initialTab={analysisTabOf(tab)}
      />
    </RouteMessages>
  );
}
