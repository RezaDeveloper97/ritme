import { setRequestLocale } from 'next-intl/server';

import { CycleLenPage } from '@/screens/onboarding-cycle-len';

interface Props { params: Promise<{ locale: string }> }

export default async function CycleLenRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <CycleLenPage />;
}
