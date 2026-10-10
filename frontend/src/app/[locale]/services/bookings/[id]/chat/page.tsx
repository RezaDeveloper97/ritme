import { setRequestLocale } from 'next-intl/server';

import { DoctorChatPage } from '@/screens/doctor-chat';

import { RouteMessages } from '../../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/services/bookings/[id]/chat` — doctor chat (nbl_v17_Chat, B-N7-05). */
export default async function DoctorChatPageRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="doctors">
      <DoctorChatPage bookingId={/^\d{1,12}$/.test(id) ? Number(id) : 0} />
    </RouteMessages>
  );
}
