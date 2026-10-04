import { setRequestLocale } from 'next-intl/server';

import { IvfCycleSetupPage } from '@/screens/ivf-cycle';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/ivf/cycle/new` — «شروع سیکل درمان» setup (CB-IVF-06b; form, no nav). */
export default async function NewIvfCycleRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="ivfCycle">
      <IvfCycleSetupPage />
    </RouteMessages>
  );
}
