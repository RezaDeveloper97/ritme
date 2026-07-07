'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { Banner, BannerPosition, BannersByPosition } from '../model/types';
import { bannersEnvelopeSchema } from './schema';

/**
 * Query-key factory for banners (CLAUDE.md §8). One cache entry holds every
 * slot, so the several home-page slideshows share a single request.
 */
export const bannerKeys = {
  all: ['banner'] as const,
  list: () => [...bannerKeys.all, 'list'] as const,
};

/** GET /banners — active banners for every home slot, grouped by position. */
export async function fetchBanners(): Promise<BannersByPosition> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/banners');
  return bannersEnvelopeSchema.parse(data.data);
}

/**
 * Active banners for a single slot. Shares one cached request across all slots
 * via `select`, and stays disabled until authenticated so it never fires on
 * public screens. Returns `[]` while loading or when the slot is empty.
 */
export function useBanners(position: BannerPosition): Banner[] {
  const { data } = useQuery({
    queryKey: bannerKeys.list(),
    queryFn: fetchBanners,
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
    select: (all) => all[position],
  });

  return data ?? [];
}
