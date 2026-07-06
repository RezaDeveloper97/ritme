import { setRequestLocale } from 'next-intl/server';

import { ProfilePersonalPage } from '@/screens/profile-personal';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function ProfilePersonalRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <ProfilePersonalPage />;
}
