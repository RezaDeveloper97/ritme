import { setRequestLocale } from 'next-intl/server';

import { BirthdayPage } from '@/screens/onboarding-birthday';

interface Props { params: Promise<{ locale: string }> }

export default async function BirthdayRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <BirthdayPage />;
}
