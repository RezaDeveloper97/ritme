import { setRequestLocale } from 'next-intl/server';

import { VitalsAddPage } from '@/screens/vitals-add';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/vitals/heart-rate/new` — add heart rate (nbl_Vitals_AddHR). A form: close header, no bottom nav. */
export default async function VitalsAddHrRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="vitalsAdd">
      <VitalsAddPage type="hr" />
    </RouteMessages>
  );
}
