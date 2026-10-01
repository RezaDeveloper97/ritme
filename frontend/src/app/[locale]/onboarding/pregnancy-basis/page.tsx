import { setRequestLocale } from 'next-intl/server';

import { PregnancyStep } from '@/screens/onboarding-flow';

import { RouteMessages } from '../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

/** `/onboarding/pregnancy-basis` — nbl_Onb_Preg (B-N2-02). */
export default async function PregnancyBasisRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingPregnancyBasis">
      <PregnancyStep />
    </RouteMessages>
  );
}
