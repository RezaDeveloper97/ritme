import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { MenopauseLogPage } from '@/screens/menopause-log';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/menopause/log` — ثبت علائم (CB-MENO-06, nbl_Meno_Log). A form: no bottom nav. */
export default async function MenopauseLogRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  // `?date` is read with useSearchParams, which needs a Suspense boundary.
  return (
    <RouteMessages route="menopauseLog">
      <Suspense>
        <MenopauseLogPage />
      </Suspense>
    </RouteMessages>
  );
}
