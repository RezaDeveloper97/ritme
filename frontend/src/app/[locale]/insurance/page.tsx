import { setRequestLocale } from 'next-intl/server';

import { InsurancePage } from '@/screens/insurance';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/insurance` — بیمه: admin-edited info page (B-N10-06), entry from the services hub tile. */
export default async function InsuranceRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="insurance">
      <InsurancePage />
    </RouteMessages>
  );
}
