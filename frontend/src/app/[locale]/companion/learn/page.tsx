import { setRequestLocale } from 'next-intl/server';

import { CompanionLearnPage } from '@/screens/companion-learn';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/companion/learn` — «یادگیری» (companion v2 compLearn), companion nav. */
export default async function CompanionLearnRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="companionLearn">
      <CompanionLearnPage />
    </RouteMessages>
  );
}
