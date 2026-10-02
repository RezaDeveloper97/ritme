import { setRequestLocale } from 'next-intl/server';

import { LossStartPage } from '@/screens/loss-start';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/loss` — calm start of the pregnancy loss path (CB-LOSS-02, nbl_Loss_Start). Full screen: no tab bar, banners or shop. */
export default async function LossStartRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="lossStart">
      <LossStartPage />
    </RouteMessages>
  );
}
