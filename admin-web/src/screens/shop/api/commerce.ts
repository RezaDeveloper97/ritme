'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource, pagedSchema, translationsSchema } from '@/shared/api';

/**
 * Internal shop admin (CB-SHOP-05, admin-api.md §25): sellers, commerce fields, variants + stock, delivery slots,
 * the shipment queue, orders, returns, reviews, discount codes and the layette template. The order screens hold the
 * recipient's name, phone and address — the API serves them to the `support` capability only.
 */

const nullableNumber = z.number().nullable();
const write = z.unknown();

// ─── options ────────────────────────────────────────────────────────────────────────────────────────────────────────

const optionsSchema = z.object({
  sellers: z.array(z.object({ id: z.number(), slug: z.string(), name: translationsSchema, is_active: z.boolean() })),
  categories: z.array(
    z.object({ id: z.number(), slug: z.string(), title: translationsSchema, parent_id: nullableNumber, is_active: z.boolean() }),
  ),
  layette_groups: z.array(z.string()),
  skin_types: z.array(z.string()),
  shipment_statuses: z.array(z.string()),
  return_statuses: z.array(z.string()),
  review_statuses: z.array(z.string()),
  discount_kinds: z.array(z.string()),
  low_stock: z.number(),
  image: z.object({ max_kb: z.number(), min_width: z.number(), min_height: z.number(), types: z.array(z.string()), max_count: z.number() }),
});
export type CommerceOptions = z.infer<typeof optionsSchema>;

export function useCommerceOptions() {
  return useQuery({
    queryKey: ['shop-commerce-options'],
    queryFn: ({ signal }) => api.get('/shop/commerce-options', { schema: optionsSchema, signal }),
    staleTime: 60_000,
  });
}

// ─── sellers ────────────────────────────────────────────────────────────────────────────────────────────────────────

export const sellerSchema = z.object({
  id: z.number(),
  slug: z.string(),
  name: translationsSchema,
  is_verified: z.boolean(),
  ship_min_days: z.number(),
  ship_max_days: z.number(),
  discreet_packaging: z.boolean(),
  shipping_fee: z.number(),
  free_shipping_over: nullableNumber,
  cod_max_total: nullableNumber,
  return_days: z.number(),
  sort_order: z.number(),
  is_active: z.boolean(),
  products_count: z.number(),
});
export type Seller = z.infer<typeof sellerSchema>;

export const sellersApi = createResource({
  key: 'shop-sellers',
  path: '/shop/sellers',
  list: z.object({ items: z.array(sellerSchema) }),
  detail: z.object({ seller: sellerSchema }),
  alsoInvalidate: [['shop-commerce-options']],
});

// ─── product commerce + variants ────────────────────────────────────────────────────────────────────────────────────

const textOrTranslations = z.union([z.string(), translationsSchema]);

export const commerceSchema = z.object({
  id: z.number(),
  title: translationsSchema,
  category_id: z.number(),
  seller_id: nullableNumber,
  subtitle: translationsSchema,
  description: translationsSchema,
  images: z.array(z.object({ path: z.string(), url: z.string() })),
  attributes: z.preprocess((v) => (v && typeof v === 'object' && !Array.isArray(v) ? v : {}), z.record(z.string(), textOrTranslations)),
  irc_code: z.string().nullable(),
  expires_on: z.string().nullable(),
  skin_types: z.array(z.string()),
  badges: z.preprocess((v) => (Array.isArray(v) ? v : []), z.array(textOrTranslations)),
  discreet_packaging: z.boolean(),
  rating_avg: z.string().nullable(),
  rating_count: z.number(),
  purchase_count: z.number(),
  variants_count: z.number(),
  is_link_out: z.boolean(),
});
export type Commerce = z.infer<typeof commerceSchema>;

export const variantSchema = z.object({
  id: z.number(),
  product_id: z.number(),
  sku: z.string().nullable(),
  size: translationsSchema,
  color: translationsSchema,
  volume: translationsSchema,
  price: z.number(),
  compare_at_price: nullableNumber,
  stock: z.number(),
  sort_order: z.number(),
  is_active: z.boolean(),
});
export type Variant = z.infer<typeof variantSchema>;

const commerceKey = (id: number) => ['shop-commerce', id] as const;

export function useCommerce(productId: number) {
  return useQuery({
    queryKey: commerceKey(productId),
    queryFn: ({ signal }) =>
      api.get(`/shop/products/${productId}/commerce`, { schema: z.object({ product: commerceSchema }), signal }),
  });
}

export function useSaveCommerce(productId: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: unknown) => api.put(`/shop/products/${productId}/commerce`, body, { schema: write }),
    onSuccess: () => client.invalidateQueries({ queryKey: commerceKey(productId) }),
  });
}

export function useUploadImage(productId: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (file: File) => {
      const body = new FormData();
      body.append('image', file);
      return api.post(`/shop/products/${productId}/images`, body, { schema: write });
    },
    onSuccess: () => client.invalidateQueries({ queryKey: commerceKey(productId) }),
  });
}

/** PUT/DELETE /shop/variants/:id and POST /shop/variants/:id/stock; the per-product list is `useVariants`. */
export const variantsApi = createResource({
  key: 'shop-variants',
  path: '/shop/variants',
  list: z.object({ items: z.array(variantSchema) }),
  detail: z.object({ variant: variantSchema }),
  alsoInvalidate: [['shop-stock'], ['shop-commerce']],
});

export function useVariants(productId: number) {
  return useQuery({
    queryKey: [...variantsApi.keys.all, 'product', productId],
    queryFn: ({ signal }) =>
      api.get(`/shop/products/${productId}/variants`, { schema: z.object({ items: z.array(variantSchema) }), signal }),
  });
}

export function useCreateVariant(productId: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: unknown) => api.post(`/shop/products/${productId}/variants`, body, { schema: write }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: variantsApi.keys.all });
      void client.invalidateQueries({ queryKey: commerceKey(productId) });
    },
  });
}

export const stockRowSchema = z.object({
  id: z.number(),
  product_id: z.number(),
  product_title: translationsSchema,
  seller_id: nullableNumber,
  sku: z.string().nullable(),
  size: translationsSchema,
  color: translationsSchema,
  volume: translationsSchema,
  price: z.number(),
  stock: z.number(),
  is_active: z.boolean(),
  low: z.boolean(),
});
export type StockRow = z.infer<typeof stockRowSchema>;

export const stockApi = createResource({
  key: 'shop-stock',
  path: '/shop/stock',
  list: pagedSchema(stockRowSchema, {}),
  detail: z.unknown(),
});

/** Category tree (sub-category under a root). */
export function useSetCategoryParent() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, parentId }: { id: number; parentId: number | null }) =>
      api.put(`/shop/categories/${id}/parent`, { parent_id: parentId }, { schema: write }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['shop-commerce-options'] });
      void client.invalidateQueries({ queryKey: ['shop-categories'] });
    },
  });
}

// ─── delivery slots ─────────────────────────────────────────────────────────────────────────────────────────────────

export const slotSchema = z.object({
  id: z.number(),
  seller_id: z.number(),
  slot_date: z.string(),
  starts_at: z.string(),
  ends_at: z.string(),
  capacity: nullableNumber,
  booked: z.number(),
  is_active: z.boolean(),
});
export type Slot = z.infer<typeof slotSchema>;

export const slotsApi = createResource({
  key: 'shop-slots',
  path: '/shop/slots',
  list: z.object({ items: z.array(slotSchema), from: z.string(), to: z.string() }),
  detail: z.object({ slot: slotSchema }),
});

export function useGenerateSlots() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: unknown) =>
      api.post('/shop/slots/generate', body, { schema: z.object({ created: z.number() }) }),
    onSuccess: () => client.invalidateQueries({ queryKey: slotsApi.keys.all }),
  });
}

// ─── shipments, orders, returns (support capability) ────────────────────────────────────────────────────────────────

const slotRefSchema = z.object({ date: z.string(), starts_at: z.string(), ends_at: z.string() }).nullable();

export const shipmentRowSchema = z.object({
  id: z.number(),
  order_id: z.number(),
  order_code: z.string(),
  order_status: z.string(),
  seller: z.object({ id: z.number(), name: translationsSchema }),
  status: z.string(),
  allowed_statuses: z.array(z.string()),
  slot: slotRefSchema,
  items_total: z.number(),
  shipping_fee: z.number(),
  payable_total: z.number(),
  discreet: z.boolean(),
  tracking_code: z.string().nullable(),
  recipient_name: z.string(),
  city: z.string(),
  delivered_at: z.string().nullable(),
  created_at: z.string().nullable(),
});
export type ShipmentRow = z.infer<typeof shipmentRowSchema>;

export const shipmentsApi = createResource({
  key: 'shop-shipments',
  path: '/shop/shipments',
  list: pagedSchema(shipmentRowSchema, {}),
  detail: z.unknown(),
  alsoInvalidate: [['shop-orders']],
});

const orderSchema = z.object({
  id: z.number(),
  code: z.string(),
  status: z.string(),
  payment_method: z.string(),
  payment_status: z.string(),
  discreet: z.boolean(),
  recipient_name: z.string(),
  recipient_phone: z.string(),
  city: z.string(),
  address_text: z.string(),
  postal_code: z.string().nullable(),
  items_total: z.number(),
  shipping_total: z.number(),
  discount_total: z.number(),
  payable_total: z.number(),
  placed_at: z.string().nullable(),
  cancelled_at: z.string().nullable(),
  can_cancel: z.boolean(),
  shipments: z.array(
    z.object({
      id: z.number(),
      seller: z.object({ id: z.number(), name: translationsSchema }),
      status: z.string(),
      allowed_statuses: z.array(z.string()),
      slot: slotRefSchema,
      items_total: z.number(),
      shipping_fee: z.number(),
      tracking_code: z.string().nullable(),
      delivered_at: z.string().nullable(),
      items: z.array(
        z.object({
          id: z.number(),
          title: translationsSchema,
          variant_label: translationsSchema,
          unit_price: z.number(),
          quantity: z.number(),
          line_total: z.number(),
        }),
      ),
    }),
  ),
  events: z.array(z.object({ id: z.number(), shipment_id: nullableNumber, status: z.string(), created_at: z.string().nullable() })),
  returns: z.array(
    z.object({ id: z.number(), order_item_id: z.number(), quantity: z.number(), reason: z.string(), status: z.string() }),
  ),
});
export type AdminOrder = z.infer<typeof orderSchema>;

export const ordersApi = createResource({
  key: 'shop-orders',
  path: '/shop/orders',
  list: z.unknown(),
  detail: z.object({ order: orderSchema }),
  alsoInvalidate: [['shop-shipments']],
});

export const returnRowSchema = z.object({
  id: z.number(),
  order_id: z.number(),
  order_code: z.string(),
  item_title: translationsSchema,
  variant_label: translationsSchema,
  unit_price: z.number(),
  quantity: z.number(),
  reason: z.string(),
  note: z.string().nullable(),
  status: z.string(),
  can_restock: z.boolean(),
  created_at: z.string().nullable(),
});
export type ReturnRow = z.infer<typeof returnRowSchema>;

export const returnsApi = createResource({
  key: 'shop-returns',
  path: '/shop/returns',
  list: pagedSchema(returnRowSchema, {}),
  detail: z.unknown(),
  alsoInvalidate: [['shop-orders'], ['shop-stock']],
});

// ─── reviews, discount codes, layette template (content) ────────────────────────────────────────────────────────────

export const reviewSchema = z.object({
  id: z.number(),
  product_id: z.number(),
  product_title: translationsSchema,
  author_name: z.string(),
  variant_label: translationsSchema,
  rating: z.number(),
  body: z.string().nullable(),
  status: z.string(),
  created_at: z.string().nullable(),
});
export type Review = z.infer<typeof reviewSchema>;

export const reviewsApi = createResource({
  key: 'shop-reviews',
  path: '/shop/reviews',
  list: pagedSchema(reviewSchema, {}),
  detail: z.unknown(),
});

export const discountCodeSchema = z.object({
  id: z.number(),
  code: z.string(),
  kind: z.string(),
  value: z.number(),
  max_discount: nullableNumber,
  min_items_total: z.number(),
  max_uses: nullableNumber,
  used_count: z.number(),
  per_user_limit: nullableNumber,
  starts_at: z.string().nullable(),
  ends_at: z.string().nullable(),
  is_active: z.boolean(),
});
export type DiscountCode = z.infer<typeof discountCodeSchema>;

export const discountCodesApi = createResource({
  key: 'shop-discount-codes',
  path: '/shop/discount-codes',
  list: pagedSchema(discountCodeSchema, {}),
  detail: z.object({ discount_code: discountCodeSchema }),
});

export const layetteItemSchema = z.object({
  id: z.number(),
  group_key: z.string(),
  title: translationsSchema,
  note: translationsSchema,
  suggested_qty: nullableNumber,
  is_essential: z.boolean(),
  is_first_days: z.boolean(),
  category_id: nullableNumber,
  sort_order: z.number(),
  is_active: z.boolean(),
});
export type LayetteItem = z.infer<typeof layetteItemSchema>;

export const layetteApi = createResource({
  key: 'shop-layette',
  path: '/shop/layette-items',
  list: z.object({ items: z.array(layetteItemSchema) }),
  detail: z.object({ item: layetteItemSchema }),
});
