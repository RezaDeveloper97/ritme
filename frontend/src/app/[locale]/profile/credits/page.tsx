import { setRequestLocale } from 'next-intl/server';

import { CreditsUsagePage } from '@/screens/credits-usage';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/profile/credits` — «مصرف اعتبار» (AI credits: balance, costs, last 30 days, history). */
export default async function ProfileCreditsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="profileCredits">
      <CreditsUsagePage />
    </RouteMessages>
  );
}
