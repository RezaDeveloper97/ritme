import { setRequestLocale } from 'next-intl/server';

import { AppointmentFormPage } from '@/screens/reminder-appointment-form';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  // Only the visit kind comes from the URL. Title, topic, date and the
  // care-plan key arrive as a one-time `?prefill=<id>` handoff read on the
  // client — health context never goes in a query string (audit M3-M7 #3).
  searchParams: Promise<{ kind?: string }>;
}

export default async function NewAppointmentRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { kind } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="reminders">
      <AppointmentFormPage prefill={{ kind }} />
    </RouteMessages>
  );
}
