'use client';

import { useShopCatalogProducts } from '@/entities/shop';
import { getApiErrorStatus } from '@/shared/api';

import { ShopCategoryPage as ShopLinkOutCategoryPage } from './ShopCategoryPage';
import { ShopListPage } from './ShopListPage';

/**
 * `/shop/[category]` (CB-SHOP-07): an internal catalog category renders the filtered list; a slug the catalog doesn't
 * know (404) keeps bloom's link-out list (B-N10-03) untouched. The probe query shares its cache key with the list's
 * first render, so there is no extra request.
 */
export function ShopCategoryGate({ category }: { category: string }) {
  const probe = useShopCatalogProducts({ category, sort: 'popular', page: 1 });
  if (probe.isError && getApiErrorStatus(probe.error) === 404) return <ShopLinkOutCategoryPage category={category} />;
  return <ShopListPage category={category} />;
}
