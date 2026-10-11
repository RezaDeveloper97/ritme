import { setRequestLocale } from 'next-intl/server';

import { ShopPage } from '@/screens/shop';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/shop` — «فروشگاه» categories (B-N10-03). */
export default async function ShopRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="shop">
      <ShopPage />
    </RouteMessages>
  );
}
