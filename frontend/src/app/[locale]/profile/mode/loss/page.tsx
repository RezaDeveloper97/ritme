import { setRequestLocale } from 'next-intl/server';

import { PregnancyLossPage } from '@/screens/mode';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/profile/mode/loss` — the calm pregnancy exit (B-N2-03; CB-LOSS-02 later moves it to `/loss`). */
export default async function ProfileModeLossRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="profileModeLoss">
      <PregnancyLossPage />
    </RouteMessages>
  );
}
