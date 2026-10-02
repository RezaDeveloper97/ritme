import { setRequestLocale } from 'next-intl/server';

import { LossNextPage } from '@/screens/loss-next';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/loss/next` — the next step after a loss (CB-LOSS-02, nbl_Loss_Next). Full screen: no tab bar, banners or shop. */
export default async function LossNextRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="lossNext">
      <LossNextPage />
    </RouteMessages>
  );
}
