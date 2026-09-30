import { setRequestLocale } from 'next-intl/server';

import { AccountPage } from '@/screens/profile';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/profile/account` — حساب کاربری (B-N1-10, nbl_Me_Profile). */
export default async function ProfileAccountRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="profileAccount">
      <AccountPage />
    </RouteMessages>
  );
}
