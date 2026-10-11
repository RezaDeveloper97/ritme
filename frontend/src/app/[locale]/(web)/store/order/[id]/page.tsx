import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { getTranslations, setRequestLocale } from 'next-intl/server';

import { WebShopDone } from '@/screens/web-shop';

import { RouteMessages } from '../../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** Signed-in page (CB-SHOP-11): noindex, no canonical; the content is the visitor's own cart / order. */
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'webShop' });
  return { title: t('done.metaTitle'), robots: { index: false, follow: false } };
}

/** `/store/order/[id]` — desktop order placed (CB-SHOP-11). */
export default async function StoreOrderRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const n = Number(id);
  if (!Number.isInteger(n) || n <= 0) notFound();
  return (
    <RouteMessages route="webShop">
      <WebShopDone id={n} />
    </RouteMessages>
  );
}
