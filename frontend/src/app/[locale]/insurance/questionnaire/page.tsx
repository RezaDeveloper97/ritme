import { setRequestLocale } from 'next-intl/server';

import { InsuranceQuestionnairePage } from '@/screens/insurance-questionnaire';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/insurance/questionnaire` — پرسشنامه سلامت (CB-INS-05, nbl_Ins_Health). */
export default async function InsuranceQuestionnaireRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="insuranceQuestionnaire">
      <InsuranceQuestionnairePage />
    </RouteMessages>
  );
}
