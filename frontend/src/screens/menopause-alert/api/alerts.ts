'use client';

import { useQuery } from '@tanstack/react-query';

import { menopauseKeys } from '@/entities/menopause';
import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { type MenoAlert, menoAlertsSchema } from '../model/alerts';

/**
 * GET /catalog/meno_alerts?audience=menopause — admin-editable clinical copy,
 * localized by `Accept-Language` (hence the locale in the key). Content, not
 * health data; cached for the session.
 */
export async function fetchMenoAlerts(): Promise<MenoAlert[]> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/catalog/meno_alerts', {
    params: { audience: 'menopause' },
  });
  return menoAlertsSchema.parse(data.data);
}

export function useMenoAlerts(locale: string) {
  return useQuery({
    queryKey: [...menopauseKeys.all, 'catalog', 'meno_alerts', locale] as const,
    queryFn: fetchMenoAlerts,
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: 1,
  });
}
