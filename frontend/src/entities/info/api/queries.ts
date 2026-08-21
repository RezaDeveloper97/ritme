'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';

import type { InfoGroup, InfoSection } from '../model/types';
import { infoSectionsSchema } from './schema';

/**
 * Query-key factory for the text screens (§8). Keyed by group and locale
 * because the API returns already-localized copy — each pair is its own cache
 * entry.
 */
export const infoKeys = {
  all: ['info'] as const,
  sections: (group: InfoGroup, locale: string) =>
    [...infoKeys.all, group, locale] as const,
};

/**
 * GET /info/{group} — the active boxes of one screen in display order.
 * `locale` travels as a query param because browsers drop a JS-set
 * Accept-Language header.
 */
export async function fetchInfoSections(
  group: InfoGroup,
  locale: string,
): Promise<InfoSection[]> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/info/${group}`, {
    params: { locale },
  });
  return infoSectionsSchema.parse(data.data);
}

/**
 * One text screen as maintained in the admin panel. Public content, so the
 * query runs whether or not anyone is signed in, and it is cached hard — the
 * text changes on an admin's schedule, not the reader's.
 */
export function useInfoSections(group: InfoGroup, locale: string) {
  return useQuery({
    queryKey: infoKeys.sections(group, locale),
    queryFn: () => fetchInfoSections(group, locale),
    staleTime: 60 * 60_000,
    retry: false,
  });
}
