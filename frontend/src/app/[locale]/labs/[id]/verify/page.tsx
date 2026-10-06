import { setRequestLocale } from 'next-intl/server';

import { LabVerifyPage } from '@/screens/lab-verify';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/labs/[id]/verify` — nbl_Lab_Verify (B-N6-07). */
export default async function LabVerifyPageRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="labDetail">
      <LabVerifyPage id={Number(id)} />
    </RouteMessages>
  );
}
