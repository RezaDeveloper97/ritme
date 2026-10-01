import { setRequestLocale } from 'next-intl/server';

import { ContraceptionOtherPage } from '@/screens/contraception-other';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/contraception/other` — یادآورهای روش پیشگیری (CB-CONTRA-03, nbl_Contra_Other). */
export default async function ContraceptionOtherRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="contraceptionOther">
      <ContraceptionOtherPage />
    </RouteMessages>
  );
}
