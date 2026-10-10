import { setRequestLocale } from 'next-intl/server';

import { BleedingChartPage } from '@/screens/condition-bleeding';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/programs/bleeding` — «جدول خونریزی» (CB-COND-05, nbl_Cond_Bleed); back-header form, no bottom nav. */
export default async function ProgramsBleedingRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="programsBleeding">
      <BleedingChartPage />
    </RouteMessages>
  );
}
