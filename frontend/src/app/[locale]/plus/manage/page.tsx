import { setRequestLocale } from 'next-intl/server';

import { ManagePage } from '@/screens/plus-manage';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/plus/manage` — my subscription (B-N2-07, nbl_Prem_Manage). */
export default async function PlusManageRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="plusManage">
      <ManagePage />
    </RouteMessages>
  );
}
