import { setRequestLocale } from 'next-intl/server';

import { WelcomePage } from '@/screens/welcome';

interface Props { params: Promise<{ locale: string }> }

export default async function WelcomeRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <WelcomePage />;
}
