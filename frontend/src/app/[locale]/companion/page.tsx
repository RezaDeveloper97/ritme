import { setRequestLocale } from 'next-intl/server';

import { CompanionHomePage } from '@/screens/companion-home';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/companion` — «ریتمی همراه» v2 member home (compHome, or compPersonal without a link), male accounts. */
export default async function CompanionRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="companion">
      <CompanionHomePage />
    </RouteMessages>
  );
}
