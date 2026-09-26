import { setRequestLocale } from 'next-intl/server';

import { PregnancySetupPage } from '@/screens/pregnancy-onboarding';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function PregnancySetupRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="pregnancySetup">
      <PregnancySetupPage />
    </RouteMessages>
  );
}
