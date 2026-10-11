'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient, getApiErrorCode, getApiErrorStatus } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { shopKeys } from './keys';
import {
  type ReturnReason,
  type ShopAddressInput,
  shopAddressEnvelopeSchema,
  shopCartEnvelopeSchema,
  shopCheckoutSchema,
  shopOrderEnvelopeSchema,
  shopOrdersSchema,
  shopReturnEnvelopeSchema,
  shopReviewEnvelopeSchema,
} from './order-schema';

const noRetry = (count: number, error: unknown) => ![403, 404].includes(getApiErrorStatus(error) ?? 0) && count < 2;

/** The server's refusal code (`error_code`) of a failed cart / checkout call, or null. */
export const shopErrorCode = (error: unknown): string | null => getApiErrorCode(error) ?? null;

const ERROR_KEYS = [
  'variant_unavailable', 'out_of_stock', 'quantity_limit', 'cart_full', 'discount_invalid', 'discount_exhausted', 'discount_used',
  'discount_min_total', 'slot_full', 'slot_invalid', 'cod_limit_exceeded', 'address_invalid', 'cart_unavailable', 'cart_empty', 'busy',
  'not_cancellable', 'too_many_open_orders', 'not_reviewable', 'already_reviewed',
] as const;

/** Message key under `shop.errors.*` for a failed cart / checkout / order call (`generic` for anything unknown). */
export function shopCodeKey(code: string | null | undefined): (typeof ERROR_KEYS)[number] | 'generic' {
  return (ERROR_KEYS as readonly string[]).includes(code ?? '') ? (code as (typeof ERROR_KEYS)[number]) : 'generic';
}

export const shopErrorKey = (error: unknown) => shopCodeKey(getApiErrorCode(error));

/** GET /shop/cart — grouped per seller. 403 = teen-mode account. */
export const useShopCart = () =>
  useQuery({
    queryKey: shopKeys.cart(),
    queryFn: async () => shopCartEnvelopeSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/shop/cart')).data.data).cart,
    enabled: isAuthenticated(),
    staleTime: 15_000,
    retry: noRetry,
  });

function useAfterCartChange() {
  const qc = useQueryClient();
  return (data: unknown) => {
    qc.setQueryData(shopKeys.cart(), shopCartEnvelopeSchema.parse(data).cart);
    void qc.invalidateQueries({ queryKey: shopKeys.checkout() });
  };
}

/** POST /shop/cart/items — add a variant (the product page's «افزودن به سبد»). */
export function useAddToCart() {
  const after = useAfterCartChange();
  return useMutation({
    mutationFn: async (v: { variantId: number; quantity?: number }) =>
      (await apiClient.post<ApiEnvelope<unknown>>('/shop/cart/items', { variant_id: v.variantId, quantity: v.quantity ?? 1 })).data.data,
    onSuccess: after,
  });
}

/** PUT /shop/cart/items/:variant_id — set a line's quantity. */
export function useSetCartQuantity() {
  const after = useAfterCartChange();
  return useMutation({
    mutationFn: async (v: { variantId: number; quantity: number }) =>
      (await apiClient.put<ApiEnvelope<unknown>>(`/shop/cart/items/${v.variantId}`, { quantity: v.quantity })).data.data,
    onSuccess: after,
  });
}

/** DELETE /shop/cart/items/:variant_id. */
export function useRemoveCartItem() {
  const after = useAfterCartChange();
  return useMutation({
    mutationFn: async (variantId: number) => (await apiClient.delete<ApiEnvelope<unknown>>(`/shop/cart/items/${variantId}`)).data.data,
    onSuccess: after,
  });
}

/** PUT /shop/cart/discount — «کد تخفیف داری؟» (422 + error_code when refused). */
export function useApplyDiscount() {
  const after = useAfterCartChange();
  return useMutation({
    mutationFn: async (code: string) => (await apiClient.put<ApiEnvelope<unknown>>('/shop/cart/discount', { code })).data.data,
    onSuccess: after,
  });
}

/** DELETE /shop/cart/discount. */
export function useRemoveDiscount() {
  const after = useAfterCartChange();
  return useMutation({
    mutationFn: async () => (await apiClient.delete<ApiEnvelope<unknown>>('/shop/cart/discount')).data.data,
    onSuccess: after,
  });
}

/** GET /shop/checkout — the review step (addresses, slots per seller, totals, blockers). */
export const useShopCheckout = () =>
  useQuery({
    queryKey: shopKeys.checkout(),
    queryFn: async () => shopCheckoutSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/shop/checkout')).data.data),
    enabled: isAuthenticated(),
    staleTime: 0,
    retry: noRetry,
  });

/** POST/PUT /shop/addresses[/:id] — create (no id) or edit an address. Never logged. */
export function useSaveAddress() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (v: { id?: number; input: ShopAddressInput }) => {
      const res = v.id
        ? await apiClient.put<ApiEnvelope<unknown>>(`/shop/addresses/${v.id}`, v.input)
        : await apiClient.post<ApiEnvelope<unknown>>('/shop/addresses', v.input);
      return shopAddressEnvelopeSchema.parse(res.data.data).address;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: shopKeys.checkout() }),
  });
}

/** DELETE /shop/addresses/:id. */
export function useDeleteAddress() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await apiClient.delete(`/shop/addresses/${id}`);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: shopKeys.checkout() }),
  });
}

export interface PlaceOrderInput {
  /** One per checkout attempt; reuse it on retry so a double tap never places two orders. */
  idempotencyKey: string;
  addressId: number;
  discreet: boolean;
  slots: { seller_id: number; slot_id: number }[];
}

/** POST /shop/orders — COD only. Same key again answers 200 `replayed: true` with the first order. */
export function usePlaceOrder() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (v: PlaceOrderInput) =>
      shopOrderEnvelopeSchema.parse(
        (
          await apiClient.post<ApiEnvelope<unknown>>('/shop/orders', {
            idempotency_key: v.idempotencyKey,
            address_id: v.addressId,
            payment_method: 'cod',
            discreet: v.discreet,
            slots: v.slots,
          })
        ).data.data,
      ).order,
    onSuccess: (order) => {
      qc.setQueryData(shopKeys.order(order.id), order);
      void qc.invalidateQueries({ queryKey: shopKeys.cart() });
      void qc.invalidateQueries({ queryKey: shopKeys.checkout() });
      void qc.invalidateQueries({ queryKey: shopKeys.orders() });
      void qc.invalidateQueries({ queryKey: shopKeys.checklistAll() });
    },
  });
}

/** GET /shop/orders — the 50 newest. */
export const useShopOrders = () =>
  useQuery({
    queryKey: shopKeys.orders(),
    queryFn: async () => shopOrdersSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/shop/orders')).data.data).orders,
    enabled: isAuthenticated(),
    staleTime: 15_000,
    retry: noRetry,
  });

/** GET /shop/orders/:id. */
export const useShopOrder = (id: number) =>
  useQuery({
    queryKey: shopKeys.order(id),
    queryFn: async () =>
      shopOrderEnvelopeSchema.parse((await apiClient.get<ApiEnvelope<unknown>>(`/shop/orders/${id}`)).data.data).order,
    enabled: isAuthenticated() && id > 0,
    staleTime: 15_000,
    retry: noRetry,
  });

/** POST /shop/orders/:id/cancel — while placed and every shipment awaits the seller. */
export function useCancelOrder(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () =>
      shopOrderEnvelopeSchema.parse((await apiClient.post<ApiEnvelope<unknown>>(`/shop/orders/${id}/cancel`)).data.data).order,
    onSuccess: (order) => {
      qc.setQueryData(shopKeys.order(id), order);
      void qc.invalidateQueries({ queryKey: shopKeys.orders() });
      void qc.invalidateQueries({ queryKey: shopKeys.checklistAll() });
    },
  });
}

/** POST /shop/orders/:id/returns — an item of a delivered shipment, inside the seller's return window. */
export function useRequestReturn(orderId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (v: { orderItemId: number; quantity: number; reason: ReturnReason; note?: string }) =>
      shopReturnEnvelopeSchema.parse(
        (
          await apiClient.post<ApiEnvelope<unknown>>(`/shop/orders/${orderId}/returns`, {
            order_item_id: v.orderItemId,
            quantity: v.quantity,
            reason: v.reason,
            note: v.note || null,
          })
        ).data.data,
      ).return,
    onSuccess: () => qc.invalidateQueries({ queryKey: shopKeys.order(orderId) }),
  });
}

/**
 * POST /shop/orders/:id/items/:itemId/review — CB-SHOP-12 F-5: her review of a delivered item (rating 1–5, optional
 * text), one per order item; it stays `pending` until the admin publishes it.
 */
export function useCreateReview(orderId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (v: { orderItemId: number; rating: number; body?: string }) =>
      shopReviewEnvelopeSchema.parse(
        (
          await apiClient.post<ApiEnvelope<unknown>>(`/shop/orders/${orderId}/items/${v.orderItemId}/review`, {
            rating: v.rating,
            body: v.body || null,
          })
        ).data.data,
      ).review,
    onSuccess: () => qc.invalidateQueries({ queryKey: shopKeys.order(orderId) }),
  });
}
