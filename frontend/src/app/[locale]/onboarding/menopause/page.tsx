import { setRequestLocale } from 'next-intl/server';

import { MenopauseStep } from '@/screens/onboarding-flow';

import { RouteMessages } from '../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

/** `/onboarding/menopause` — nbl_Onb_Meno (B-N2-02). */
export default async function MenopauseRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingMenopause">
      <MenopauseStep />
    </RouteMessages>
  );
}
