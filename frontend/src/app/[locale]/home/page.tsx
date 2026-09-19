import { setRequestLocale } from 'next-intl/server';

import { HomePage } from '@/screens/home';

import { RouteMessages } from '../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

export default async function HomeRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="home">
      <HomePage />
    </RouteMessages>
  );
}
