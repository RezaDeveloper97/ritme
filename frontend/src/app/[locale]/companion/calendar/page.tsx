import { setRequestLocale } from 'next-intl/server';

import { CompanionCalendarPage } from '@/screens/companion-calendar';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/companion/calendar` — a companion (male) account's calendar: his reminders + her likely period days. */
export default async function CompanionCalendarRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="companionCalendar">
      <CompanionCalendarPage />
    </RouteMessages>
  );
}
