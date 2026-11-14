import { setRequestLocale } from 'next-intl/server';

import { CreditsBuyPage } from '@/screens/credits-buy';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/plus/credits` — «خرید اعتبار» (credit packs → /plus/checkout?pack=). */
export default async function PlusCreditsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="plusCredits">
      <CreditsBuyPage />
    </RouteMessages>
  );
}
