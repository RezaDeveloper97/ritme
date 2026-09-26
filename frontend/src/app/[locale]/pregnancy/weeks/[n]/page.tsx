import { setRequestLocale } from 'next-intl/server';

import { PregnancyWeekPage } from '@/screens/pregnancy-week';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; n: string }>;
}

export default async function PregnancyWeekRoute({ params }: Props) {
  const { locale, n } = await params;
  setRequestLocale(locale);
  const week = Number.parseInt(n, 10);
  return (
    <RouteMessages route="pregnancyWeek">
      <PregnancyWeekPage week={Number.isFinite(week) ? week : null} />
    </RouteMessages>
  );
}
