import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { LogPage } from '@/screens/log';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function LogRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  // LogPage reads `?date` via useSearchParams, which requires a Suspense
  // boundary so the route isn't forced into full client-side rendering.
  return (
    <Suspense>
      <LogPage />
    </Suspense>
  );
}
