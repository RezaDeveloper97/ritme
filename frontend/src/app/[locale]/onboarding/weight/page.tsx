import { setRequestLocale } from 'next-intl/server';

import { WeightPage } from '@/screens/onboarding-weight';

interface Props { params: Promise<{ locale: string }> }

export default async function WeightRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <WeightPage />;
}
