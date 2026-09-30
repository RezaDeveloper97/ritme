import { setRequestLocale } from 'next-intl/server';

import { NotificationSettingsPage } from '@/screens/notification-settings';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/profile/notifications` — اعلان‌ها و یادآورها (B-N1-11, nbl_Me_Notifications). */
export default async function ProfileNotificationsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="profileNotifications">
      <NotificationSettingsPage />
    </RouteMessages>
  );
}
