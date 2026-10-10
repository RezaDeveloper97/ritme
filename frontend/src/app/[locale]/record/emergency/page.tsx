import { setRequestLocale } from 'next-intl/server';

import { RecordEmergencyPage } from '@/screens/record-emergency';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/record/emergency` — «کارت اضطراری» (CB-REC-05, nbl_Rec_Emergency). Back header, no bottom nav. */
export default async function RecordEmergencyRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="recordEmergency">
      <RecordEmergencyPage />
    </RouteMessages>
  );
}
