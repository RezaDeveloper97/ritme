import { z } from 'zod';

import { isIconName, type IconName } from '@/shared/ui';

/*
 * Parsers of /api/v1/shop/* (bloom B-N10-03, backend-go/api/openapi.yaml). Snake_case = wire contract. Content is
 * admin-edited: unknown icons fall back, optional text is null. The shop never carries health data.
 */

const text = z.string().nullable().catch(null);

export const shopCategorySchema = z.object({
  id: z.number(),
  slug: z.string(),
  title: z.string(),
  icon: z
    .string()
    .nullable()
    .catch(null)
    .transform((v): IconName => (v && isIconName(v) ? v : 'box')),
  product_count: z.number().catch(0),
});
export type ShopCategory = z.infer<typeof shopCategorySchema>;

export const shopProductSchema = z.object({
  id: z.number(),
  category: z.string(),
  title: z.string(),
  image_url: text,
  price_text: text,
  badge: text,
  partner_label: text,
  link_url: text,
});
export type ShopProduct = z.infer<typeof shopProductSchema>;

export const shopCategoriesSchema = z.object({ categories: z.array(shopCategorySchema) });

export const shopProductsSchema = z.object({
  category: shopCategorySchema.nullable().catch(null),
  products: z.array(shopProductSchema),
  meta: z.object({ current_page: z.number(), last_page: z.number(), total: z.number() }).catch({ current_page: 1, last_page: 1, total: 0 }),
});

/*
 * CB-SHOP-06 — parsers of GET /shop/catalog/home (CB-SHOP-02, internal catalog; money = integer toman). Additive to
 * the bloom link-out parsers above.
 */

export const shopCatalogCategorySchema = z.object({
  id: z.number(),
  slug: z.string(),
  title: z.string(),
  icon: z
    .string()
    .nullable()
    .catch(null)
    .transform((v): IconName => (v && isIconName(v) ? v : 'box')),
  parent: z.string().nullable().catch(null),
  product_count: z.number().catch(0),
});
export type ShopCatalogCategory = z.infer<typeof shopCatalogCategorySchema>;

export const shopCatalogCardSchema = z.object({
  id: z.number(),
  title: z.string(),
  subtitle: text,
  category: z.string(),
  image_url: text,
  price: z.number().catch(0),
  compare_at_price: z.number().nullable().catch(null),
  rating_avg: z.number().nullable().catch(null),
  rating_count: z.number().catch(0),
  purchase_count: z.number().catch(0), // CB-SHOP-07
  badges: z.array(z.string()).catch([]),
  in_stock: z.boolean().catch(true),
  discreet_packaging: z.boolean().catch(false), // CB-SHOP-07
  seller: z
    .object({ name: z.string(), is_verified: z.boolean().catch(false) })
    .nullable()
    .catch(null),
});
export type ShopCatalogCard = z.infer<typeof shopCatalogCardSchema>;

export const shopCatalogHomeSchema = z.object({
  root: shopCatalogCategorySchema,
  roots: z.array(shopCatalogCategorySchema).catch([]),
  sub_categories: z.array(shopCatalogCategorySchema).catch([]),
  sections: z
    .array(z.object({ category: shopCatalogCategorySchema, products: z.array(shopCatalogCardSchema).catch([]) }))
    .catch([]),
});
export type ShopCatalogHome = z.infer<typeof shopCatalogHomeSchema>;

/*
 * CB-SHOP-07 — parsers of GET /shop/catalog/products (list + filters), /products/:id (detail) and /products/:id/reviews
 * (CB-SHOP-02). Additive. Money = integer toman.
 */

export const shopCatalogFacetSchema = z.object({ value: z.string(), label: z.string(), count: z.number().catch(0) });
export type ShopCatalogFacet = z.infer<typeof shopCatalogFacetSchema>;

const facets = z.array(shopCatalogFacetSchema).catch([]);

export const SHOP_SORTS = ['popular', 'rating', 'newest', 'price_asc', 'price_desc'] as const;
export type ShopSort = (typeof SHOP_SORTS)[number];

const pageMeta = z
  .object({ current_page: z.number().catch(1), last_page: z.number().catch(1), total: z.number().catch(0) })
  .catch({ current_page: 1, last_page: 1, total: 0 });

export const shopCatalogListSchema = z.object({
  category: shopCatalogCategorySchema.nullable().catch(null),
  sub_categories: z.array(shopCatalogCategorySchema).catch([]),
  filters: z
    .object({
      sizes: facets,
      colors: facets,
      volumes: facets,
      materials: facets,
      skin_types: facets,
      price: z.object({ min: z.number().nullable().catch(null), max: z.number().nullable().catch(null) }).catch({ min: null, max: null }),
    })
    .catch({ sizes: [], colors: [], volumes: [], materials: [], skin_types: [], price: { min: null, max: null } }),
  products: z.array(shopCatalogCardSchema),
  meta: pageMeta,
});
export type ShopCatalogList = z.infer<typeof shopCatalogListSchema>;

export const shopCatalogReviewSchema = z.object({
  id: z.number(),
  author_name: z.string().catch(''),
  rating: z.number().catch(5),
  body: text,
  variant_label: text,
  verified_purchase: z.boolean().catch(false),
  created_at: text,
});
export type ShopCatalogReview = z.infer<typeof shopCatalogReviewSchema>;

export const shopCatalogVariantSchema = z.object({
  id: z.number(),
  size: text,
  color: text,
  volume: text,
  label: z.string().catch(''),
  price: z.number().catch(0),
  compare_at_price: z.number().nullable().catch(null),
  stock: z.number().catch(0),
  in_stock: z.boolean().catch(false),
});
export type ShopCatalogVariant = z.infer<typeof shopCatalogVariantSchema>;

export const shopCatalogProductSchema = shopCatalogCardSchema.extend({
  description: text,
  images: z.array(z.string()).catch([]),
  attributes: z.record(z.string(), z.string()).catch({}),
  irc_code: text,
  expires_on: text,
  skin_types: z.array(z.string()).catch([]),
  seller: z
    .object({
      name: z.string(),
      is_verified: z.boolean().catch(false),
      ship_min_days: z.number().catch(0),
      ship_max_days: z.number().catch(0),
      shipping_fee: z.number().catch(0),
      free_shipping_over: z.number().nullable().catch(null),
      return_days: z.number().catch(0),
      discreet_packaging: z.boolean().catch(false),
    })
    .nullable()
    .catch(null),
  variants: z.array(shopCatalogVariantSchema).catch([]),
  default_variant_id: z.number().nullable().catch(null),
  reviews: z
    .object({
      rating_avg: z.number().nullable().catch(null),
      rating_count: z.number().catch(0),
      items: z.array(shopCatalogReviewSchema).catch([]),
    })
    .catch({ rating_avg: null, rating_count: 0, items: [] }),
  bought_together: z.array(shopCatalogCardSchema).catch([]),
});
export type ShopCatalogProduct = z.infer<typeof shopCatalogProductSchema>;

export const shopCatalogReviewsSchema = z.object({ reviews: z.array(shopCatalogReviewSchema), meta: pageMeta });
