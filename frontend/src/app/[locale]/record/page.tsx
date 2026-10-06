import { setRequestLocale } from 'next-intl/server';

import { HealthRecordPage } from '@/screens/health-record';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/record` — «پرونده سلامت من» (bloom B-N6-03, nbl_Record_Summary). Back header, owner-only; reached from Me. */
export default async function HealthRecordRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="healthRecord">
      <HealthRecordPage />
    </RouteMessages>
  );
}
