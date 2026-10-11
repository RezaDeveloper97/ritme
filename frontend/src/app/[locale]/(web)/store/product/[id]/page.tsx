import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { getTranslations, setRequestLocale } from 'next-intl/server';

import { fetchPublicShopProduct, formatToman } from '@/entities/shop';
import { WebShopProduct } from '@/screens/web-shop';

import { RouteMessages } from '../../../../../RouteMessages';
import { originOf } from '../../origin';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

const parseId = (id: string): number | null => {
  const n = Number(id);
  return Number.isInteger(n) && n > 0 ? n : null;
};

/** Title / description / OpenGraph / canonical from the public catalog API, server-side (no session, no user data). */
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale, id } = await params;
  const n = parseId(id);
  const t = await getTranslations({ locale, namespace: 'webShop' });
  const shop = await getTranslations({ locale, namespace: 'shop' });
  const p = n ? await fetchPublicShopProduct(n, locale) : null;
  if (!p) return { title: t('product.metaFallback'), robots: { index: false, follow: true } };

  const { unit, n: amount } = formatToman(p.price, locale);
  const title = t('product.metaTitle', { title: p.title });
  const description = (p.description ?? p.subtitle)?.slice(0, 160) || t('product.metaDescription', { title: p.title, price: shop(`price.${unit}`, { n: amount }) });
  const origin = await originOf();
  const url = `${origin ?? ''}/${locale}/store/product/${p.id}`;
  const image = p.images[0] ?? p.image_url;
  return {
    title,
    description,
    alternates: { canonical: url },
    openGraph: { title, description, url, type: 'website', siteName: 'Ritme', ...(image ? { images: [{ url: image, alt: p.title }] } : {}) },
    twitter: { card: image ? 'summary_large_image' : 'summary', title, description },
  };
}

/** `/store/product/[id]` — one product for the web: server-fetched so the first HTML carries the content (CB-SHOP-10). */
export default async function StoreProductRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const n = parseId(id);
  if (!n) notFound();
  const initial = await fetchPublicShopProduct(n, locale);
  return (
    <RouteMessages route="webShop">
      <WebShopProduct id={n} initial={initial} />
    </RouteMessages>
  );
}
