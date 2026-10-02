import { setRequestLocale } from 'next-intl/server';

import { MenopauseStagePage } from '@/screens/menopause-stage';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/menopause/stage` — حالت یائسگی (CB-MENO-05, nbl_Meno_Stage). A form: no bottom nav. */
export default async function MenopauseStageRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="menopauseStage">
      <MenopauseStagePage />
    </RouteMessages>
  );
}
