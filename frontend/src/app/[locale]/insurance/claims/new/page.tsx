import { setRequestLocale } from 'next-intl/server';

import { InsuranceClaimNewPage } from '@/screens/insurance-claim-new';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/insurance/claims/new` — ثبت خسارت (CB-INS-04, nbl_Ins_ClaimNew). */
export default async function InsuranceClaimNewRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="insuranceClaimNew">
      <InsuranceClaimNewPage />
    </RouteMessages>
  );
}
