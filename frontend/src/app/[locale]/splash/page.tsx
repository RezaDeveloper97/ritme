import { setRequestLocale } from 'next-intl/server';

import { SplashPage } from '@/screens/auth-splash';

interface Props { params: Promise<{ locale: string }> }

export default async function SplashRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <SplashPage />;
}
