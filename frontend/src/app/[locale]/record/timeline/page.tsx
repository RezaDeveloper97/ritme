import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { RecordTimelinePage } from '@/screens/record-timeline';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/record/timeline` — «سوابق و اسناد» (CB-REC-04, nbl_Rec_Timeline). Back header, no bottom nav. */
export default async function RecordTimelineRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  // `?kind` (the filter chip, never health data) is read with useSearchParams, which needs a Suspense boundary.
  return (
    <RouteMessages route="recordTimeline">
      <Suspense>
        <RecordTimelinePage />
      </Suspense>
    </RouteMessages>
  );
}
