import { setRequestLocale } from 'next-intl/server';

import { DoctorProfilePage } from '@/screens/doctor-profile';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/services/doctors/[id]` — profile + slots (nbl_v17_DoctorProfile, B-N7-05). */
export default async function DoctorProfilePageRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="doctors">
      <DoctorProfilePage doctorId={/^\d{1,12}$/.test(id) ? Number(id) : 0} />
    </RouteMessages>
  );
}
