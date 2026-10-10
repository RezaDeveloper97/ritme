import { setRequestLocale } from 'next-intl/server';

import { DoctorsPage } from '@/screens/doctors';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/services/doctors` — directory (nbl_v17_Doctors, B-N7-05). */
export default async function DoctorsPageRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="doctors">
      <DoctorsPage />
    </RouteMessages>
  );
}
