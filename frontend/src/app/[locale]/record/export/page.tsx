import { setRequestLocale } from 'next-intl/server';

import { RecordExportPage } from '@/screens/record-export';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/record/export` — «گزارش برای پزشک» (bloom B-N6-04, nbl_Record_Export). Back header → /record. */
export default async function RecordExportRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="recordExport">
      <RecordExportPage />
    </RouteMessages>
  );
}
