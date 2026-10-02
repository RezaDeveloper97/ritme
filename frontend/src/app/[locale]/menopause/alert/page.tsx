import { setRequestLocale } from 'next-intl/server';

import { MenopauseAlertPage } from '@/screens/menopause-alert';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/menopause/alert` — خونریزی بعد از یائسگی (CB-MENO-09, nbl_Meno_Alert). Back header: no bottom nav. */
export default async function MenopauseAlertRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="menopauseAlert">
      <MenopauseAlertPage />
    </RouteMessages>
  );
}
