import { setRequestLocale } from 'next-intl/server';

import { CycleDurationPage } from '@/screens/onboarding-cycle-duration';

interface Props { params: Promise<{ locale: string }> }

export default async function CycleDurationRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <CycleDurationPage />;
}
