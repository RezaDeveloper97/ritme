import { setRequestLocale } from 'next-intl/server';

import { AppointmentDetailPage } from '@/screens/reminder-appointment-detail';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

export default async function AppointmentDetailRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="reminders">
      <AppointmentDetailPage id={Number(id)} />
    </RouteMessages>
  );
}
