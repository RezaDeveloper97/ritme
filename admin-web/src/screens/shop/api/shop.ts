'use client';

import { z } from 'zod';

import { createResource, pagedSchema, translationsSchema } from '@/shared/api';

/** Shop catalog (admin-api.md §25, D-91). */
export const shopCategorySchema = z.object({
  id: z.number(),
  slug: z.string(),
  title: translationsSchema,
  icon: z.string().nullable(),
  sort_order: z.number(),
  is_active: z.boolean(),
  products_count: z.number(),
});
export type ShopCategory = z.infer<typeof shopCategorySchema>;

export const shopProductSchema = z.object({
  id: z.number(),
  category_id: z.number(),
  category_slug: z.string(),
  title: translationsSchema,
  image_path: z.string().nullable(),
  image_url: z.string().nullable(),
  price_text: z.string().nullable(),
  badge: z.string().nullable(),
  external_url: z.string().nullable(),
  partner_label: z.string().nullable(),
  sort_order: z.number(),
  is_active: z.boolean(),
  clicks_30d: z.number(),
});
export type ShopProduct = z.infer<typeof shopProductSchema>;

const optionsSchema = z.object({
  categories: z.array(shopCategorySchema),
  image: z.object({ max_kb: z.number(), min_width: z.number(), min_height: z.number(), types: z.array(z.string()) }),
});
export type ShopOptions = z.infer<typeof optionsSchema>;

export const shopCategoriesApi = createResource({
  key: 'shop-categories',
  path: '/shop/categories',
  list: z.object({ items: z.array(shopCategorySchema) }),
  detail: z.object({ category: shopCategorySchema }),
  alsoInvalidate: [['shop-products']],
});

export const shopProductsApi = createResource({
  key: 'shop-products',
  path: '/shop/products',
  list: pagedSchema(shopProductSchema, {}),
  detail: z.object({ product: shopProductSchema }),
  options: optionsSchema,
  alsoInvalidate: [['shop-categories']],
});
