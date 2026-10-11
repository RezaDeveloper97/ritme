import { setRequestLocale } from 'next-intl/server';

import { MyOrdersPage } from '@/screens/my-orders';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/profile/orders` — «سفارش‌ها» in Me (CB-SHOP-08). */
export default async function MyOrdersRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="shop">
      <MyOrdersPage />
    </RouteMessages>
  );
}
