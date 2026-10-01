import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { PlansPage } from '@/screens/plus-plans';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/plus/plans` — plan choice (B-N2-07, nbl_Prem_Plans). Reads ?plan= via useSearchParams. */
export default async function PlusPlansRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  // useSearchParams bails out of prerendering unless it sits under a suspense boundary.
  return (
    <RouteMessages route="plusPlans">
      <Suspense>
        <PlansPage />
      </Suspense>
    </RouteMessages>
  );
}
