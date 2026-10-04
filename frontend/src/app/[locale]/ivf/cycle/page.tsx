import { setRequestLocale } from 'next-intl/server';

import { IvfCycleEditorPage } from '@/screens/ivf-cycle';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/ivf/cycle` — «مرحله و تاریخ‌ها» (CB-IVF-06b; from the IVF home timeline, form, no nav). */
export default async function IvfCycleRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="ivfCycle">
      <IvfCycleEditorPage />
    </RouteMessages>
  );
}
