import { setRequestLocale } from 'next-intl/server';

import { RecordSharePage } from '@/screens/record-share';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/record/share` — «چه کسی پرونده‌ات را می‌بیند؟» (CB-REC-05, nbl_Rec_Share). Back header, no bottom nav. */
export default async function RecordShareRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="recordShare">
      <RecordSharePage />
    </RouteMessages>
  );
}
