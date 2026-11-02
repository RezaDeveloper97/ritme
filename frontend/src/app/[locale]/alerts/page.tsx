import { setRequestLocale } from 'next-intl/server';

import { AlertsPage } from '@/screens/alerts';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/alerts` — «هشدارها و پیام‌ها»: the generic alerts inbox for every mode (T-M10-15, SPEC §5.4.1). */
export default async function AlertsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="alerts">
      <AlertsPage />
    </RouteMessages>
  );
}
