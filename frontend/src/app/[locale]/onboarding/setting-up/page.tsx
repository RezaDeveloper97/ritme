import { setRequestLocale } from 'next-intl/server';

import { ReadyStep } from '@/screens/onboarding-flow';

import { RouteMessages } from '../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

/** `/onboarding/setting-up` — nbl_Onb_Ready (B-N2-02). */
export default async function SettingUpRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingSettingUp">
      <ReadyStep />
    </RouteMessages>
  );
}
