import { setRequestLocale } from 'next-intl/server';

import { PeriodLenPage } from '@/screens/onboarding-period-len';

interface Props { params: Promise<{ locale: string }> }

export default async function PeriodLenRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <PeriodLenPage />;
}
