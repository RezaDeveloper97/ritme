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
import { Link, type Locale, useDirection, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { isMapEnabled, NeshanMap, type MapBbox, type MapPin } from '@/shared/ui/map';
import { Card, EmptyState, Icon, ScreenHeader, SecondaryButton, SkyLayer } from '@/shared/ui';
import { DirectoryCard, DirectoryFilterSheet } from '@/widgets/directory-filters';

import { CsError, CsSkeleton } from './ListSkeleton';
import '@/shared/ui/styles/page.css';
import '@/shared/ui/styles/csv.css';
import '@/shared/ui/styles/cdh.css';
import './dmp.css';

/** Map page size: the API caps `per_page`; a bounding box keeps a dense city below it. */
const MAP_LIMIT = 50;
/** The search box waits this long after the last keystroke before it asks the API. */
const SEARCH_DEBOUNCE_MS = 350;

const bboxParam = (b: MapBbox): string => `${b.west},${b.south},${b.east},${b.north}`;

/**
 * CB-DIR-07 — `nbl_Dir_Map`: places as pins, a search box, category chips, the filter sheet, «search this area»
 * (the visible box is sent as `bbox`), the selected place as a bottom card and a «list» toggle. Without a Neshan
 * key (or if the SDK fails) there is no map at all and the same places show as a list. The position is asked for
 * only when she taps the locate button and is never stored.
 */
export function DirectoryMapPage() {
  const t = useTranslations('cityServices');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const options = useCityServiceOptions();
  const [city, , ready] = useCityChoice();
  const [category, setCategory] = useState<string | null>(null);
  const [qDraft, setQDraft] = useState('');
  const [q, setQ] = useState('');
  const [filters, setFilters] = useState<DirectoryFilters>(EMPTY_FILTERS);
  const [sheet, setSheet] = useState(false);
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
  const list = useDirectoryListings({
    city: activeCity,
    category,
    q,
    ...toDirectoryParams(filters, catalog, near.point),
    bbox,
    perPage: MAP_LIMIT,
  });

  const places = useMemo(() => list.data?.listings ?? [], [list.data]);
  const pins = useMemo<MapPin[]>(
    () => places.filter((p) => p.lat !== null && p.lng !== null).map((p) => ({ id: String(p.id), lat: p.lat as number, lng: p.lng as number, label: p.name })),
    [places],
  );
  const pick = places.find((p) => String(p.id) === selected) ?? null;
  const total = list.data?.meta.total ?? places.length;
  const count = filterCount(filters);
  const narrowed = count > 0 || q.trim() !== '' || category !== null || bbox !== undefined;
  const center = near.point ?? undefined;
  const onMap = mapOk && ready;
  const rtl = useDirection() === 'rtl';

  const catChip = (key: string, label: string, on: boolean, onClick: () => void) => (
    <button key={key} type="button" role="radio" aria-checked={on} className={clsx('chip csv-chip', on && 'on')} onClick={onClick}>
      {label}
    </button>
  );

  const top = (
    <div className="dmp-top">
      <div className="dmp-search-row">
        {onMap ? (
          <button type="button" className="dmp-icon-btn" aria-label={t('back')} onClick={() => router.push('/city-services')}>
            <Icon name={rtl ? 'arrowR' : 'arrowL'} size={20} />
          </button>
        ) : null}
        <div className="cdh-search dmp-search" role="search">
          <Icon name="search" size={18} />
          <input
            type="search"
            value={qDraft}
            maxLength={80}
            placeholder={t('home.searchPlaceholder')}
            aria-label={t('home.searchLabel')}
            onChange={(e) => setQDraft(e.target.value)}
          />
        </div>
        <button type="button" className="dmp-icon-btn" aria-label={t('filters.title')} onClick={() => setSheet(true)}>
          <Icon name="filterLines" size={20} />
          {count > 0 ? <span className="dmp-dot">{formatNumber(count, locale)}</span> : null}
        </button>
      </div>
      {options.data ? (
        <div role="radiogroup" aria-label={t('list.categoryLabel')} className="csv-chips">
          {catChip('all', t('list.allCategories'), category === null, () => setCategory(null))}
          {options.data.categories.map((c) => catChip(c.slug, c.name, category === c.slug, () => setCategory(c.slug)))}
          {bbox ? (
            <button type="button" className="chip csv-chip on" onClick={() => setBbox(undefined)}>
              {t('map.clearArea')}
              <Icon name="x" size={14} />
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  );

  const toggle = (
    <Link href="/city-services" className="dmp-toggle">
      <Icon name="grid" size={16} />
      {t('map.listToggle', { count: formatNumber(total, locale) })}
    </Link>
  );

  const apply = (next: DirectoryFilters) => {
    setFilters(next);
    if (next.sort === 'distance' && !near.point) near.request();
  };

  const sheetEl = (
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
  );

  if (onMap) {
    return (
      <div className="view dmp-view">
        <NeshanMap
          className="dmp-map"
          ariaLabel={t('map.ariaLabel')}
          pins={pins}
          center={center}
          selectedId={selected}
          onSelect={setSelected}
          top={top}
          controls={toggle}
          card={pick ? <DirectoryCard place={pick} catalog={catalog} compact /> : null}
          searchAreaLabel={t('map.searchArea')}
          onSearchArea={(b) => {
            setBbox(bboxParam(b));
            setSelected(null);
          }}
          myLocationLabel={t('map.myLocation')}
          // The position only drives this visit's distance labels/sort; it is never stored or logged.
          onLocate={near.set}
          onLocateError={() => undefined}
          onUnavailable={() => setMapOk(false)}
        />
        {sheetEl}
      </div>
    );
  }

  // No map (no key locally, or the SDK did not load): the same places as a list.
  let body;
  if (options.isPending || list.isPending || !ready) {
    body = <CsSkeleton label={t('loading')} />;
  } else if (options.isError || list.isError) {
    body = (
      <CsError
        title={t('loadError')}
        retry={t('retry')}
        busy={list.isFetching}
        onRetry={() => {
          void options.refetch();
          void list.refetch();
        }}
      />
    );
  } else if (places.length === 0) {
    body = (
      <Card>
        <EmptyState
          icon="home"
          title={narrowed ? t('list.noMatchTitle') : t('list.emptyTitle')}
          body={narrowed ? t('list.noMatchBody') : t('list.emptyBody')}
          action={
            narrowed ? (
              <SecondaryButton
                onClick={() => {
                  setFilters(EMPTY_FILTERS);
                  setCategory(null);
                  setQ('');
                  setQDraft('');
                  setBbox(undefined);
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
      <ul className="csv-list">
        {places.map((l) => (
          <li key={l.id}>
            <DirectoryCard place={l} catalog={catalog} />
          </li>
        ))}
      </ul>
    );
  }
  return (
    <div className="view csv-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('map.title')} subtitle={t('map.unavailable')} onBack={() => router.back('/city-services')} backLabel={t('back')} />
        <div className="csv-body">
          {top}
          {body}
        </div>
        <div className="page-tail" />
      </div>
      {sheetEl}
    </div>
  );
}
