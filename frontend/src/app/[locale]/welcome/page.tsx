import { setRequestLocale } from 'next-intl/server';

import { WelcomePage } from '@/screens/welcome';

import { RouteMessages } from '../../RouteMessages';

interface Props { params: Promise<{ locale: string }> }

export default async function WelcomeRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="welcome">
      <WelcomePage />
    </RouteMessages>
  );
}
