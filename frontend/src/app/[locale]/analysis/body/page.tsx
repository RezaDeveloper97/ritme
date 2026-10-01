import { setRequestLocale } from 'next-intl/server';

import { AnalysisBodyPage } from '@/screens/analysis-body';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/analysis/body` — the body report (B-N3-09). */
export default async function AnalysisBodyRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="analysisReport">
      <AnalysisBodyPage />
    </RouteMessages>
  );
}
