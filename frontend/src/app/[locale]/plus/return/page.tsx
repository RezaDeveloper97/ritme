import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { PlusReturnPage } from '@/screens/plus-success';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/plus/return` — gateway return (PLUS_CALLBACK_URL, B-N2-05): verifies, then the success view. */
export default async function PlusReturnRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  // useSearchParams bails out of prerendering unless it sits under a suspense boundary.
  return (
    <RouteMessages route="plusSuccess">
      <Suspense>
        <PlusReturnPage />
      </Suspense>
    </RouteMessages>
  );
}
