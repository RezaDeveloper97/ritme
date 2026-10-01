import { setRequestLocale } from 'next-intl/server';

import { GoalStep } from '@/screens/onboarding-flow';

import { RouteMessages } from '../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

/** `/onboarding/intention` — nbl_Onb_Goal (B-N2-02). */
export default async function IntentionRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingIntention">
      <GoalStep />
    </RouteMessages>
  );
}
