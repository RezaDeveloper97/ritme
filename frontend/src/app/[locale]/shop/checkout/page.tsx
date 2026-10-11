import { setRequestLocale } from 'next-intl/server';

import { ShopCheckoutPage } from '@/screens/shop-checkout';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/shop/checkout` — «تکمیل خرید», pay on delivery only (CB-SHOP-08, nbd_Shop_Checkout). */
export default async function ShopCheckoutRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="shop">
      <ShopCheckoutPage />
    </RouteMessages>
  );
}
