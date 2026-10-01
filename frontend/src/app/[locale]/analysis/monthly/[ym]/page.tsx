import { setRequestLocale } from 'next-intl/server';

import { AnalysisMonthlyPage } from '@/screens/analysis-monthly';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; ym: string }>;
  searchParams: Promise<{ calendar?: string | string[] }>;
}

/** `/analysis/monthly/[ym]?calendar=` — the monthly report (B-N3-10). */
export default async function AnalysisMonthlyRoute({ params, searchParams }: Props) {
  const { locale, ym } = await params;
  const { calendar } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="analysis">
      <AnalysisMonthlyPage ym={ym} calendar={typeof calendar === 'string' ? calendar : undefined} />
    </RouteMessages>
  );
}
