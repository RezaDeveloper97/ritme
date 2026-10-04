import { setRequestLocale } from 'next-intl/server';

import { LogKickPage } from '@/screens/log-kick';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/pregnancy/kicks` — kick counter (B-N5-08, Log_Kick). */
export default async function PregnancyKicksRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="pregnancyKicks">
      <LogKickPage />
    </RouteMessages>
  );
}
