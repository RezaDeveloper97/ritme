import type { SearchScope } from '../model/types';

/**
 * Query-key factory for `/search` (CLAUDE.md §8). Answers are localized
 * server-side, so the locale is part of the key.
 */
export const searchKeys = {
  all: ['search'] as const,
  results: (query: string, scope: SearchScope, locale: string) =>
    [...searchKeys.all, 'results', locale, scope, query] as const,
};
