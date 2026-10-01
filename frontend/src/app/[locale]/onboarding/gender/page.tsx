import { setRequestLocale } from 'next-intl/server';

import { GenderStep } from '@/screens/onboarding-flow';

import { RouteMessages } from '../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

/** `/onboarding/gender` — nbl_Onb_Gender (B-N2-02). */
export default async function GenderRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingGender">
      <GenderStep />
    </RouteMessages>
  );
}
