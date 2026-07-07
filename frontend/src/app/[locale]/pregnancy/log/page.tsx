import { setRequestLocale } from 'next-intl/server';

import { PregnancyLogPage } from '@/screens/pregnancy-log';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ tab?: string }>;
}

export default async function PregnancyLogRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { tab } = await searchParams;
  setRequestLocale(locale);
  return <PregnancyLogPage initialTab={tab} />;
}
