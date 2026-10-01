import { setRequestLocale } from 'next-intl/server';

import { AnalysisCorrelationsPage } from '@/screens/analysis-correlations';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/analysis/correlations` — the correlations report (B-N3-09). */
export default async function AnalysisCorrelationsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="analysis">
      <AnalysisCorrelationsPage />
    </RouteMessages>
  );
}
