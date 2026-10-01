import { setRequestLocale } from 'next-intl/server';

import { PartnerStep } from '@/screens/onboarding-flow';

import { RouteMessages } from '../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

/** `/onboarding/partner` — nbl_Onb_Partner, stub until B-N4-05 (B-N2-02). */
export default async function PartnerRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingPartner">
      <PartnerStep />
    </RouteMessages>
  );
}
