import { setRequestLocale } from 'next-intl/server';

import { CycleStep } from '@/screens/onboarding-flow';

import { RouteMessages } from '../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

/** `/onboarding/cycle` — nbl_Onb_Cycle (B-N2-02). */
export default async function CycleRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingCycle">
      <CycleStep />
    </RouteMessages>
  );
}
