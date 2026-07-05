import { setRequestLocale } from 'next-intl/server';

import { LogPage } from '@/screens/log';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function LogRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <LogPage />;
}
