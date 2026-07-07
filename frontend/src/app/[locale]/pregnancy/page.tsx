import { setRequestLocale } from 'next-intl/server';

import { PregnancyPage } from '@/screens/pregnancy';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function PregnancyRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <PregnancyPage />;
}
