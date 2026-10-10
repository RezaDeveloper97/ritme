import { setRequestLocale } from 'next-intl/server';

import { PmddPage } from '@/screens/condition-pmdd';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/programs/pmdd` — «پرسشنامه خلق روزانه» (CB-COND-04, nbl_Cond_PMDD). A form: no bottom nav. */
export default async function ProgramsPmddRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="programsPmdd">
      <PmddPage />
    </RouteMessages>
  );
}
