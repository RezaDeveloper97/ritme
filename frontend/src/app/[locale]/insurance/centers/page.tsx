import { setRequestLocale } from 'next-intl/server';

import { InsuranceCentersPage } from '@/screens/insurance-centers';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/insurance/centers` — مراکز طرف قرارداد (CB-INS-06, nbl_Ins_Centers). */
export default async function InsuranceCentersRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="insuranceCenters">
      <InsuranceCentersPage />
    </RouteMessages>
  );
}
