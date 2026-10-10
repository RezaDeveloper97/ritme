import { setRequestLocale } from 'next-intl/server';

import { DoctorBookingsPage } from '@/screens/doctor-bookings';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/services/bookings` — my bookings (B-N7-05). */
export default async function DoctorBookingsPageRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="doctors">
      <DoctorBookingsPage />
    </RouteMessages>
  );
}
