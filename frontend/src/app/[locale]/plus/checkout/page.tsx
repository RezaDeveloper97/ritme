import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { CheckoutPage } from '@/screens/plus-checkout';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/plus/checkout?plan=` — review and pay (B-N2-07, nbl_Prem_Checkout). */
export default async function PlusCheckoutRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  // useSearchParams bails out of prerendering unless it sits under a suspense boundary.
  return (
    <RouteMessages route="plusCheckout">
      <Suspense>
        <CheckoutPage />
      </Suspense>
    </RouteMessages>
  );
}
