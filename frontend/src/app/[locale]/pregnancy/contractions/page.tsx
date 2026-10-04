import { setRequestLocale } from 'next-intl/server';

import { LogContractionPage } from '@/screens/log-contraction';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/pregnancy/contractions` — contraction timer (B-N5-08, Log_Contraction). */
export default async function PregnancyContractionsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="pregnancyContractions">
      <LogContractionPage />
    </RouteMessages>
  );
}
