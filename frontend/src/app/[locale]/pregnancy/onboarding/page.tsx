import { setRequestLocale } from 'next-intl/server';

import { PregnancyOnboardingPage } from '@/screens/pregnancy-onboarding';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function PregnancyOnboardingRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="pregnancyOnboarding">
      <PregnancyOnboardingPage />
    </RouteMessages>
  );
}
