import type { ShopListParams } from '../api/queries';
import { SHOP_SORTS, type ShopSort } from '../api/schema';

/** The multi-valued facets and the two price bounds of a catalog list. */
export interface ShopListFilters {
  size: string[];
  color: string[];
  volume: string[];
  material: string[];
  skin_type: string[];
  price_min: string;
  price_max: string;
}

export const NO_LIST_FILTERS: ShopListFilters = { size: [], color: [], volume: [], material: [], skin_type: [], price_min: '', price_max: '' };

/** Everything a catalog list shows beyond its category: it lives in the query string (2026-10-12). */
export interface ShopListState {
  q: string;
  filters: ShopListFilters;
  sort: ShopSort;
  page: number;
}

export const DEFAULT_LIST_STATE: ShopListState = { q: '', filters: NO_LIST_FILTERS, sort: 'popular', page: 1 };

const FACETS = ['size', 'color', 'volume', 'material', 'skin_type'] as const;
const csv = (v: string | null | undefined) =>
  v
    ? v
        .split(',')
        .map((x) => x.trim())
        .filter(Boolean)
    : [];
const digits = (v: string | null | undefined) => (v ?? '').replace(/\D/g, '').slice(0, 9);

/** Reads a list state from query-string values (unknown / malformed values fall back to the defaults). */
export function parseShopListState(get: (key: string) => string | null | undefined): ShopListState {
  const sort = get('sort');
  const page = Number.parseInt(get('page') ?? '', 10);
  return {
    q: (get('q') ?? '').trim().slice(0, 100),
    filters: {
      size: csv(get('size')),
      color: csv(get('color')),
      volume: csv(get('volume')),
      material: csv(get('material')),
      skin_type: csv(get('skin_type')),
      price_min: digits(get('price_min')),
      price_max: digits(get('price_max')),
    },
    sort: (SHOP_SORTS as readonly string[]).includes(sort ?? '') ? (sort as ShopSort) : 'popular',
    page: Number.isFinite(page) && page > 0 ? Math.min(page, 10_000) : 1,
  };
}

/** Next's `searchParams` object (first value of a repeated key) → a list state. */
export function parseShopListRecord(sp: Record<string, string | string[] | undefined>): ShopListState {
  return parseShopListState((k) => {
    const v = sp[k];
    return Array.isArray(v) ? v[0] : v;
  });
}

/** `?size=S,M&sort=price_asc` — only non-default parts; `''` for the default list. Stable key order. */
export function shopListSearch(s: ShopListState): string {
  const qs = new URLSearchParams();
  if (s.q) qs.set('q', s.q);
  for (const k of FACETS) if (s.filters[k].length) qs.set(k, s.filters[k].join(','));
  if (s.filters.price_min) qs.set('price_min', s.filters.price_min);
  if (s.filters.price_max) qs.set('price_max', s.filters.price_max);
  if (s.sort !== 'popular') qs.set('sort', s.sort);
  if (s.page > 1) qs.set('page', String(s.page));
  const out = qs.toString();
  return out ? `?${out}` : '';
}

/** The API params of a list state. */
export function shopListParams(category: string, s: ShopListState): ShopListParams {
  return {
    category,
    q: s.q || undefined,
    size: s.filters.size.join(',') || undefined,
    color: s.filters.color.join(',') || undefined,
    volume: s.filters.volume.join(',') || undefined,
    material: s.filters.material.join(',') || undefined,
    skin_type: s.filters.skin_type.join(',') || undefined,
    price_min: s.filters.price_min ? Number(s.filters.price_min) : undefined,
    price_max: s.filters.price_max ? Number(s.filters.price_max) : undefined,
    sort: s.sort,
    page: s.page,
  };
}
