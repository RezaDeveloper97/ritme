import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';

import { fetchPublicShopList } from '@/entities/shop';
import { WebShopList } from '@/screens/web-shop';

import { RouteMessages } from '../../../../RouteMessages';
import { originOf } from '../origin';

interface Props {
  params: Promise<{ locale: string; category: string }>;
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale, category } = await params;
  const t = await getTranslations({ locale, namespace: 'webShop' });
  const list = await fetchPublicShopList(category, locale);
  const name = list?.category?.title;
  if (!name) return { title: t('metaFallback'), robots: { index: false, follow: true } };
  const origin = await originOf();
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

/** `/store/[category]` — filtered product list; the first page is server-rendered (CB-SHOP-10). */
export default async function StoreCategoryRoute({ params }: Props) {
  const { locale, category } = await params;
  setRequestLocale(locale);
  const initial = await fetchPublicShopList(category, locale);
  return (
    <RouteMessages route="webShop">
      <WebShopList category={category} initial={initial} />
    </RouteMessages>
  );
}
