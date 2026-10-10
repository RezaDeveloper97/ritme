import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';
import { DoctorReturnPage } from '@/screens/doctor-return';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/services/bookings/return` — gateway return (PAYMENT_CALLBACK, B-N7-05): verifies, then opens the booking. */
export default async function DoctorReturnPageRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="doctors">
      <Suspense>
        <DoctorReturnPage />
      </Suspense>
    </RouteMessages>
  );
}
