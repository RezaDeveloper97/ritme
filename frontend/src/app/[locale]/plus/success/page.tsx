import { setRequestLocale } from 'next-intl/server';

import { PlusSuccessPage } from '@/screens/plus-success';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/plus/success` — welcome to Plus (B-N2-07, nbl_Prem_Success). */
export default async function PlusSuccessRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="plusSuccess">
      <PlusSuccessPage />
    </RouteMessages>
  );
}
