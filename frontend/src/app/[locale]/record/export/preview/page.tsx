import { setRequestLocale } from 'next-intl/server';

import { RecordPreviewPage } from '@/screens/record-export';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/record/export/preview` — «پیش‌نمایش» (bloom B-N6-04, nbl_Record_Preview). Close → /record/export. */
export default async function RecordPreviewRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="recordExport">
      <RecordPreviewPage />
    </RouteMessages>
  );
}
