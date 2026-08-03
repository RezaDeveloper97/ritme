import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { CalendarPage } from '@/screens/calendar';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function CalendarRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  // CalendarPage reads ?editDates=1 via useSearchParams, which bails out of
  // prerendering unless it sits under a suspense boundary.
  return (
    <Suspense>
      <CalendarPage />
    </Suspense>
  );
}
