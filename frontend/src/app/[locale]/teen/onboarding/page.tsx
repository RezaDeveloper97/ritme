import { setRequestLocale } from 'next-intl/server';

import { TeenOnboardingPage } from '@/screens/teen-onboarding';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/teen/onboarding` — teen answers (CB-TEEN-02, nbl_Teen_Onb). A form: no bottom nav. */
export default async function TeenOnboardingRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="teenOnboarding">
      <TeenOnboardingPage />
    </RouteMessages>
  );
}
