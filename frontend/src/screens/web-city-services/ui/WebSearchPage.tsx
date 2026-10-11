'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useState } from 'react';

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
import { type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Card, EmptyState, Icon, Skeleton, SkeletonGroup } from '@/shared/ui';
import { isMapEnabled, NeshanMap, type MapBbox, type MapPin } from '@/shared/ui/map';
import { DirectoryCard } from '@/widgets/directory-filters';

import { APP_DIRECTORY } from '../model/place';
import { DirectoryWebShell } from './DirectoryWebShell';

const PER_PAGE = 12;
const SEARCH_DEBOUNCE_MS = 350;
const bboxParam = (b: MapBbox): string => `${b.west},${b.south},${b.east},${b.north}`;

/**
 * CB-DIR-11 — `W_Dir_Search`: hero + search, filter sidebar, result list and a sticky map (shared NeshanMap). Public:
 * no session needed. Without a Neshan key (or if the SDK fails) the map column disappears and the list takes the width.
 * The «near me» position is asked for only when she picks the distance sort and is never stored.
 */
export function WebSearchPage() {
  const t = useTranslations('cityServices');
  const w = useTranslations('webDirectory');
  const locale = useLocale() as Locale;
  const options = useCityServiceOptions({ anon: true });
  const [city, setCity, ready] = useCityChoice();
  const [category, setCategory] = useState<string | null>(null);
  const [qDraft, setQDraft] = useState('');
  const [q, setQ] = useState('');
  const [filters, setFilters] = useState<DirectoryFilters>(EMPTY_FILTERS);
  const [page, setPage] = useState(1);
  const [bbox, setBbox] = useState<string | undefined>(undefined);
  const [selected, setSelected] = useState<string | null>(null);
  const [mapOk, setMapOk] = useState(isMapEnabled());
  const near = useNearMe();

  useEffect(() => {
    const id = window.setTimeout(() => setQ(qDraft), SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(id);
  }, [qDraft]);

  const cities = options.data?.cities ?? [];
  const activeCity = city !== null && cities.some((c) => c.id === city) ? city : null;
  const catalog = options.data?.place_age_ranges ?? [];
  const list = useDirectoryListings(
    { city: activeCity, category, q, ...toDirectoryParams(filters, catalog, near.point), bbox, perPage: PER_PAGE, page },
    ready,
  );
  const places = useMemo(() => list.data?.listings ?? [], [list.data]);
  const meta = list.data?.meta;
  const total = meta?.total ?? places.length;
  const pins = useMemo<MapPin[]>(
    () => places.filter((p) => p.lat !== null && p.lng !== null).map((p) => ({ id: String(p.id), lat: p.lat as number, lng: p.lng as number, label: p.name })),
    [places],
  );
  const pick = places.find((p) => String(p.id) === selected) ?? null;
  const cityName = cities.find((c) => c.id === activeCity)?.name;
  const num = (n: number) => formatNumber(n, locale);

  // Any filter change goes back to page 1.
  const change = <T,>(set: (v: T) => void) => (v: T) => {
    set(v);
    setPage(1);
    setSelected(null);
  };
  const applyFilters = (next: DirectoryFilters) => {
    change(setFilters)(next);
    if (next.sort === 'distance' && !near.point) near.request();
  };
  const toggleAmenity = (code: string) =>
    applyFilters({ ...filters, amenities: filters.amenities.includes(code) ? filters.amenities.filter((a) => a !== code) : [...filters.amenities, code] });
  const nothingActive = filterCount(filters) === 0 && !bbox && !category && !q;
  const sorts = (['rank', 'distance', 'rating'] as const).filter((s) => !options.data?.sorts.length || options.data.sorts.includes(s));

  const chip = (key: string, label: string, on: boolean, onClick: () => void) => (
    <button key={key} type="button" role="radio" aria-checked={on} className={clsx('chip csv-chip', on && 'on')} onClick={onClick}>
      {label}
    </button>
  );

  let results;
  if (options.isPending || list.isPending || !ready) {
    results = (
      <SkeletonGroup label={t('loading')} className="wcs-list">
        <Skeleton shape="block" />
        <Skeleton shape="block" />
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (options.isError || list.isError) {
    results = (
      <Card>
        <EmptyState
          icon="warning"
          title={t('loadError')}
          action={
            <button
              type="button"
              className="wcs-link-btn"
              onClick={() => {
                void options.refetch();
                void list.refetch();
              }}
            >
              {t('retry')}
            </button>
          }
        />
      </Card>
    );
  } else if (places.length === 0) {
    results = (
      <Card>
        <EmptyState icon="home" title={t('list.emptyTitle')} body={t('list.emptyBody')} />
      </Card>
    );
  } else {
    results = (
      <ul className="wcs-list">
        {places.map((l) => (
          <li key={l.id}>
            <DirectoryCard place={l} catalog={catalog} hrefBase="/places" />
          </li>
        ))}
      </ul>
    );
  }

  const lastPage = meta?.last_page ?? 1;

  return (
    <DirectoryWebShell appHref={APP_DIRECTORY}>
      <section className="wcs-hero">
        <p className="wcs-eyebrow">{w('search.title')}</p>
        <h1 className="wcs-h1">{w('search.sub')}</h1>
        <p className="wcs-lead">{w('search.lead')}</p>
        <div className="wcs-searchbar" role="search">
          <label className="wcs-field wcs-field-grow">
            <Icon name="search" size={18} />
            <input
              type="search"
              value={qDraft}
              maxLength={80}
              placeholder={t('home.searchPlaceholder')}
              aria-label={t('home.searchLabel')}
              onChange={(e) => {
                setQDraft(e.target.value);
                setPage(1);
              }}
            />
          </label>
          <label className="wcs-field">
            <Icon name="mapPin" size={18} />
            <select
              value={activeCity ?? ''}
              aria-label={t('list.cityLabel')}
              onChange={(e) => {
                setCity(e.target.value ? Number(e.target.value) : null);
                setPage(1);
                setBbox(undefined);
              }}
            >
              <option value="">{t('list.allCities')}</option>
              {cities.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </label>
        </div>
        {options.data ? (
          <div role="radiogroup" aria-label={t('list.categoryLabel')} className="csv-chips wcs-chips">
            {chip('all', t('list.allCategories'), category === null, () => change(setCategory)(null))}
            {options.data.categories.map((c) => chip(c.slug, c.name, category === c.slug, () => change(setCategory)(c.slug)))}
          </div>
        ) : null}
      </section>

      <div className={clsx('wcs-grid', mapOk && 'has-map')}>
        <aside className="wcs-side" aria-label={w('search.filtersTitle')}>
          <div className="wcs-side-head">
            <h2 className="wcs-h2">{w('search.filtersTitle')}</h2>
            {!nothingActive ? (
              <button
                type="button"
                className="wcs-link-btn"
                onClick={() => {
                  applyFilters(EMPTY_FILTERS);
                  setBbox(undefined);
                  setCategory(null);
                  setQDraft('');
                  setQ('');
                }}
              >
                {w('search.clearAll')}
              </button>
            ) : null}
          </div>
          <fieldset className="wcs-group">
            <legend>{t('filters.age')}</legend>
            <div role="radiogroup" aria-label={t('filters.age')} className="wcs-opts">
              {chip('any', t('filters.anyAge'), filters.age === null, () => applyFilters({ ...filters, age: null }))}
              {(options.data?.place_age_ranges ?? []).map((a) => chip(a.code, a.name ?? a.code, filters.age === a.code, () => applyFilters({ ...filters, age: a.code })))}
            </div>
          </fieldset>
          <label className="wcs-check">
            <input type="checkbox" checked={filters.openNow} onChange={(e) => applyFilters({ ...filters, openNow: e.target.checked })} />
            {t('filters.openNow')}
          </label>
          {(options.data?.amenities ?? []).length > 0 ? (
            <fieldset className="wcs-group">
              <legend>{t('filters.amenities')}</legend>
              {(options.data?.amenities ?? []).map((a) => (
                <label key={a.code} className="wcs-check">
                  <input type="checkbox" checked={filters.amenities.includes(a.code)} onChange={() => toggleAmenity(a.code)} />
                  {a.name ?? a.code}
                </label>
              ))}
            </fieldset>
          ) : null}
          <fieldset className="wcs-group">
            <legend>{t('filters.sort')}</legend>
            {sorts.map((s) => (
              <label key={s} className="wcs-check">
                <input type="radio" name="wcs-sort" checked={filters.sort === s} onChange={() => applyFilters({ ...filters, sort: s })} />
                {t(`filters.sorts.${s}`)}
              </label>
            ))}
            {filters.sort === 'distance' ? <p className="wcs-hint">{t('filters.distanceNote')}</p> : null}
          </fieldset>
        </aside>

        <section className="wcs-results" aria-label={w('search.resultsCount', { count: num(total) })}>
          <div className="wcs-results-head">
            <h2 className="wcs-h2">
              {cityName ? w('search.resultsIn', { count: num(total), city: cityName }) : w('search.resultsCount', { count: num(total) })}
            </h2>
            {bbox ? (
              <button type="button" className="chip csv-chip on" onClick={() => change(setBbox)(undefined)}>
                {t('map.clearArea')}
                <Icon name="x" size={14} />
              </button>
            ) : null}
          </div>
          {!mapOk ? <p className="wcs-hint">{w('search.mapOff')}</p> : null}
          {results}
          {lastPage > 1 ? (
            <nav className="wcs-pager" aria-label={w('search.pagesLabel')}>
              <button type="button" className="wcs-pager-btn" disabled={page <= 1} onClick={() => setPage(page - 1)}>
                {t('list.prev')}
              </button>
              <span>{t('list.page', { page: num(page), last: num(lastPage) })}</span>
              <button type="button" className="wcs-pager-btn" disabled={page >= lastPage} onClick={() => setPage(page + 1)}>
                {t('list.next')}
              </button>
            </nav>
          ) : null}
          <p className="wcs-hint">{t('home.rankNote')}</p>
        </section>

        {mapOk ? (
          <div className="wcs-mapcol">
            <NeshanMap
              className="wcs-map"
              ariaLabel={t('map.ariaLabel')}
              pins={pins}
              center={near.point ?? undefined}
              selectedId={selected}
              onSelect={setSelected}
              card={pick ? <DirectoryCard place={pick} catalog={catalog} compact hrefBase="/places" /> : null}
              searchAreaLabel={t('map.searchArea')}
              onSearchArea={(b) => {
                change(setBbox)(bboxParam(b));
              }}
              myLocationLabel={t('map.myLocation')}
              onLocate={near.set}
              onLocateError={() => undefined}
              onUnavailable={() => setMapOk(false)}
            />
          </div>
        ) : null}
      </div>
    </DirectoryWebShell>
  );
}
