import { setRequestLocale } from 'next-intl/server';

import { IntentionPage } from '@/screens/onboarding-intention';

import { RouteMessages } from '../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

export default async function IntentionRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingIntention">
      <IntentionPage />
    </RouteMessages>
  );
}
