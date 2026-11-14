'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api } from '@/shared/api';

/** A user's AI credits (GET /users/:id/ai-credits) and the manual gift / deduction (admin-api.md §29). */

const balanceSchema = z.object({
  active: z.boolean(),
  period: z.string().nullable(),
  period_ends_at: z.string().nullable(),
  plan_total: z.number(),
  plan_remaining: z.number(),
  extra: z.number(),
  extra_locked: z.boolean(),
  spendable: z.number(),
});

const grantSchema = z.object({
  id: z.number(),
  kind: z.string(),
  amount: z.number(),
  remaining: z.number(),
  current: z.boolean(),
  valid_until: z.string().nullable(),
  note: z.string().nullable(),
  created_at: z.string().nullable(),
});
export type CreditGrant = z.infer<typeof grantSchema>;

const spendSchema = z.object({
  id: z.number(),
  feature: z.string(),
  credits: z.number(),
  plan_credits: z.number(),
  extra_credits: z.number(),
  status: z.string(),
  created_at: z.string().nullable(),
});
export type CreditSpend = z.infer<typeof spendSchema>;

const adjustmentSchema = z.object({
  id: z.number(),
  admin: z.object({ id: z.number().nullable(), name: z.string().nullable() }),
  note: z.string().nullable(),
  details: z
    .object({ delta: z.number().optional(), granted: z.number().optional(), taken: z.number().optional() })
    .passthrough()
    .nullable(),
  created_at: z.string().nullable(),
});
export type CreditAdjustment = z.infer<typeof adjustmentSchema>;

export const userCreditsSchema = z.object({
  user_id: z.number(),
  balance: balanceSchema,
  grants: z.array(grantSchema),
  spends: z.array(spendSchema),
  adjustments: z.array(adjustmentSchema),
  limits: z.object({ max_delta: z.number(), max_note: z.number() }),
});
export type UserCredits = z.infer<typeof userCreditsSchema>;

export const userCreditsKeys = { user: (id: number) => ['user-ai-credits', id] as const };

export function useUserCredits(userId: number, enabled = true) {
  return useQuery({
    queryKey: userCreditsKeys.user(userId),
    queryFn: ({ signal }) => api.get(`/users/${userId}/ai-credits`, { schema: userCreditsSchema, signal }),
    enabled,
  });
}

const adjustResultSchema = z.object({
  delta: z.number(),
  granted: z.number(),
  taken: z.number(),
  grant_id: z.number().nullable(),
  credits: userCreditsSchema,
});

export function useAdjustCredits(userId: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: { delta: number; note: string }) =>
      api.post(`/users/${userId}/ai-credits/adjust`, body, { schema: adjustResultSchema }),
    onSuccess: (res) => client.setQueryData(userCreditsKeys.user(userId), res.credits),
  });
}
