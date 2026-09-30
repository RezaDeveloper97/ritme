import { setRequestLocale } from 'next-intl/server';

import { CycleSettingsPage } from '@/screens/cycle-settings';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/cycle/settings` — تنظیمات سیکل (B-N1-09, nbl_Cycle_Settings). */
export default async function CycleSettingsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="cycleSettings">
      <CycleSettingsPage />
    </RouteMessages>
  );
}
