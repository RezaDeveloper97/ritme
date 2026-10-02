import { setRequestLocale } from 'next-intl/server';

import { LossCarePage } from '@/screens/loss-care';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/loss/care` — care & support after a loss (CB-LOSS-02, nbl_Loss_Care). Full screen: no tab bar, banners or shop. */
export default async function LossCareRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="lossCare">
      <LossCarePage />
    </RouteMessages>
  );
}
