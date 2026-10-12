import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { fetchPublicShopList, parseShopListRecord, shopListSearch } from '@/entities/shop';
import { WebShopList } from '@/screens/web-shop';

import { RouteMessages } from '../../../../RouteMessages';
import { originOf } from '../origin';

interface Props {
  params: Promise<{ locale: string; category: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale, category } = await params;
  const t = await getTranslations({ locale, namespace: 'webShop' });
  const list = await fetchPublicShopList(category, locale);
  const name = list?.category?.title;
  if (!name) return { title: t('metaFallback'), robots: { index: false, follow: true } };
  const origin = await originOf();
  // The canonical is the category itself: filtered / sorted / paged views are the same page.
  const url = `${origin ?? ''}/${locale}/store/${category}`;
  const title = t('list.metaTitle', { name });
  const description = t('list.metaDescription', { name });
  return {
    title,
    description,
    alternates: { canonical: url },
    openGraph: { title, description, url, type: 'website', siteName: 'Ritme' },
    twitter: { card: 'summary', title, description },
  };
}

/** `/store/[category]` — filtered product list; the first page is server-rendered from the query string (CB-SHOP-10). */
export default async function StoreCategoryRoute({ params, searchParams }: Props) {
  const { locale, category } = await params;
  setRequestLocale(locale);
  const state = parseShopListRecord(await searchParams);
  const initial = await fetchPublicShopList(category, locale, state);
  return (
    <RouteMessages route="webShop">
      {/* The list reads its filters from the query string (useSearchParams needs a Suspense boundary). */}
      <Suspense>
        <WebShopList category={category} initial={initial} initialSearch={shopListSearch(state)} />
      </Suspense>
    </RouteMessages>
  );
}
