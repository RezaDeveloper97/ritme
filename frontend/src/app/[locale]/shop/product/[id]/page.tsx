import { notFound } from 'next/navigation';
import { setRequestLocale } from 'next-intl/server';

import { ShopProductPage } from '@/screens/shop';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/shop/product/[id]` — product page of the internal catalog (CB-SHOP-07). */
export default async function ShopProductRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const n = Number(id);
  if (!Number.isInteger(n) || n <= 0) notFound();
  return (
    <RouteMessages route="shop">
      <ShopProductPage id={n} />
    </RouteMessages>
  );
}
