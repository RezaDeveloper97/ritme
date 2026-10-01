'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { PlusCatalog, PlusInvoice, PlusQuote, PlusStatus } from '../model/types';
import { catalogSchema, historySchema, quoteEnvelopeSchema, statusSchema } from './schema';

/** Query-key factory for Ritme Plus (CLAUDE.md §8). */
export const plusKeys = {
  all: ['plus'] as const,
  plans: () => [...plusKeys.all, 'plans'] as const,
  status: () => [...plusKeys.all, 'status'] as const,
  history: () => [...plusKeys.all, 'history'] as const,
  quote: (planId: number, code: string | null) => [...plusKeys.all, 'quote', planId, code ?? ''] as const,
};

/** GET /plus/plans (public) — active plans, VAT rate and trial length. */
export async function fetchPlusPlans(): Promise<PlusCatalog> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/plus/plans');
  return catalogSchema.parse(data.data);
}

/** GET /plus/status — tier, subscription, trial and this month's entitlements. */
export async function fetchPlusStatus(): Promise<PlusStatus> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/plus/status');
  return statusSchema.parse(data.data);
}

/** GET /plus/history — the latest invoices, newest first. */
export async function fetchPlusHistory(): Promise<PlusInvoice[]> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/plus/history');
  return historySchema.parse(data.data ?? []);
}

/**
 * POST /plus/checkout `{preview: true}` — prices a plan (and an optional
 * discount code) without creating anything. Side-effect free, so it is read
 * through a query; a rejected code is a 422 with `errors.discount_code`.
 */
export async function fetchPlusQuote(planId: number, code: string | null): Promise<PlusQuote> {
  const { data } = await apiClient.post<ApiEnvelope<unknown>>('/plus/checkout', {
    plan_id: planId,
    preview: true,
    ...(code ? { discount_code: code } : {}),
  });
  return quoteEnvelopeSchema.parse(data.data);
}

export function usePlusPlans() {
  return useQuery({ queryKey: plusKeys.plans(), queryFn: fetchPlusPlans, staleTime: 10 * 60_000 });
}

export function usePlusStatus() {
  return useQuery({
    queryKey: plusKeys.status(),
    queryFn: fetchPlusStatus,
    enabled: isAuthenticated(),
    staleTime: 60_000,
  });
}

export function usePlusHistory(enabled = true) {
  return useQuery({
    queryKey: plusKeys.history(),
    queryFn: fetchPlusHistory,
    enabled: enabled && isAuthenticated(),
    staleTime: 60_000,
  });
}

/** The checkout price of `planId` (null = not chosen yet), with `code` applied when given. */
export function usePlusQuote(planId: number | null, code: string | null = null) {
  return useQuery({
    queryKey: plusKeys.quote(planId ?? 0, code),
    queryFn: () => fetchPlusQuote(planId ?? 0, code),
    enabled: planId !== null && isAuthenticated(),
    staleTime: 60_000,
    retry: false,
  });
}
