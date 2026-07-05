import { setRequestLocale } from 'next-intl/server';

import { CyclePage } from '@/screens/cycle';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function CycleRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <CyclePage />;
}
