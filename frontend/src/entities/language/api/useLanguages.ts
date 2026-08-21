'use client';

import { useQuery } from '@tanstack/react-query';

import type { Language } from '../model/types';

import { fetchLanguages, FALLBACK_LANGUAGES } from './languagesApi';

/**
 * The languages the app currently offers.
 *
 * Public and rarely-changing, so it is cached aggressively and never refetched
 * on focus — an admin adding a language is not something the user needs to see
 * mid-session. The bundled locales stand in until the request resolves, so the
 * language picker is never empty.
 */
export function useLanguages() {
  return useQuery<Language[]>({
    queryKey: ['languages'],
    queryFn: fetchLanguages,
    staleTime: 30 * 60_000,
    gcTime: 60 * 60_000,
    refetchOnWindowFocus: false,
    placeholderData: FALLBACK_LANGUAGES,
  });
}
