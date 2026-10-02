import { setRequestLocale } from 'next-intl/server';

import { CompanionLinksPage } from '@/screens/companion-home';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/companion/links` — Me «کد همدم» for a companion account (B-N4-05). */
export default async function CompanionLinksRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="companionLinks">
      <CompanionLinksPage />
    </RouteMessages>
  );
}
