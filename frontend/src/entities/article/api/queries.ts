'use client';

import { keepPreviousData, useInfiniteQuery, useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  ArticleFilters,
  ArticlePage,
  ArticleWithRelated,
  CycleArticles,
} from '../model/types';
import {
  articleDetailSchema,
  articlePageSchema,
  cycleArticlesSectionSchema,
} from './schema';

/** How many articles the library asks for per page. */
const PAGE_SIZE = 12;

/** Query-key factory for articles (CLAUDE.md §8). */
export const articleKeys = {
  all: ['article'] as const,
  cycleSection: () => [...articleKeys.all, 'cycle-section'] as const,
  list: (filters: ArticleFilters) => [...articleKeys.all, 'list', filters] as const,
  detail: (slug: string) => [...articleKeys.all, 'detail', slug] as const,
};

/**
 * GET /home/sections/articles — the published articles that match the phase
 * the user is in today, general ones first-class alongside them (the backend
 * owns that matching).
 *
 * The endpoint answers `section: null` when nothing applies — an empty state,
 * not an error — so this resolves to `null` and the caller renders nothing.
 */
export async function fetchCycleArticles(): Promise<CycleArticles | null> {
  const { data } = await apiClient.get<ApiEnvelope<{ section: unknown }>>(
    '/home/sections/articles',
  );
  const section = data.data?.section ?? null;

  return section === null ? null : cycleArticlesSectionSchema.parse(section);
}

/**
 * Articles for the current cycle phase. Stays disabled until authenticated so
 * it never fires on public screens, and is cached for a few minutes — the set
 * only changes when the phase does (or when an admin publishes).
 */
export function useCycleArticles() {
  return useQuery({
    queryKey: articleKeys.cycleSection(),
    queryFn: fetchCycleArticles,
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}

/** GET /articles — one page of the library, filtered as the reader asked. */
export async function fetchArticlePage(
  filters: ArticleFilters,
  page: number,
): Promise<ArticlePage> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/articles', {
    params: {
      page,
      per_page: PAGE_SIZE,
      // Omit an inactive filter entirely rather than sending an empty value.
      ...(filters.category ? { category: filters.category } : {}),
      ...(filters.q.trim() ? { q: filters.q.trim() } : {}),
    },
  });

  return articlePageSchema.parse(data.data ?? {});
}

/**
 * The article library, page by page — the list screen appends with a "load
 * more" button rather than paging, which keeps the reader's place on a phone.
 *
 * Each filter combination is its own cache entry, so returning to a previous
 * category shows instantly instead of re-fetching.
 */
export function useArticles(filters: ArticleFilters) {
  return useInfiniteQuery({
    queryKey: articleKeys.list(filters),
    queryFn: ({ pageParam }) => fetchArticlePage(filters, pageParam),
    initialPageParam: 1,
    getNextPageParam: (last) => (last.page < last.lastPage ? last.page + 1 : undefined),
    // Keep the previous filter's results on screen while the next ones load, so
    // switching a category doesn't blank the grid and the chips out from under
    // the reader's finger.
    placeholderData: keepPreviousData,
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}

/** GET /articles/{slug} — one article with its body and related reads. */
export async function fetchArticle(slug: string): Promise<ArticleWithRelated> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(
    `/articles/${encodeURIComponent(slug)}`,
  );

  return articleDetailSchema.parse(data.data ?? {});
}

/**
 * A single article. Published content changes rarely, so it stays fresh for a
 * few minutes; an unknown slug is a 404 the screen renders as "not found"
 * rather than a retry loop.
 */
export function useArticle(slug: string) {
  return useQuery({
    queryKey: articleKeys.detail(slug),
    queryFn: () => fetchArticle(slug),
    enabled: isAuthenticated() && slug !== '',
    staleTime: 5 * 60_000,
    retry: false,
  });
}
