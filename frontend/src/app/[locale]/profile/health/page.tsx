import { setRequestLocale } from 'next-intl/server';

import { ProfileHealthPage } from '@/screens/profile-health';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function ProfileHealthRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <ProfileHealthPage />;
}
