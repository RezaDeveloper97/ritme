import { setRequestLocale } from 'next-intl/server';

import { ShopCartPage } from '@/screens/shop-cart';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/shop/cart` — «سبد خرید» (CB-SHOP-08, nbd_Shop_Cart). */
export default async function ShopCartRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="shop">
      <ShopCartPage />
    </RouteMessages>
  );
}
