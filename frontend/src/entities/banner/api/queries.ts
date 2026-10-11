'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { Ad, AdSlot, Banner, BannerPosition, BannersByPosition } from '../model/types';
import { adsEnvelopeSchema, bannersEnvelopeSchema } from './schema';

/**
 * Query-key factory for banners (CLAUDE.md §8). One cache entry holds every
 * slot, so the several home-page slideshows share a single request.
 */
export const bannerKeys = {
  all: ['banner'] as const,
  list: () => [...bannerKeys.all, 'list'] as const,
  ads: (mode: string | null) => [...bannerKeys.all, 'ads', mode] as const,
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

/**
 * Whether the banner request has settled (loaded or failed), without caring
 * what came back. Same cache entry as {@link useBanners}, so it adds no request.
 * A screen uses it to hold content that sits *below* a banner slot until the
 * slot's height is known — a banner popping in later would push that content
 * down (a layout shift, perf baseline §3 #5).
 */
export function useBannersSettled(): boolean {
  const { isPending, fetchStatus } = useQuery({
    queryKey: bannerKeys.list(),
    queryFn: fetchBanners,
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
    select: () => true,
  });
  // A disabled (signed out) or offline-paused query never settles on its
  // own; only a request actually in flight counts as unsettled.
  return !isPending || fetchStatus !== 'fetching';
}

/** GET /banners/ads?mode= — sponsored slots for the current app mode (targeting is by mode only). */
export async function fetchAds(mode: string | null): Promise<Record<AdSlot, Ad[]>> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/banners/ads', {
    params: mode ? { mode } : undefined,
  });
  return adsEnvelopeSchema.parse(data.data);
}

/** Sponsored items of one slot; `[]` while loading, on error, for teen mode or when nothing is booked. */
export function useAds(slot: AdSlot, mode: string | null): Ad[] {
  const { data } = useQuery({
    queryKey: bannerKeys.ads(mode),
    queryFn: () => fetchAds(mode),
    enabled: isAuthenticated() && mode !== 'teen',
    staleTime: 5 * 60_000,
    retry: false,
    select: (all) => all[slot],
  });
  return data ?? [];
}

/**
 * POST /banners/:id/events — a per-day aggregate counter. Fire and forget: no user id, no health context, never
 * retried and never surfaced; a failure (rate limit, offline) just drops the count.
 */
export function sendAdEvent(id: number, event: 'impression' | 'click'): void {
  apiClient.post(`/banners/${id}/events`, { event }).catch(() => undefined);
}
