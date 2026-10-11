'use client';

import clsx from 'clsx';
import { useFormatter, useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  EMPTY_FILTERS,
  filterCount,
  toDirectoryParams,
  useCityChoice,
  useCityServiceOptions,
  useDirectoryListings,
  useNearMe,
  type DirectoryFilters,
} from '@/entities/city-service';
import { Link, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Card, EmptyState, Icon, ScreenHeader, SecondaryButton, SkyLayer } from '@/shared/ui';
import { DirectoryCard, DirectoryFilterSheet } from '@/widgets/directory-filters';

import type { BrowseQuery } from './CityServicesHome';
import { CsError, CsSkeleton } from './ListSkeleton';

/**
 * CB-DIR-07 — `nbl_Dir_List`: the directory list. Search, a city and category row, the filter sheet (child age,
 * open now, amenities, sort), result cards with verified badge, rating, price-from and next slot, and a map link.
 * «Near me» / distance sort asks for the position only when the user picks it; it stays in memory and is sent as
 * `lat`/`lng` search params, never stored.
 */
export function DirectoryList({ initial, onHome }: { initial: BrowseQuery; onHome: () => void }) {
  const t = useTranslations('cityServices');
  const fmt = useFormatter();
  const locale = useLocale() as Locale;
  const options = useCityServiceOptions();
  const [city, setCity, ready] = useCityChoice();
  const [category, setCategory] = useState<string | null>(initial.category);
  const [q, setQ] = useState(initial.q);
  const [qDraft, setQDraft] = useState(initial.q);
  // The home can hand over the child's age in months; a catalog age pick in the sheet replaces it.
  const [homeAge, setHomeAge] = useState<number | null>(initial.ageMonths);
  const [filters, setFilters] = useState<DirectoryFilters>(EMPTY_FILTERS);
  const [sheet, setSheet] = useState(false);
  const [page, setPage] = useState(1);
  const near = useNearMe();

  const cities = options.data?.cities ?? [];
  const activeCity = city !== null && cities.some((c) => c.id === city) ? city : null;
  const catalog = options.data?.place_age_ranges ?? [];
  const derived = toDirectoryParams(filters, catalog, near.point);
  const list = useDirectoryListings({
    city: activeCity,
    category,
    q,
    ...derived,
    ageMonths: derived.ageMonths ?? homeAge,
    page,
  });

  const apply = (next: DirectoryFilters) => {
    setFilters(next);
    if (next.age) setHomeAge(null);
    if (next.sort === 'distance' && !near.point) near.request();
    setPage(1);
  };
  const nearby = () => {
    if (near.point) {
      near.clear();
      setFilters((f) => (f.sort === 'distance' ? { ...f, sort: 'rank' } : f));
    } else {
      near.request();
      setFilters((f) => ({ ...f, sort: 'distance' }));
    }
    setPage(1);
  };

  const count = filterCount(filters) + (homeAge !== null && !filters.age ? 1 : 0);
  const cityName = cities.find((c) => c.id === activeCity)?.name ?? null;
  const chip = (key: string, label: string, on: boolean, onClick: () => void) => (
    <button key={key} type="button" role="radio" aria-checked={on} className={clsx('chip csv-chip', on && 'on')} onClick={onClick}>
      {label}
    </button>
  );

  let body;
  if (options.isPending || list.isPending || !ready) {
    body = <CsSkeleton label={t('loading')} />;
  } else if (options.isError || list.isError) {
    body = (
      <CsError
        title={t('loadError')}
        retry={t('retry')}
        busy={options.isFetching || list.isFetching}
        onRetry={() => {
          void options.refetch();
          void list.refetch();
        }}
      />
    );
  } else if (list.data.listings.length === 0) {
    body = (
      <Card>
        <EmptyState
          icon="home"
          title={t('list.emptyTitle')}
          body={t('list.emptyBody')}
          action={
            count > 0 ? (
              <SecondaryButton
                onClick={() => {
                  setFilters(EMPTY_FILTERS);
                  setHomeAge(null);
                }}
              >
                {t('filters.clear')}
              </SecondaryButton>
            ) : undefined
          }
        />
      </Card>
    );
  } else {
    body = (
      <>
        <p className="dfl-count" aria-live="polite">
          {t('list.results', { count: formatNumber(list.data.meta.total, locale) })}
          {' · '}
          {t(`list.sortedBy.${derived.sort ?? 'rank'}`)}
        </p>
        <ul className="csv-list">
          {list.data.listings.map((l) => (
            <li key={l.id}>
              <DirectoryCard place={l} catalog={catalog} />
            </li>
          ))}
        </ul>
        {list.data.meta.last_page > 1 ? (
          <nav className="csv-pager" aria-label={t('list.pages')}>
            <SecondaryButton variant="text" disabled={page <= 1} onClick={() => setPage((n) => n - 1)}>
              {t('list.prev')}
            </SecondaryButton>
            <span>{t('list.page', { page: fmt.number(page), last: fmt.number(list.data.meta.last_page) })}</span>
            <SecondaryButton variant="text" disabled={page >= list.data.meta.last_page} onClick={() => setPage((n) => n + 1)}>
              {t('list.next')}
            </SecondaryButton>
          </nav>
        ) : null}
      </>
    );
  }

  return (
    <div className="view csv-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={cityName ? t('list.titleIn', { city: cityName }) : t('title')}
          subtitle={t('tag')}
          onBack={onHome}
          backLabel={t('back')}
        />
        <div className="csv-body">
          <form
            className="cdh-search"
            role="search"
            onSubmit={(e) => {
              e.preventDefault();
              setQ(qDraft);
              setPage(1);
            }}
          >
            <Icon name="search" size={18} />
            <input
              type="search"
              value={qDraft}
              maxLength={80}
              placeholder={t('home.searchPlaceholder')}
              aria-label={t('home.searchLabel')}
              onChange={(e) => setQDraft(e.target.value)}
            />
          </form>
          <div className="csv-chips" aria-label={t('filters.title')}>
            <button type="button" className={clsx('chip csv-chip dfl-filter', count > 0 && 'on')} onClick={() => setSheet(true)}>
              <Icon name="filterLines"size={16} />
              {t('filters.title')}
              {count > 0 ? <span className="dfl-badge">{formatNumber(count, locale)}</span> : null}
            </button>
            <button type="button" aria-pressed={near.point !== null} className={clsx('chip csv-chip', near.point && 'on')} onClick={nearby}>
              <Icon name="mapPin" size={16} />
              {t('list.nearMe')}
            </button>
            <button
              type="button"
              aria-pressed={filters.openNow}
              className={clsx('chip csv-chip', filters.openNow && 'on')}
              onClick={() => apply({ ...filters, openNow: !filters.openNow })}
            >
              {t('home.openNow')}
            </button>
            {homeAge !== null && !filters.age ? (
              <button type="button" className="chip csv-chip on" aria-label={t('filters.removeAge')} onClick={() => setHomeAge(null)}>
                {t('list.ageChip', { months: formatNumber(homeAge, locale) })}
                <Icon name="x"size={14} />
              </button>
            ) : null}
          </div>
          {near.status === 'denied' ? <p className="dfl-note">{t('list.locationDenied')}</p> : null}
          {options.data ? (
            <>
              {cities.length > 0 ? (
                <div role="radiogroup" aria-label={t('list.cityLabel')} className="csv-chips">
                  {chip('all', t('list.allCities'), activeCity === null, () => {
                    setCity(null);
                    setPage(1);
                  })}
                  {cities.map((c) =>
                    chip(String(c.id), c.name, activeCity === c.id, () => {
                      setCity(c.id);
                      setPage(1);
                    }),
                  )}
                </div>
              ) : null}
              <div role="radiogroup" aria-label={t('list.categoryLabel')} className="csv-chips">
                {chip('all', t('list.allCategories'), category === null, () => {
                  setCategory(null);
                  setPage(1);
                })}
                {options.data.categories.map((c) =>
                  chip(c.slug, c.name, category === c.slug, () => {
                    setCategory(c.slug);
                    setPage(1);
                  }),
                )}
              </div>
            </>
          ) : null}
          {body}
          <Link href="/profile/bookings" className="csv-link">
            <Icon name="calendar" size={18} />
            {t('list.bookings')}
          </Link>
          <p className="csv-note">{t('note')}</p>
        </div>
        <div className="page-tail" />
      </div>
      <Link href="/city-services/map" className="dfl-map-fab">
        <Icon name="mapPin" size={18} />
        {t('list.mapFab')}
      </Link>
      <DirectoryFilterSheet
        open={sheet}
        onClose={() => setSheet(false)}
        value={filters}
        onApply={apply}
        ageRanges={catalog}
        amenities={options.data?.amenities ?? []}
        sorts={options.data?.sorts ?? []}
        nearReady={near.point !== null}
      />
    </div>
  );
}
