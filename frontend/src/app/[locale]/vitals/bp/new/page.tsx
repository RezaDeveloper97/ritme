import { setRequestLocale } from 'next-intl/server';

import { VitalsAddPage } from '@/screens/vitals-add';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/vitals/bp/new` — add blood pressure (nbl_Vitals_AddBP). A form: close header, no bottom nav. */
export default async function VitalsAddBpRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="vitalsAdd">
      <VitalsAddPage type="bp" />
    </RouteMessages>
  );
}
