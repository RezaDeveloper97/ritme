import { setRequestLocale } from 'next-intl/server';

import { AppointmentFormPage } from '@/screens/reminder-appointment-form';

import { RouteMessages } from '../../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

export default async function EditAppointmentRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="reminders">
      <AppointmentFormPage id={Number(id)} />
    </RouteMessages>
  );
}
