'use client';

import { useInfiniteQuery, useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { AiCredits, AiUsagePage, CreditPackCatalog, CreditPackQuote } from '../model/types';
import { creditsSchema, packCatalogSchema, packQuoteSchema, usageSchema } from './schema';

/** Query-key factory for AI credits (CLAUDE.md §8). */
export const aiCreditKeys = {
  all: ['ai-credit'] as const,
  balance: () => [...aiCreditKeys.all, 'balance'] as const,
  usage: () => [...aiCreditKeys.all, 'usage'] as const,
  packs: () => [...aiCreditKeys.all, 'packs'] as const,
  packQuote: (packId: number) => [...aiCreditKeys.all, 'pack-quote', packId] as const,
};

/** GET /ai/credits — the balance, the period and what each feature costs. */
export async function fetchAiCredits(): Promise<AiCredits> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/ai/credits');
  return creditsSchema.parse(data.data);
}

/** GET /ai/credits/usage — last 30 days; `before` pages the history. */
export async function fetchAiUsage(before: number | null): Promise<AiUsagePage> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/ai/credits/usage', {
    params: { before },
  });
  return usageSchema.parse(data.data);
}

/** GET /plus/credit-packs (public). */
export async function fetchCreditPacks(): Promise<CreditPackCatalog> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/plus/credit-packs');
  return packCatalogSchema.parse(data.data);
}

/** POST /plus/credit-packs/checkout `{preview: true}` — prices a pack without creating anything. */
export async function fetchCreditPackQuote(packId: number): Promise<CreditPackQuote> {
  const { data } = await apiClient.post<ApiEnvelope<unknown>>('/plus/credit-packs/checkout', {
    pack_id: packId,
    preview: true,
  });
  return packQuoteSchema.parse(data.data);
}

export function useAiCredits() {
  return useQuery({
    queryKey: aiCreditKeys.balance(),
    queryFn: fetchAiCredits,
    enabled: isAuthenticated(),
    staleTime: 30_000,
  });
}

export function useAiUsage() {
  return useInfiniteQuery({
    queryKey: aiCreditKeys.usage(),
    queryFn: ({ pageParam }) => fetchAiUsage(pageParam),
    initialPageParam: null as number | null,
    getNextPageParam: (last) => last.history.next,
    enabled: isAuthenticated(),
    staleTime: 30_000,
  });
}

export function useCreditPacks() {
  return useQuery({ queryKey: aiCreditKeys.packs(), queryFn: fetchCreditPacks, staleTime: 10 * 60_000 });
}

/** The checkout price of a pack (null = not chosen yet). */
export function useCreditPackQuote(packId: number | null) {
  return useQuery({
    queryKey: aiCreditKeys.packQuote(packId ?? 0),
    queryFn: () => fetchCreditPackQuote(packId ?? 0),
    enabled: packId !== null && isAuthenticated(),
    staleTime: 60_000,
    retry: false,
  });
}
