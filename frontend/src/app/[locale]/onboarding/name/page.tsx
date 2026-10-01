import { setRequestLocale } from 'next-intl/server';

import { NameStep } from '@/screens/onboarding-flow';

import { RouteMessages } from '../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

/** `/onboarding/name` — nbl_Onb_Name (B-N2-02). */
export default async function NameRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingName">
      <NameStep />
    </RouteMessages>
  );
}
