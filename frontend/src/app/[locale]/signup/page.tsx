import { setRequestLocale } from 'next-intl/server';

import { SignupPage } from '@/screens/auth-signup';

interface Props { params: Promise<{ locale: string }> }

export default async function SignupRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <SignupPage />;
}
