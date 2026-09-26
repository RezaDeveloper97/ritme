import { setRequestLocale } from 'next-intl/server';

import { PregnancyWeekPage } from '@/screens/pregnancy-week';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/pregnancy/weeks` — the Week screen on the current week. */
export default async function PregnancyWeeksRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="pregnancyWeek">
      <PregnancyWeekPage week={null} />
    </RouteMessages>
  );
}
