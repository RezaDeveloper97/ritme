import { setRequestLocale } from 'next-intl/server';

import { InsuranceClaimsPage } from '@/screens/insurance-claims';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/insurance/claims` — سوابق خسارت (CB-INS-04, nbl_Ins_Claims). */
export default async function InsuranceClaimsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="insuranceClaims">
      <InsuranceClaimsPage />
    </RouteMessages>
  );
}
