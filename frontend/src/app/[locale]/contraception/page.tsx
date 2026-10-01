import { setRequestLocale } from 'next-intl/server';

import { ContraceptionPage } from '@/screens/contraception';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/contraception` — قرص پیشگیری (CB-CONTRA-02, nbl_Contra_Pill). */
export default async function ContraceptionRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="contraception">
      <ContraceptionPage />
    </RouteMessages>
  );
}
