import { setRequestLocale } from 'next-intl/server';

import { IvfMedsPage } from '@/screens/ivf-meds';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/ivf/meds` — «برنامه تزریق» (nbl_IVF_Meds, CB-IVF-03), the IVF stage tab «درمان». */
export default async function IvfMedsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="ivfMeds">
      <IvfMedsPage />
    </RouteMessages>
  );
}
