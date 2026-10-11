'use client';

import { keepPreviousData, useInfiniteQuery, useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient, getApiErrorStatus } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { shopKeys } from './keys';
import {
  shopCatalogHomeSchema,
  shopCatalogListSchema,
  shopCatalogProductSchema,
  shopCatalogReviewsSchema,
  shopCategoriesSchema,
  shopProductsSchema,
  type ShopSort,
} from './schema';

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
 * GET /shop/catalog/home?root= — the internal-catalog home of one root (CB-SHOP-06). `root` '' = the server's first
 * root. A teen-mode account gets 403 `teen_commercial_blocked`, an empty catalog 404 — the caller reads the error
 * status (no retries for those).
 */
export const useShopCatalogHome = (root: string) =>
  useQuery({
    queryKey: shopKeys.catalogHome(root),
    queryFn: async () =>
      shopCatalogHomeSchema.parse(
        (await apiClient.get<ApiEnvelope<unknown>>('/shop/catalog/home', { params: root ? { root } : undefined })).data.data,
      ),
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: (count, error) => ![403, 404].includes(getApiErrorStatus(error) ?? 0) && count < 2,
  });

const noRetryFor = (count: number, error: unknown) => ![403, 404].includes(getApiErrorStatus(error) ?? 0) && count < 2;

/** Filters of the catalog list (CB-SHOP-07); multi-valued filters are comma-joined by the caller. */
export interface ShopListParams {
  category: string;
  q?: string;
  size?: string;
  color?: string;
  volume?: string;
  material?: string;
  skin_type?: string;
  price_min?: number;
  price_max?: number;
  sort?: ShopSort;
  page: number;
}

/**
 * GET /shop/catalog/products — filtered, sorted page of one catalog category (CB-SHOP-07). 404 = not an internal
 * category (the caller falls back to bloom's link-out list), 403 = teen mode.
 */
export const useShopCatalogProducts = (params: ShopListParams) =>
  useQuery({
    queryKey: shopKeys.catalogList(params),
    queryFn: async () =>
      shopCatalogListSchema.parse(
        (await apiClient.get<ApiEnvelope<unknown>>('/shop/catalog/products', { params: { ...params, per_page: 20 } })).data.data,
      ),
    enabled: isAuthenticated(),
    placeholderData: keepPreviousData,
    staleTime: 60_000,
    retry: noRetryFor,
  });

/** GET /shop/catalog/products/:id — the product page (CB-SHOP-07). */
export const useShopCatalogProduct = (id: number) =>
  useQuery({
    queryKey: shopKeys.catalogProduct(id),
    queryFn: async () =>
      shopCatalogProductSchema.parse((await apiClient.get<ApiEnvelope<{ product: unknown }>>(`/shop/catalog/products/${id}`)).data.data?.product),
    enabled: isAuthenticated() && Number.isFinite(id),
    staleTime: 60_000,
    retry: noRetryFor,
  });

/** GET /shop/catalog/products/:id/reviews — all published reviews, loaded page by page once «همه» is opened. */
export const useShopCatalogReviews = (id: number, enabled: boolean) =>
  useInfiniteQuery({
    queryKey: shopKeys.catalogReviews(id),
    queryFn: async ({ pageParam }) =>
      shopCatalogReviewsSchema.parse(
        (await apiClient.get<ApiEnvelope<unknown>>(`/shop/catalog/products/${id}/reviews`, { params: { page: pageParam } })).data.data,
      ),
    initialPageParam: 1,
    getNextPageParam: (last) => (last.meta.current_page < last.meta.last_page ? last.meta.current_page + 1 : undefined),
    enabled: enabled && isAuthenticated(),
    staleTime: 60_000,
  });

/**
 * POST /shop/products/:id/click — counts one link-out click (per product and day on the server; no user data is
 * stored). Fire and forget: the outgoing link opens regardless of the result.
 */
export function countShopClick(id: number): void {
  void apiClient.post(`/shop/products/${id}/click`).catch(() => undefined);
}
