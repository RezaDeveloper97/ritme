import { setRequestLocale } from 'next-intl/server';

import { LogCustomizePage } from '@/screens/log-customize';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/log/customize` — the log sheet's gear (B-N3-04). */
export default async function LogCustomizeRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="logCustomize">
      <LogCustomizePage />
    </RouteMessages>
  );
}
