import { setRequestLocale } from 'next-intl/server';

import { ConditionsPage } from '@/screens/onboarding-conditions';

import { RouteMessages } from '../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

export default async function ConditionsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingConditions">
      <ConditionsPage />
    </RouteMessages>
  );
}
