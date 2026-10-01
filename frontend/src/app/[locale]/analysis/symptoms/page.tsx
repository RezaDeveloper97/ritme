import { setRequestLocale } from 'next-intl/server';

import { AnalysisSymptomsPage } from '@/screens/analysis-symptoms';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/analysis/symptoms` — the symptoms report (B-N3-09). */
export default async function AnalysisSymptomsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="analysisReport">
      <AnalysisSymptomsPage />
    </RouteMessages>
  );
}
