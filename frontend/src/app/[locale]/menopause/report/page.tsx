import { setRequestLocale } from 'next-intl/server';

import { MenopauseReportPage } from '@/screens/record-export';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ from?: string }>;
}

/**
 * `/menopause/report` — گزارش برای پزشک (CB-MENO-11, nbl_Meno_Report) on bloom's report builder. Back header (→ the
 * entry: home, or `/menopause/alert` with `?from=alert`): no bottom nav.
 */
export default async function MenopauseReportRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { from } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="menopauseReport">
      <MenopauseReportPage from={from === 'alert' ? 'alert' : 'home'} />
    </RouteMessages>
  );
}
