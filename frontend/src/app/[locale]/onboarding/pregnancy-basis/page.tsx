import { setRequestLocale } from 'next-intl/server';

import { PregnancyBasisPage } from '@/screens/onboarding-pregnancy-basis';

interface Props { params: Promise<{ locale: string }> }

export default async function PregnancyBasisRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <PregnancyBasisPage />;
}
