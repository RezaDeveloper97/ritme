import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';

import { fetchPublicShopHome } from '@/entities/shop';
import { WebShopHome } from '@/screens/web-shop';

import { RouteMessages } from '../../../RouteMessages';
import { originOf } from './origin';

interface Props {
  params: Promise<{ locale: string }>;
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'webShop' });
  const origin = await originOf();
  const url = `${origin ?? ''}/${locale}/store`;
  const title = t('home.metaTitle');
  const description = t('home.metaDescription');
  return {
    title,
    description,
    alternates: { canonical: url },
    openGraph: { title, description, url, type: 'website', siteName: 'Ritme' },
    twitter: { card: 'summary', title, description },
  };
}

/** `/store` — public desktop shop home (CB-SHOP-10). `store` is a public segment (no session needed). */
export default async function StoreRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const initial = await fetchPublicShopHome(locale);
  return (
    <RouteMessages route="webShop">
      <WebShopHome initial={initial} />
    </RouteMessages>
  );
}
