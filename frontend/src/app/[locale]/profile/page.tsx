import { setRequestLocale } from 'next-intl/server';

import { ProfilePage } from '@/screens/profile';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function ProfileRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="profile">
      <ProfilePage />
    </RouteMessages>
  );
}
