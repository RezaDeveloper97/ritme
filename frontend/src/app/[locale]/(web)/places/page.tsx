import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';

import { WebSearchPage } from '@/screens/web-city-services';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'webDirectory' });
  return {
    title: t('search.metaTitle'),
    description: t('search.metaDescription'),
    alternates: { canonical: `/${locale}/places` },
    openGraph: { title: t('search.metaTitle'), description: t('search.metaDescription'), type: 'website', url: `/${locale}/places` },
  };
}

/** `/places` — public desktop directory search + map (CB-DIR-11). `places` is a public segment (no session needed). */
export default async function PlacesRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="webDirectory">
      <WebSearchPage />
    </RouteMessages>
  );
}
