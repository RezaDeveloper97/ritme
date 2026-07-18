'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { phaseContentSchema } from './schema';

/**
 * Query-key factory for phase-details content (§8). Keyed by phase AND locale
 * because the API returns already-localized copy — the two locales are distinct
 * cache entries.
 */
export const phaseContentKeys = {
  all: ['phase-content'] as const,
  detail: (phase: string, locale: string) =>
    [...phaseContentKeys.all, phase, locale] as const,
};

type PhaseContentResult = z.infer<typeof phaseContentSchema>;

/**
 * GET /cycle/phase-content/{phase} — the nine educational sections for a
 * sub-phase. `locale` is passed as a query param (browsers drop a JS-set
 * Accept-Language header) so the backend answers in the active app locale.
 */
export async function fetchPhaseContent(
  phase: string,
  locale: string,
): Promise<PhaseContentResult> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(
    `/cycle/phase-content/${phase}`,
    { params: { locale } },
  );
  return phaseContentSchema.parse(data.data);
}

/**
 * Reads the current phase's educational content (§8 — server state in TanStack
 * Query). `phase` comes from cycle_view.subphase and may be null when the engine
 * has no confident phase yet; the query stays disabled until it's present so the
 * screen shows a safe fallback instead of firing a bad request. `retry: false`
 * so a 404 (no content for this phase) surfaces immediately as a fallback.
 */
export function usePhaseContent(phase: string | null, locale: string) {
  return useQuery({
    queryKey: phaseContentKeys.detail(phase ?? '', locale),
    queryFn: () => fetchPhaseContent(phase as string, locale),
    enabled: Boolean(phase) && isAuthenticated(),
    staleTime: 30 * 60_000,
    retry: false,
  });
}
