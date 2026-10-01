import { setRequestLocale } from 'next-intl/server';

import { ModePage } from '@/screens/mode';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/profile/mode` — مرحله زندگی (B-N2-03, nbl_Me_Mode). */
export default async function ProfileModeRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="profileMode">
      <ModePage />
    </RouteMessages>
  );
}
