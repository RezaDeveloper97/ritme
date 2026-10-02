import { setRequestLocale } from 'next-intl/server';

import { PartnerLinkedStep } from '@/screens/onboarding-flow';

import { RouteMessages } from '../../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

/** `/onboarding/partner/linked` — nbl_Onb_PartnerLinked (B-N4-05). */
export default async function PartnerLinkedRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingPartnerLinked">
      <PartnerLinkedStep />
    </RouteMessages>
  );
}
