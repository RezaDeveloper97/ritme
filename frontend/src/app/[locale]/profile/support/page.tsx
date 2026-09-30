import { setRequestLocale } from 'next-intl/server';

import { SupportPage } from '@/screens/support';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/profile/support` — پشتیبانی (B-N1-12, nbl_Me_Support). */
export default async function ProfileSupportRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="profileSupport">
      <SupportPage />
    </RouteMessages>
  );
}
