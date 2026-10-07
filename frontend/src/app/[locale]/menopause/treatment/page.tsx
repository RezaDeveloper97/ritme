import { setRequestLocale } from 'next-intl/server';

import { MenopauseTreatmentPage } from '@/screens/menopause-treatment';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/menopause/treatment` — درمان و مراقبت (CB-MENO-10, nbl_Meno_Treatment). Back header: no bottom nav. */
export default async function MenopauseTreatmentRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="menopauseTreatment">
      <MenopauseTreatmentPage />
    </RouteMessages>
  );
}
