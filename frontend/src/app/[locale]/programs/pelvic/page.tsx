import { setRequestLocale } from 'next-intl/server';

import { PelvicPage } from '@/screens/pelvic';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/programs/pelvic` — «کف لگن و مثانه» (CB-PELV-02, nbl_Pelvic_Plan). A services hub: bottom nav (خدمات). */
export default async function PelvicRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="pelvic">
      <PelvicPage />
    </RouteMessages>
  );
}
