'use client';

import { keepPreviousData, useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { SEARCH_MAX_LENGTH, SEARCH_MIN_LENGTH, type SearchResults, type SearchScope } from '../model/types';
import { searchKeys } from './keys';
import { searchResultsSchema } from './schema';

/*
 * `GET /api/v1/search` (CB-NAV-01, Go only). The query is health data
 * (CLAUDE.md §11): it travels only in this request — never logged, never put
 * in the page URL — and answers are kept a short while in memory only.
 */

/** The query as the API will see it, or `null` when it is too short to send. */
export function searchableQuery(raw: string): string | null {
  const q = raw.trim().slice(0, SEARCH_MAX_LENGTH);
  return [...q].length >= SEARCH_MIN_LENGTH ? q : null;
}

export async function fetchSearch(query: string, scope: SearchScope): Promise<SearchResults> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/search', { params: { q: query, scope } });
  return searchResultsSchema.parse(data.data);
}

/**
 * Results for an already-debounced query. Disabled below the minimum length;
 * keeps the previous answer on screen while the next one loads so the list
 * doesn't flash empty between keystrokes.
 */
export function useSearch(query: string, scope: SearchScope, locale: string) {
  const q = searchableQuery(query);
  return useQuery({
    queryKey: searchKeys.results(q ?? '', scope, locale),
    queryFn: () => fetchSearch(q ?? '', scope),
    enabled: q !== null && isAuthenticated(),
    placeholderData: keepPreviousData,
    staleTime: 30_000,
    gcTime: 60_000,
    retry: 1,
  });
}
