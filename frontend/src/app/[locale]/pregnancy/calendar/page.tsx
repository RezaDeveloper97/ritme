import { setRequestLocale } from 'next-intl/server';

import { PregnancyCalendarPage } from '@/screens/pregnancy-calendar';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/pregnancy/calendar` — pregnancy calendar, visits and care plan (T-M7-13). */
export default async function PregnancyCalendarRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="pregnancyCalendar">
      <PregnancyCalendarPage />
    </RouteMessages>
  );
}
