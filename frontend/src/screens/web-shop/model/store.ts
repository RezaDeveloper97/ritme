/** Public desktop shop routes (CB-SHOP-10) and the app screens they map to for signed-in phone-width visitors. */
export const STORE_HOME = '/store';
export const APP_SHOP = '/shop';

export const storeCategoryHref = (slug: string) => `${STORE_HOME}/${slug}`;
export const storeProductHref = (id: number) => `${STORE_HOME}/product/${id}`;

export const FACET_KEYS = ['size', 'color', 'volume', 'material', 'skin_type'] as const;
export type FacetKey = (typeof FACET_KEYS)[number];
export const FACET_SOURCE = { size: 'sizes', color: 'colors', volume: 'volumes', material: 'materials', skin_type: 'skin_types' } as const;

export interface Filters {
  size: string[];
  color: string[];
  volume: string[];
  material: string[];
  skin_type: string[];
  price_min: string;
  price_max: string;
}
export const NO_FILTERS: Filters = { size: [], color: [], volume: [], material: [], skin_type: [], price_min: '', price_max: '' };

/** True while the list is in its initial state, i.e. the server-rendered first page is still the right data. */
export const isDefaultList = (f: Filters, sort: string, page: number): boolean =>
  sort === 'popular' && page === 1 && FACET_KEYS.every((k) => f[k].length === 0) && !f.price_min && !f.price_max;

/** Hero copy key of a root category; the titles themselves come from the server. */
export const heroKey = (slug: string): 'layette' | 'beauty' | 'other' => (slug === 'layette' || slug === 'beauty' ? slug : 'other');
