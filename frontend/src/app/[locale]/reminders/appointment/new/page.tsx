import { setRequestLocale } from 'next-intl/server';

import { AppointmentFormPage } from '@/screens/reminder-appointment-form';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ kind?: string; title?: string; date?: string; care_item_key?: string }>;
}

export default async function NewAppointmentRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { kind, title, date, care_item_key: careItemKey } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="reminders">
      <AppointmentFormPage prefill={{ kind, title, date, careItemKey }} />
    </RouteMessages>
  );
}
