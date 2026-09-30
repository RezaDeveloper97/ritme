import { setRequestLocale } from 'next-intl/server';

import { PrivacyPage } from '@/screens/privacy';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/profile/privacy` — حریم خصوصی و امنیت (B-N1-12, nbl_Me_Privacy). */
export default async function ProfilePrivacyRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="profilePrivacy">
      <PrivacyPage />
    </RouteMessages>
  );
}
