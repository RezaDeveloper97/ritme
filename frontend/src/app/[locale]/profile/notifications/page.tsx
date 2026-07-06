import { setRequestLocale } from 'next-intl/server';

import { NotificationsPage } from '@/screens/profile-notifications';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function ProfileNotificationsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <NotificationsPage />;
}
