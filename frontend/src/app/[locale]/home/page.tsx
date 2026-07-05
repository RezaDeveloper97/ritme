import { setRequestLocale } from 'next-intl/server';

import { HomePage } from '@/screens/home';

interface Props { params: Promise<{ locale: string }> }

export default async function HomeRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <HomePage />;
}
