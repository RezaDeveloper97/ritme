'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource, pagedSchema, translationsSchema } from '@/shared/api';

/** Ritme Plus admin (admin-api.md §15). Money is integer rials; the UI shows toman (lib/money). */

const userSchema = z.object({ id: z.number(), name: z.string().nullable(), mobile: z.string().nullable() });
const planRefSchema = z.object({ id: z.number(), code: z.string(), title: translationsSchema }).nullable();

export const planSchema = z.object({
  id: z.number(),
  code: z.string(),
  title: translationsSchema,
  badge: translationsSchema,
  duration_months: z.number(),
  price_rials: z.number(),
  monthly_display_rials: z.number().nullable(),
  monthly_price_rials: z.number(),
  is_highlighted: z.boolean(),
  is_active: z.boolean(),
  sort_order: z.number(),
  invoices_count: z.number(),
  subscriptions_count: z.number(),
  active_subscriptions: z.number(),
});
export type Plan = z.infer<typeof planSchema>;

const plansListSchema = z.object({
  items: z.array(planSchema),
  limits: z.object({ min_price_rials: z.number(), max_price_rials: z.number(), max_duration_months: z.number() }),
  next_sort_order: z.number(),
});

export const plansApi = createResource({
  key: 'plus-plans',
  path: '/plus/plans',
  list: plansListSchema,
  detail: z.object({ plan: planSchema }),
});

export const discountSchema = z.object({
  id: z.number(),
  code: z.string(),
  kind: z.enum(['percent', 'amount']),
  value: z.number(),
  max_redemptions: z.number().nullable(),
  per_user_limit: z.number().nullable(),
  plan_ids: z.array(z.number()).nullable(),
  starts_at: z.string().nullable(),
  expires_at: z.string().nullable(),
  is_active: z.boolean(),
  uses: z.object({ paid: z.number(), pending: z.number() }),
  state: z.string(),
});
export type Discount = z.infer<typeof discountSchema>;

export const discountsApi = createResource({
  key: 'plus-discounts',
  path: '/plus/discount-codes',
  list: pagedSchema(discountSchema, {}),
  detail: z.object({ discount_code: discountSchema }),
});

/** `DELETE` answers whether the row was deleted or only deactivated (it is referenced). */
export const removeResultSchema = z.object({ id: z.number(), deleted: z.boolean(), deactivated: z.boolean() });

export function useRemoveOrDeactivate(path: string, key: readonly unknown[]) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => api.delete(`${path}/${id}`, { schema: removeResultSchema }),
    onSuccess: () => void client.invalidateQueries({ queryKey: key }),
  });
}

export const settingsSchema = z.object({
  trial_offer_percent: z.number(),
  vat_rate_bps: z.number(),
  vat_override_bps: z.number().nullable(),
  vat_env_rate_bps: z.number(),
  vat_source: z.enum(['env', 'admin']),
  trial_days: z.number(),
});
export type PlusSettings = z.infer<typeof settingsSchema>;
const settingsBody = z.object({ settings: settingsSchema });
const settingsKey = ['plus-settings'] as const;

export function useSettings() {
  return useQuery({ queryKey: settingsKey, queryFn: ({ signal }) => api.get('/plus/settings', { schema: settingsBody, signal }) });
}

export function useSaveSettings() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: { trial_offer_percent: number; vat_rate_bps: number | null }) =>
      api.put('/plus/settings', body, { schema: settingsBody }),
    onSuccess: (data) => client.setQueryData(settingsKey, data),
  });
}

export const subscriptionSchema = z.object({
  id: z.number(),
  user: userSchema,
  plan: planRefSchema,
  invoice_reference: z.string().nullable(),
  status: z.string(),
  effective_status: z.string(),
  source: z.string(),
  starts_at: z.string().nullable(),
  ends_at: z.string().nullable(),
  days_left: z.number(),
  auto_renew: z.boolean(),
});
export type Subscription = z.infer<typeof subscriptionSchema>;

const countsSchema = z.object({ active: z.number(), canceled: z.number(), expired: z.number(), refunded: z.number() });

export const subscriptionsApi = createResource({
  key: 'plus-subscriptions',
  path: '/plus/subscriptions',
  list: pagedSchema(subscriptionSchema, { counts: countsSchema }),
  detail: z.unknown(),
});

const receiptSchema = z.object({ ref_id: z.string(), amount_rials: z.number().nullable(), paid_at: z.string().nullable() });

export const paymentRowSchema = z.object({
  id: z.number(),
  reference: z.string(),
  user: userSchema,
  plan: planRefSchema,
  status: z.string(),
  effective_status: z.string(),
  total_rials: z.number(),
  discount_rials: z.number(),
  discount_code: z.string().nullable(),
  gateway: z.string().nullable(),
  receipt: receiptSchema.nullable(),
  created_at: z.string().nullable(),
});
export type PaymentRow = z.infer<typeof paymentRowSchema>;

const actionSchema = z.object({
  id: z.number(),
  action: z.string(),
  admin: z.object({ id: z.number().nullable(), name: z.string().nullable() }),
  amount_rials: z.number().nullable(),
  days: z.number().nullable(),
  gateway: z.string().nullable(),
  gateway_ref: z.string().nullable(),
  note: z.string().nullable(),
  created_at: z.string().nullable(),
});
export type PlusAction = z.infer<typeof actionSchema>;

export const paymentSchema = z.object({
  id: z.number(),
  reference: z.string(),
  user: userSchema,
  plan: planRefSchema,
  duration_months: z.number(),
  status: z.string(),
  subtotal_rials: z.number(),
  discount_rials: z.number(),
  vat_rate_bps: z.number(),
  vat_rials: z.number(),
  total_rials: z.number(),
  discount_code: z.string().nullable(),
  gateway: z.string().nullable(),
  receipt: receiptSchema.extend({ gateway: z.string().nullable() }).nullable(),
  paid_at: z.string().nullable(),
  created_at: z.string().nullable(),
  subscriptions: z.array(
    z.object({ id: z.number(), status: z.string(), source: z.string(), starts_at: z.string().nullable(), ends_at: z.string().nullable() }),
  ),
  refund: z.object({ refundable: z.boolean(), gateway_available: z.boolean(), amount_rials: z.number() }),
  actions: z.array(actionSchema),
});
export type Payment = z.infer<typeof paymentSchema>;

export const paymentsApi = createResource({
  key: 'plus-payments',
  path: '/plus/payments',
  list: pagedSchema(paymentRowSchema, {
    summary: z.object({ paid_count: z.number(), paid_rials: z.number(), refunded_rials: z.number() }),
    gateways: z.array(z.string()),
    statuses: z.array(z.string()),
  }),
  detail: z.object({ payment: paymentSchema }),
  alsoInvalidate: [['plus-subscriptions']],
});
