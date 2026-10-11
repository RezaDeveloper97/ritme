import { setRequestLocale } from 'next-intl/server';

import { InsuranceCoveragePage } from '@/screens/insurance-coverage';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/insurance/coverage` — پوشش‌های بیمه‌نامه (CB-INS-03, nbl_Ins_Coverage). */
export default async function InsuranceCoverageRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="insuranceCoverage">
      <InsuranceCoveragePage />
    </RouteMessages>
  );
}
