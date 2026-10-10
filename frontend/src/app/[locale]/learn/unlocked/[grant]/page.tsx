import { setRequestLocale } from 'next-intl/server';

import { LearnUnlockedPage } from '@/screens/learn-unlocked';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; grant: string }>;
}

/** `/learn/unlocked/[grant]` — «دوره برایت باز شد» (nbl_Learn_Unlocked, B-N8-03), the inbox notice's target. */
export default async function LearnUnlockedRoute({ params }: Props) {
  const { locale, grant } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="learnUnlocked">
      <LearnUnlockedPage grantId={/^\d{1,12}$/.test(grant) ? Number(grant) : 0} />
    </RouteMessages>
  );
}
