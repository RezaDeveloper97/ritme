import { notFound } from 'next/navigation';
import { setRequestLocale } from 'next-intl/server';

import { ShopOrderPage } from '@/screens/shop-order';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/shop/order/[id]` — order placed (`?placed=1`) and order detail (CB-SHOP-08, nbd_Shop_Order). */
export default async function ShopOrderRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const n = Number(id);
  if (!Number.isInteger(n) || n <= 0) notFound();
  return (
    <RouteMessages route="shop">
      <ShopOrderPage id={n} />
    </RouteMessages>
  );
}
