import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';

import { WebShopCheckout } from '@/screens/web-shop';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** Signed-in page (CB-SHOP-11): noindex, no canonical; the content is the visitor's own cart / order. */
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'webShop' });
  return { title: t('checkout.metaTitle'), robots: { index: false, follow: false } };
}

/** `/store/checkout` — desktop checkout (CB-SHOP-11). */
export default async function StoreCheckoutRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="webShop">
      <WebShopCheckout />
    </RouteMessages>
  );
}
