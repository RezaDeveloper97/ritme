import { setRequestLocale } from 'next-intl/server';

import { ContraceptionSetupPage } from '@/screens/contraception-setup';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/contraception/setup` — روش پیشگیری (CB-CONTRA-02, nbl_Contra_Setup). */
export default async function ContraceptionSetupRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="contraceptionSetup">
      <ContraceptionSetupPage />
    </RouteMessages>
  );
}
