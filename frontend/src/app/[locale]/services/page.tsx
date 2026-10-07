import { setRequestLocale } from 'next-intl/server';

import { ServicesPage } from '@/screens/services';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/services` — the «خدمات» tab hub (B-N7-01, v17_Main). */
export default async function ServicesRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="services">
      <ServicesPage />
    </RouteMessages>
  );
}
