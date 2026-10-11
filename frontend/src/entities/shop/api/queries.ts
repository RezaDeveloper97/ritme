'use client';

import { keepPreviousData, useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { shopKeys } from './keys';
import { shopCategoriesSchema, shopProductsSchema } from './schema';

/** GET /shop/categories — the active categories (empty for a teen-mode account). */
export const useShopCategories = () =>
  useQuery({
    queryKey: shopKeys.categories(),
    queryFn: async () => shopCategoriesSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/shop/categories')).data.data),
    enabled: isAuthenticated(),
    staleTime: 60_000,
  });

/** GET /shop/products?category=&page= — one page (20) of a category. */
export const useShopProducts = (category: string, page: number) =>
  useQuery({
    queryKey: shopKeys.products(category, page),
    queryFn: async () =>
      shopProductsSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/shop/products', { params: { category, page } })).data.data),
    enabled: isAuthenticated(),
    placeholderData: keepPreviousData,
    staleTime: 60_000,
  });

/**
 * POST /shop/products/:id/click — counts one link-out click (per product and day on the server; no user data is
 * stored). Fire and forget: the outgoing link opens regardless of the result.
 */
export function countShopClick(id: number): void {
  void apiClient.post(`/shop/products/${id}/click`).catch(() => undefined);
}
