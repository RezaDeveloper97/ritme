import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';
import { DoctorBookPage } from '@/screens/doctor-book';

import { RouteMessages } from '../../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/services/doctors/[id]/book` — review & book (nbl_v17_ReviewBook, B-N7-05). */
export default async function DoctorBookPageRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="doctors">
      <Suspense>
        <DoctorBookPage doctorId={/^\d{1,12}$/.test(id) ? Number(id) : 0} />
      </Suspense>
    </RouteMessages>
  );
}
