import { setRequestLocale } from 'next-intl/server';

import { AppearancePage } from '@/screens/appearance';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/profile/appearance` — ظاهر (B-N1-10, nbl_Me_Appearance). */
export default async function ProfileAppearanceRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="profileAppearance">
      <AppearancePage />
    </RouteMessages>
  );
}
