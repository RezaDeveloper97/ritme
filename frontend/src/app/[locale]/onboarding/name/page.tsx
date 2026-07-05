import { setRequestLocale } from 'next-intl/server';

import { NamePage } from '@/screens/onboarding-name';

interface Props { params: Promise<{ locale: string }> }

export default async function NameRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <NamePage />;
}
