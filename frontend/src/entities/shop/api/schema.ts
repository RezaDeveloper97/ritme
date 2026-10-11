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
