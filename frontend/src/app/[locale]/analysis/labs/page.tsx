import { setRequestLocale } from 'next-intl/server';

import { AnalysisLabsPage } from '@/screens/analysis-labs';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/analysis/labs` — lab trends (B-N3-10; an empty state until B-N6-06 brings lab results). */
export default async function AnalysisLabsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="analysis">
      <AnalysisLabsPage />
    </RouteMessages>
  );
}
