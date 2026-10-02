import { setRequestLocale } from 'next-intl/server';

import { MenopauseScorePage } from '@/screens/menopause-score';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/menopause/score` — امتیاز علائم (CB-MENO-08, nbl_Meno_Score): the menopause tab «علائم», with the bottom nav. */
export default async function MenopauseScoreRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="menopauseScore">
      <MenopauseScorePage />
    </RouteMessages>
  );
}
