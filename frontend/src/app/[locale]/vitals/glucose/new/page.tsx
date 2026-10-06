import { setRequestLocale } from 'next-intl/server';

import { VitalsAddPage } from '@/screens/vitals-add';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/vitals/glucose/new` — add blood glucose (nbl_Vitals_AddGlucose). A form: close header, no bottom nav. */
export default async function VitalsAddGlucoseRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="vitalsAdd">
      <VitalsAddPage type="glucose" />
    </RouteMessages>
  );
}
