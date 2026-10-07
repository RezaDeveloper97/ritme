import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { RecordExportPage } from '@/screens/record-export';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/**
 * `/record/export` — «گزارش برای پزشک» (bloom B-N6-04, nbl_Record_Export). Back header → /record. `?section=vitals|checkups`
 * preselects a group (B-N6-04b share entry points); it is read with useSearchParams, which needs a Suspense boundary.
 */
export default async function RecordExportRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="recordExport">
      <Suspense>
        <RecordExportPage />
      </Suspense>
    </RouteMessages>
  );
}
