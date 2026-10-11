import { z } from 'zod';

/*
 * Parsers of /api/v1/shop/cart, /addresses, /checkout, /orders, /returns (CB-SHOP-03 → CB-SHOP-08,
 * backend-go/api/openapi.yaml). Snake_case = wire contract. Addresses, phones and recipients are PII: never log them.
 */

export const sellerTermsSchema = z.object({
  id: z.number(),
  slug: z.string().catch(''),
  name: z.string(),
  logo_url: z.string().nullable().catch(null),
  is_verified: z.boolean().catch(false),
  ship_min_days: z.number().catch(0),
  ship_max_days: z.number().catch(0),
  discreet_packaging: z.boolean().catch(false),
  shipping_fee: z.number().catch(0),
  free_shipping_over: z.number().nullable().catch(null),
  cod_max_total: z.number().nullable().catch(null),
  return_days: z.number().catch(0),
});
export type ShopSellerTerms = z.infer<typeof sellerTermsSchema>;

export const cartLineSchema = z.object({
  variant_id: z.number(),
  product_id: z.number(),
  title: z.string(),
  image_url: z.string().nullable().catch(null),
  variant_label: z.string().nullable().catch(null),
  unit_price: z.number().catch(0),
  compare_at_price: z.number().nullable().catch(null),
  quantity: z.number().catch(1),
  line_total: z.number().catch(0),
  stock: z.number().catch(0),
  status: z.enum(['ok', 'out_of_stock', 'unavailable']).catch('unavailable'),
  discreet_packaging: z.boolean().catch(false),
});
export type ShopCartLine = z.infer<typeof cartLineSchema>;

export const shopCartSchema = z.object({
  items_count: z.number().catch(0),
  sellers_count: z.number().catch(0),
  groups: z
    .array(
      z.object({
        seller: sellerTermsSchema,
        items_count: z.number().catch(0),
        items_total: z.number().catch(0),
        shipping_fee: z.number().catch(0),
        free_shipping_remaining: z.number().nullable().catch(null),
        items: z.array(cartLineSchema).catch([]),
      }),
    )
    .catch([]),
  unavailable_items: z.array(cartLineSchema).catch([]),
  discount: z
    .object({ code: z.string(), amount: z.number().catch(0), error_code: z.string().nullable().catch(null) })
    .nullable()
    .catch(null),
  items_total: z.number().catch(0),
  discount_total: z.number().catch(0),
  payable_total: z.number().catch(0),
});
export type ShopCart = z.infer<typeof shopCartSchema>;
export const shopCartEnvelopeSchema = z.object({ cart: shopCartSchema });

export const shopAddressSchema = z.object({
  id: z.number(),
  label: z.string().nullable().catch(null),
  recipient_name: z.string(),
  recipient_phone: z.string(),
  province: z.string().nullable().catch(null),
  city: z.string(),
  address_line: z.string(),
  postal_code: z.string().nullable().catch(null),
  is_default: z.boolean().catch(false),
});
export type ShopAddress = z.infer<typeof shopAddressSchema>;
export const shopAddressEnvelopeSchema = z.object({ address: shopAddressSchema });

export interface ShopAddressInput {
  label?: string | null;
  recipient_name: string;
  recipient_phone: string;
  province?: string | null;
  city: string;
  address_line: string;
  postal_code?: string | null;
  is_default?: boolean;
}

export const shopCheckoutSchema = z.object({
  cart: shopCartSchema,
  addresses: z.array(shopAddressSchema).catch([]),
  default_address_id: z.number().nullable().catch(null),
  shipments: z
    .array(
      z.object({
        seller: sellerTermsSchema,
        items_count: z.number().catch(0),
        items_total: z.number().catch(0),
        shipping_fee: z.number().catch(0),
        cod_available: z.boolean().catch(true),
        slots: z
          .array(z.object({ id: z.number(), date: z.string(), starts_at: z.string(), ends_at: z.string() }))
          .catch([]),
      }),
    )
    .catch([]),
  discreet_packaging_available: z.boolean().catch(false),
  payment_methods: z.array(z.object({ method: z.string(), available: z.boolean().catch(true) })).catch([]),
  items_total: z.number().catch(0),
  discount_total: z.number().catch(0),
  shipping_total: z.number().catch(0),
  payable_total: z.number().catch(0),
  can_place: z.boolean().catch(false),
  blockers: z.array(z.string()).catch([]),
});
export type ShopCheckout = z.infer<typeof shopCheckoutSchema>;

const orderSellerSchema = z
  .object({ id: z.number(), slug: z.string().catch(''), name: z.string(), logo_url: z.string().nullable().catch(null) })
  .nullable()
  .catch(null);
const orderSlotSchema = z.object({ date: z.string(), starts_at: z.string(), ends_at: z.string() }).nullable().catch(null);
export type ShopOrderSlot = z.infer<typeof orderSlotSchema>;

export const orderItemSchema = z.object({
  id: z.number(),
  product_id: z.number().nullable().catch(null),
  variant_id: z.number().nullable().catch(null),
  title: z.string(),
  variant_label: z.string().nullable().catch(null),
  image_url: z.string().nullable().catch(null),
  unit_price: z.number().catch(0),
  quantity: z.number().catch(1),
  line_total: z.number().catch(0),
  returnable: z.boolean().catch(false),
});
export type ShopOrderItem = z.infer<typeof orderItemSchema>;

const orderEventSchema = z.object({ status: z.string(), at: z.string().nullable().catch(null) });
export type ShopOrderEvent = z.infer<typeof orderEventSchema>;

export const shopReturnSchema = z.object({
  id: z.number(),
  order_id: z.number(),
  order_item_id: z.number(),
  quantity: z.number().catch(1),
  reason: z.string(),
  note: z.string().nullable().catch(null),
  status: z.string(),
  created_at: z.string().nullable().catch(null),
});
export type ShopReturn = z.infer<typeof shopReturnSchema>;
export const shopReturnEnvelopeSchema = z.object({ return: shopReturnSchema });

export const shopOrderSchema = z.object({
  id: z.number(),
  code: z.string(),
  status: z.string(),
  payment_method: z.string().catch('cod'),
  payment_status: z.string().catch(''),
  discreet: z.boolean().catch(false),
  can_cancel: z.boolean().catch(false),
  recipient: z.object({
    name: z.string().catch(''),
    phone: z.string().catch(''),
    city: z.string().catch(''),
    address: z.string().catch(''),
    postal_code: z.string().nullable().catch(null),
  }),
  items_total: z.number().catch(0),
  shipping_total: z.number().catch(0),
  discount_total: z.number().catch(0),
  payable_total: z.number().catch(0),
  placed_at: z.string().nullable().catch(null),
  cancelled_at: z.string().nullable().catch(null),
  shipments: z
    .array(
      z.object({
        id: z.number(),
        seller: orderSellerSchema,
        status: z.string(),
        slot: orderSlotSchema,
        items_total: z.number().catch(0),
        shipping_fee: z.number().catch(0),
        tracking_code: z.string().nullable().catch(null),
        delivered_at: z.string().nullable().catch(null),
        return_until: z.string().nullable().catch(null),
        items: z.array(orderItemSchema).catch([]),
        timeline: z.array(orderEventSchema).catch([]),
      }),
    )
    .catch([]),
  timeline: z.array(orderEventSchema).catch([]),
  returns: z.array(shopReturnSchema).catch([]),
  checklist_on_the_way: z.number().catch(0),
});
export type ShopOrder = z.infer<typeof shopOrderSchema>;
export const shopOrderEnvelopeSchema = z.object({ order: shopOrderSchema, replayed: z.boolean().catch(false) });

export const shopOrderSummarySchema = z.object({
  id: z.number(),
  code: z.string(),
  status: z.string(),
  payment_method: z.string().catch('cod'),
  payable_total: z.number().catch(0),
  items_count: z.number().catch(0),
  placed_at: z.string().nullable().catch(null),
  shipments: z
    .array(z.object({ id: z.number(), seller: orderSellerSchema, status: z.string(), slot: orderSlotSchema }))
    .catch([]),
  items: z.array(orderItemSchema).catch([]),
});
export type ShopOrderSummary = z.infer<typeof shopOrderSummarySchema>;
export const shopOrdersSchema = z.object({ orders: z.array(shopOrderSummarySchema).catch([]) });

export const RETURN_REASONS = ['wrong_size', 'damaged', 'not_as_described', 'changed_mind', 'other'] as const;
export type ReturnReason = (typeof RETURN_REASONS)[number];
