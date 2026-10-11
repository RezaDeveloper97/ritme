import { setRequestLocale } from 'next-intl/server';

import { InsuranceClaimPage } from '@/screens/insurance-claim';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/insurance/claims/{id}` — جزئیات خسارت (CB-INS-04, nbl_Ins_ClaimDetail). */
export default async function InsuranceClaimRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="insuranceClaim">
      <InsuranceClaimPage id={Number(id)} />
    </RouteMessages>
  );
}
