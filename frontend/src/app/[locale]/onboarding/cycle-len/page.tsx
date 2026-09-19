import { setRequestLocale } from 'next-intl/server';

import { CycleLenPage } from '@/screens/onboarding-cycle-len';

import { RouteMessages } from '../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

export default async function CycleLenRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingCycleLen">
      <CycleLenPage />
    </RouteMessages>
  );
}
