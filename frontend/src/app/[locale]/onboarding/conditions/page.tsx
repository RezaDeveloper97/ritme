import { setRequestLocale } from 'next-intl/server';

import { ConditionsStep } from '@/screens/onboarding-flow';

import { RouteMessages } from '../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

/** `/onboarding/conditions` — nbl_Onb_Conditions (B-N2-02). */
export default async function ConditionsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingConditions">
      <ConditionsStep />
    </RouteMessages>
  );
}
