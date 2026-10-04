import { setRequestLocale } from 'next-intl/server';

import { IvfScanPage } from '@/screens/ivf-scan';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ date?: string }>;
}

/** `/ivf/scan[?date=Y-m-d]` — «ثبت نتیجه سونو» (nbl_IVF_Scan, CB-IVF-04); back-header screen, no bottom nav. */
export default async function IvfScanRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { date } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="ivfScan">
      <IvfScanPage date={date} />
    </RouteMessages>
  );
}
