import { setRequestLocale } from 'next-intl/server';

import { GenderPage } from '@/screens/onboarding-gender';

interface Props { params: Promise<{ locale: string }> }

export default async function GenderRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <GenderPage />;
}
