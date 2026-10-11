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
  badges: z.array(z.string()).catch([]),
  in_stock: z.boolean().catch(true),
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
