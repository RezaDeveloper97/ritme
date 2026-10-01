import { setRequestLocale } from 'next-intl/server';

import { AnalysisCyclePage } from '@/screens/analysis-cycle';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/analysis/cycle` — the cycle report (B-N3-09). */
export default async function AnalysisCycleRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="analysisReport">
      <AnalysisCyclePage />
    </RouteMessages>
  );
}
