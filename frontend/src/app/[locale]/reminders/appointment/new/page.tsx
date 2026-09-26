import { setRequestLocale } from 'next-intl/server';

import { AppointmentFormPage } from '@/screens/reminder-appointment-form';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ kind?: string }>;
}

export default async function NewAppointmentRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { kind } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="reminders">
      <AppointmentFormPage initialKind={kind} />
    </RouteMessages>
  );
}
