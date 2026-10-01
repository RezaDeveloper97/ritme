import { setRequestLocale } from 'next-intl/server';

import { ContraceptionMissedPage } from '@/screens/contraception-missed';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/contraception/missed` — قرص جا افتاده (CB-CONTRA-03, nbl_Contra_Missed). */
export default async function ContraceptionMissedRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="contraceptionMissed">
      <ContraceptionMissedPage />
    </RouteMessages>
  );
}
