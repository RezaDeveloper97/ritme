'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useMemo, useState } from 'react';

import { ArticleCard, useArticles } from '@/entities/article';
import { useRouter } from '@/shared/i18n';
import { useMounted } from '@/shared/lib/use-mounted';
import { Icon, NavBack } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

/** Keystrokes settle before a page is requested — one call per pause, not per key. */
const SEARCH_DEBOUNCE_MS = 350;

/**
 * The article library: everything an admin has published, filterable by
 * category, searchable by title/excerpt, and appended page by page.
 *
 * Deliberately phase-blind — the home row already surfaces what matches the
 * user's current phase, and a browsable list keyed by it would put health data
 * into a shareable URL (CLAUDE.md §11).
 */
export function ArticlesPage() {
  const t = useTranslations('articles');
  const router = useRouter();
  const mounted = useMounted();

  const [category, setCategory] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');

  useEffect(() => {
    const id = setTimeout(() => setDebouncedSearch(search), SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(id);
  }, [search]);

  const filters = useMemo(
    () => ({ category, q: debouncedSearch }),
    [category, debouncedSearch],
  );

  const query = useArticles(filters);
  const pages = query.data?.pages ?? [];
  const articles = pages.flatMap((page) => page.articles);
  const total = pages[0]?.total ?? 0;
  // The facet is every published category, not just the ones left after
  // filtering, so the chips stay put as the reader moves between them.
  const categories = pages[0]?.categories ?? [];

  const filtering = category !== null || debouncedSearch.trim() !== '';
  const loading = !mounted || query.isPending;

  const resetFilters = () => {
    setCategory(null);
    setSearch('');
    setDebouncedSearch('');
  };

  return (
    <div className="view ar-page">
      <div className="home-grad ar-grad" />

      <div className="hdr ar-hdr">
        <NavBack onClick={() => router.back()} label={t('back')} />
        <span className="ar-hdr-title">{t('title')}</span>
        {/* Balances the back button so the title sits truly centered. */}
        <span className="ar-hdr-spacer" aria-hidden />
      </div>

      <div className="scroll ar-scroll">
        <p className="ar-subtitle">{t('subtitle')}</p>

        <div className="field ar-search">
          <Icon name="search" size={18} stroke="var(--muted-2)" />
          <input
            type="search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder={t('searchPlaceholder')}
            aria-label={t('searchPlaceholder')}
          />
          {search !== '' && (
            <button
              type="button"
              className="ar-search-clear"
              onClick={() => setSearch('')}
              aria-label={t('clearSearch')}
            >
              <Icon name="x" size={16} />
            </button>
          )}
        </div>

        {categories.length > 0 && (
          <div className="scroll-x ar-chips-wrap">
            <div className="ar-chips">
              <FilterChip
                label={t('allCategories')}
                active={category === null}
                onClick={() => setCategory(null)}
              />
              {categories.map((name) => (
                <FilterChip
                  key={name}
                  label={name}
                  active={category === name}
                  onClick={() => setCategory(category === name ? null : name)}
                />
              ))}
            </div>
          </div>
        )}

        {loading ? (
          <ArticlesSkeleton />
        ) : query.isError ? (
          <EmptyState
            title={t('error.title')}
            text={t('error.text')}
            actionLabel={t('error.retry')}
            onAction={() => void query.refetch()}
          />
        ) : articles.length === 0 ? (
          filtering ? (
            <EmptyState
              title={t('noResults.title')}
              text={t('noResults.text')}
              actionLabel={t('noResults.reset')}
              onAction={resetFilters}
            />
          ) : (
            <EmptyState title={t('empty.title')} text={t('empty.text')} />
          )
        ) : (
          <>
            <div className="ar-count">{t('count', { n: total })}</div>

            {/* aria-busy while a filter change is still in flight: the grid below
                is the previous result set, kept on purpose (keepPreviousData). */}
            <div className="ar-grid" aria-busy={query.isFetching && !query.isFetchingNextPage}>
              {articles.map((article) => (
                <ArticleCard
                  key={article.id}
                  article={article}
                  href={`/articles/${article.slug}`}
                  readTimeLabel={
                    article.readTimeMinutes === null
                      ? null
                      : t('min', { n: article.readTimeMinutes })
                  }
                />
              ))}
            </div>

            {query.hasNextPage && (
              <button
                type="button"
                className="btn btn-ghost ar-more"
                onClick={() => void query.fetchNextPage()}
                disabled={query.isFetchingNextPage}
              >
                {query.isFetchingNextPage ? t('loading') : t('loadMore')}
              </button>
            )}
          </>
        )}

        <div className="page-tail" />
      </div>

      <BottomNav />
    </div>
  );
}

function FilterChip({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      className={active ? 'ar-chip is-active' : 'ar-chip'}
      onClick={onClick}
      aria-pressed={active}
    >
      {label}
    </button>
  );
}

function EmptyState({
  title,
  text,
  actionLabel,
  onAction,
}: {
  title: string;
  text: string;
  actionLabel?: string;
  onAction?: () => void;
}) {
  return (
    <div className="ar-empty">
      <span className="ar-empty-icon">
        <Icon name="bookOpen" size={28} stroke="currentColor" />
      </span>
      <h2 className="ar-empty-title">{title}</h2>
      <p className="ar-empty-text">{text}</p>
      {actionLabel && onAction && (
        <button type="button" className="btn btn-ghost ar-empty-cta" onClick={onAction}>
          {actionLabel}
        </button>
      )}
    </div>
  );
}

/** Placeholder cards with the loaded grid's rhythm. */
function ArticlesSkeleton() {
  return (
    <div aria-hidden className="ar-grid">
      {[0, 1, 2, 3].map((i) => (
        <div key={i} className="ar-skel-card">
          <span className="skeleton-line ar-skel-cover" />
          <span className="skeleton-line ar-skel-line" />
          <span className="skeleton-line ar-skel-line is-short" />
        </div>
      ))}
    </div>
  );
}
