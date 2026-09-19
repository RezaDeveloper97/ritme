import { setRequestLocale } from 'next-intl/server';

import { PeriodLenPage } from '@/screens/onboarding-period-len';

import { RouteMessages } from '../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

export default async function PeriodLenRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingPeriodLen">
      <PeriodLenPage />
    </RouteMessages>
  );
}
