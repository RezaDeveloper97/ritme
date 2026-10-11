import { env } from '@/shared/config';

import {
  shopCatalogHomeSchema,
  shopCatalogListSchema,
  shopCatalogProductSchema,
  type ShopCatalogHome,
  type ShopCatalogList,
  type ShopCatalogProduct,
} from './schema';

/*
 * CB-SHOP-10: server-component reads of the public catalog (no session, no cookies, no user data) for SSR and
 * metadata. `null` for an unknown product / empty catalog or when the API is unreachable — the page then loads
 * client-side. The locale travels as Accept-Language so titles come back in the page language.
 */

async function getData(path: string, locale: string): Promise<unknown> {
  try {
    const res = await fetch(`${env.apiBaseUrl.replace(/\/+$/, '')}${path}`, {
      headers: { Accept: 'application/json', 'Accept-Language': locale },
      next: { revalidate: 300 },
    });
    if (!res.ok) return null;
    return ((await res.json()) as { data?: unknown }).data ?? null;
  } catch {
    return null;
  }
}

export async function fetchPublicShopProduct(id: number, locale: string): Promise<ShopCatalogProduct | null> {
  const data = (await getData(`/shop/catalog/products/${id}`, locale)) as { product?: unknown } | null;
  const parsed = shopCatalogProductSchema.safeParse(data?.product);
  return parsed.success ? parsed.data : null;
}

export async function fetchPublicShopHome(locale: string): Promise<ShopCatalogHome | null> {
  const parsed = shopCatalogHomeSchema.safeParse(await getData('/shop/catalog/home', locale));
  return parsed.success ? parsed.data : null;
}

/** First page of a category, default sort, no filters (the list page's initial state). */
export async function fetchPublicShopList(category: string, locale: string): Promise<ShopCatalogList | null> {
  const qs = new URLSearchParams({ category, sort: 'popular', page: '1', per_page: '20' });
  const parsed = shopCatalogListSchema.safeParse(await getData(`/shop/catalog/products?${qs.toString()}`, locale));
  return parsed.success ? parsed.data : null;
}
