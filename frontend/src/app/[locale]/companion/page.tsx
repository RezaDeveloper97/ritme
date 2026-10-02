import { setRequestLocale } from 'next-intl/server';

import { CompanionHomePage } from '@/screens/companion-home';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/companion` — companion panel home (B-N4-05, nbl_Hamdam_Home), male accounts. */
export default async function CompanionRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="companion">
      <CompanionHomePage />
    </RouteMessages>
  );
}
