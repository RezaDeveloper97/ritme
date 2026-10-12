import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { ShopCategoryPage } from '@/screens/shop';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; category: string }>;
}

/** `/shop/[category]` — the products of one category and the link-out (B-N10-03). */
export default async function ShopCategoryRoute({ params }: Props) {
  const { locale, category } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="shop">
      {/* The list reads its filters from the query string (useSearchParams needs a Suspense boundary). */}
      <Suspense>
        <ShopCategoryPage category={category} />
      </Suspense>
    </RouteMessages>
  );
}
