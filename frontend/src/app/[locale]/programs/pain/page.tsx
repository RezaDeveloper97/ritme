import { setRequestLocale } from 'next-intl/server';

import { PainDiaryPage } from '@/screens/condition-pain';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/programs/pain` — «دفترچه درد» (CB-COND-03, nbl_Cond_Endo). A form with a back header: no bottom nav. */
export default async function ProgramsPainRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="programsPain">
      <PainDiaryPage />
    </RouteMessages>
  );
}
