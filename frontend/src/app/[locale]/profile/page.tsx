import { setRequestLocale } from 'next-intl/server';

import { ProfilePage } from '@/screens/profile';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function ProfileRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <ProfilePage />;
}
