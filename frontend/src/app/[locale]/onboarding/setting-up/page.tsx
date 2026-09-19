import { setRequestLocale } from 'next-intl/server';

import { SettingUpPage } from '@/screens/onboarding-setting-up';

import { RouteMessages } from '../../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

export default async function SettingUpRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="onboardingSettingUp">
      <SettingUpPage />
    </RouteMessages>
  );
}
