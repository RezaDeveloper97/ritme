import { setRequestLocale } from 'next-intl/server';

import { IvfTwwPage } from '@/screens/ivf-tww';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/ivf/tww` — «دو هفته انتظار» (nbl_IVF_TWW, CB-IVF-05): countdown, mood, danger note, result → pregnancy setup or `/loss`. */
export default async function IvfTwwRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="ivfTww">
      <IvfTwwPage />
    </RouteMessages>
  );
}
