import { setRequestLocale } from 'next-intl/server';

import { HealthStep } from '@/screens/onboarding-flow';

import { RouteMessages } from '../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

/** `/onboarding/health` — nbl_Onb_Health (B-N2-02). */
export default async function HealthRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingHealth">
      <HealthStep />
    </RouteMessages>
  );
}
