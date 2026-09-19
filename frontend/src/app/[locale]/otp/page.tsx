import { setRequestLocale } from 'next-intl/server';

import { OtpPage } from '@/screens/auth-otp';

import { RouteMessages } from '../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

export default async function OtpRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="otp">
      <OtpPage />
    </RouteMessages>
  );
}
