import { setRequestLocale } from 'next-intl/server';

import { CompanionInvitePage } from '@/screens/companion-invite';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/companions/new` — افزودن همدم (B-N4-04, nbl_Hamdam_Type → Access → Children → Invite → Done). */
export default async function CompanionsNewRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="companionsNew">
      <CompanionInvitePage />
    </RouteMessages>
  );
}
