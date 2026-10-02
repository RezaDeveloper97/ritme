import { setRequestLocale } from 'next-intl/server';

import { MenopauseQuestionnairePage } from '@/screens/menopause-score';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/menopause/score/questionnaire` — the monthly 11-question flow (CB-MENO-08). A form: no bottom nav. */
export default async function MenopauseQuestionnaireRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="menopauseScoreQuestionnaire">
      <MenopauseQuestionnairePage />
    </RouteMessages>
  );
}
