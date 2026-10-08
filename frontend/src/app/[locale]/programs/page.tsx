import { setRequestLocale } from 'next-intl/server';

import { ConditionsHubPage } from '@/screens/conditions';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/programs` — «برنامه‌های مراقبت» (CB-COND-02, nbl_Cond_Hub). A services hub: bottom nav (خدمات). */
export default async function ProgramsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="programs">
      <ConditionsHubPage />
    </RouteMessages>
  );
}
