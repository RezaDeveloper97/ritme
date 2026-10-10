import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';
import { DoctorBookedPage } from '@/screens/doctor-booked';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/services/bookings/[id]` — booked / booking detail (nbl_v17_Booked, B-N7-05). */
export default async function DoctorBookedPageRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="doctors">
      <Suspense>
        <DoctorBookedPage bookingId={/^\d{1,12}$/.test(id) ? Number(id) : 0} />
      </Suspense>
    </RouteMessages>
  );
}
