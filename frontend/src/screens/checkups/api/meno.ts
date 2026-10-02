'use client';

import { useQuery } from '@tanstack/react-query';

import { menopauseKeys } from '@/entities/menopause';
import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { type MenoCheckupGroup, menoCheckupGroupsSchema, menoCheckupsIntroSchema } from '../model/meno';

/*
 * Catalog reads of the menopause checkups view (CB-MENO-09): admin-editable
 * content localized by `Accept-Language` (hence the locale in the keys), not
 * health data. Only mounted in menopause mode.
 */

const catalogKey = (group: string, locale: string) => [...menopauseKeys.all, 'catalog', group, locale] as const;

async function fetchCatalog(group: string): Promise<unknown> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/catalog/${group}`, {
    params: { audience: 'menopause' },
  });
  return data.data;
}

/** GET /catalog/meno_checkup_groups — the board's sections. */
export function useMenoCheckupGroups(locale: string) {
  return useQuery({
    queryKey: catalogKey('meno_checkup_groups', locale),
    queryFn: async (): Promise<MenoCheckupGroup[]> =>
      menoCheckupGroupsSchema.parse(await fetchCatalog('meno_checkup_groups')),
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: 1,
  });
}

/** GET /catalog/meno_tips → the `checkups_intro` note (null when absent). */
export function useMenoCheckupsIntro(locale: string) {
  return useQuery({
    queryKey: catalogKey('meno_tips', locale),
    queryFn: async (): Promise<string | null> => menoCheckupsIntroSchema.parse(await fetchCatalog('meno_tips')),
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: 1,
  });
}
