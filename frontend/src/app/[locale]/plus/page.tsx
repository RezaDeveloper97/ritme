import { setRequestLocale } from 'next-intl/server';

import { PaywallPage } from '@/screens/plus-paywall';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/plus` — ریتمی پلاس paywall (B-N2-07, nbl_Prem_Paywall). */
export default async function PlusRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="plusPaywall">
      <PaywallPage />
    </RouteMessages>
  );
}
