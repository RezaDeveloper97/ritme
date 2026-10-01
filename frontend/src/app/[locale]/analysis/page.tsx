import { setRequestLocale } from 'next-intl/server';

import { AnalysisPage } from '@/screens/analysis';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/analysis` — the analysis hub (B-N3-08, An_Hub); lives under the mode tab. */
export default async function AnalysisRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="analysis">
      <AnalysisPage />
    </RouteMessages>
  );
}
