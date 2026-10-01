import { setRequestLocale } from 'next-intl/server';

import { AnalysisPeriodPage } from '@/screens/analysis-period';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/analysis/period` — the period report (B-N3-09). */
export default async function AnalysisPeriodRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="analysisReport">
      <AnalysisPeriodPage />
    </RouteMessages>
  );
}
