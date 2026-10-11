'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { SHOP_SORTS, useShopCatalogProducts, type ShopCatalogFacet, type ShopCatalogList, type ShopSort } from '@/entities/shop';
import { getApiErrorStatus } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import {
  Card,
  ChipGroup,
  EmptyState,
  Icon,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SearchField,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { ProductTile, usePrice, useSkinLabel } from './ProductTile';

/** Selected filter values; multi-valued filters are kept as arrays and comma-joined for the API. */
interface Filters {
  size: string[];
  color: string[];
  volume: string[];
  material: string[];
  skin_type: string[];
  price_min: string;
  price_max: string;
}

const NO_FILTERS: Filters = { size: [], color: [], volume: [], material: [], skin_type: [], price_min: '', price_max: '' };
const FACET_KEYS = ['size', 'color', 'volume', 'material', 'skin_type'] as const;
type FacetKey = (typeof FACET_KEYS)[number];
const FACET_SOURCE: Record<FacetKey, 'sizes' | 'colors' | 'volumes' | 'materials' | 'skin_types'> = {
  size: 'sizes',
  color: 'colors',
  volume: 'volumes',
  material: 'materials',
  skin_type: 'skin_types',
};

const digits = (v: string) => v.replace(/\D/g, '');

function FilterSheet({
  open,
  onClose,
  list,
  value,
  onApply,
}: {
  open: boolean;
  onClose: () => void;
  list: ShopCatalogList;
  value: Filters;
  onApply: (f: Filters) => void;
}) {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  const skinLabel = useSkinLabel();
  const [draft, setDraft] = useState<Filters>(value);
  // Re-seed from the applied filters every time the sheet opens.
  useEffect(() => {
    if (open) setDraft(value);
  }, [open, value]);

  const toggle = (key: FacetKey, v: string) =>
    setDraft((d) => ({ ...d, [key]: d[key].includes(v) ? d[key].filter((x) => x !== v) : [...d[key], v] }));

  const groups = FACET_KEYS.map((key) => ({ key, items: list.filters[FACET_SOURCE[key]] })).filter((g) => g.items.length > 0);
  const { min, max } = list.filters.price;

  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="full"
      title={t('list.filters')}
      footer={
        <div className="shp-sheet-actions">
          <SecondaryButton variant="text" onClick={() => setDraft(NO_FILTERS)}>
            {t('list.reset')}
          </SecondaryButton>
          <PrimaryButton
            onClick={() => {
              onApply(draft);
              onClose();
            }}
          >
            {t('list.apply')}
          </PrimaryButton>
        </div>
      }
    >
      <div className="shp-filters">
        {groups.map((g) => (
          <section key={g.key} className="shp-fgroup">
            <h3 className="shp-fgroup-title">{t(`list.facet.${g.key}`)}</h3>
            <ChipGroup label={t(`list.facet.${g.key}`)}>
              {g.items.map((f: ShopCatalogFacet) => (
                <PillChip key={f.value} mode="multi" pressed={draft[g.key].includes(f.value)} onPressedChange={() => toggle(g.key, f.value)}>
                  {g.key === 'skin_type' ? skinLabel(f.value, f.label) : f.label}
                </PillChip>
              ))}
            </ChipGroup>
          </section>
        ))}
        <section className="shp-fgroup">
          <h3 className="shp-fgroup-title">{t('list.facet.price')}</h3>
          <div className="shp-price-range">
            <label className="shp-price-field">
              <span>{t('list.priceMin')}</span>
              <input
                inputMode="numeric"
                dir="ltr"
                className="shp-price-input"
                value={draft.price_min}
                placeholder={min !== null ? formatNumber(min, locale) : ''}
                onChange={(e) => setDraft((d) => ({ ...d, price_min: digits(e.target.value) }))}
              />
            </label>
            <label className="shp-price-field">
              <span>{t('list.priceMax')}</span>
              <input
                inputMode="numeric"
                dir="ltr"
                className="shp-price-input"
                value={draft.price_max}
                placeholder={max !== null ? formatNumber(max, locale) : ''}
                onChange={(e) => setDraft((d) => ({ ...d, price_max: digits(e.target.value) }))}
              />
            </label>
          </div>
          <p className="shp-fhint">{t('list.priceUnit')}</p>
        </section>
      </div>
    </AppSheet>
  );
}

/**
 * `/shop/[category]` for an internal catalog category (CB-SHOP-07, nbd_Shop_List): search, filter sheet with removable
 * active chips, sub-category tabs, sort and a 2-column grid. Filters are server-side (`/shop/catalog/products`).
 * Pay on delivery only — no online-payment wording. Teen mode (403) shows an unavailable state.
 */
export function ShopListPage({ category }: { category: string }) {
  const t = useTranslations('shop');
  const router = useRouter();
  const locale = useLocale() as Locale;
  const price = usePrice();
  const skinLabel = useSkinLabel();
  const [qInput, setQInput] = useState('');
  const [q, setQ] = useState('');
  const [filters, setFilters] = useState<Filters>(NO_FILTERS);
  const [sort, setSort] = useState<ShopSort>('popular');
  const [page, setPage] = useState(1);
  const [sheet, setSheet] = useState(false);

  // Debounce the search box; a new query restarts at page 1.
  useEffect(() => {
    const id = window.setTimeout(() => {
      setQ((prev) => {
        if (prev !== qInput.trim()) setPage(1);
        return qInput.trim();
      });
    }, 350);
    return () => window.clearTimeout(id);
  }, [qInput]);

  const query = useShopCatalogProducts({
    category,
    q: q || undefined,
    size: filters.size.join(',') || undefined,
    color: filters.color.join(',') || undefined,
    volume: filters.volume.join(',') || undefined,
    material: filters.material.join(',') || undefined,
    skin_type: filters.skin_type.join(',') || undefined,
    price_min: filters.price_min ? Number(filters.price_min) : undefined,
    price_max: filters.price_max ? Number(filters.price_max) : undefined,
    sort,
    page,
  });
  const status = query.isError ? getApiErrorStatus(query.error) : undefined;
  const list = query.data;

  const apply = (f: Filters) => {
    setFilters(f);
    setPage(1);
  };
  const active: { key: string; label: string; remove: () => void }[] = [];
  if (list) {
    for (const key of FACET_KEYS) {
      for (const v of filters[key]) {
        const facet = list.filters[FACET_SOURCE[key]].find((f) => f.value === v);
        const label = key === 'skin_type' ? skinLabel(v, facet?.label ?? v) : (facet?.label ?? v);
        active.push({ key: `${key}:${v}`, label, remove: () => apply({ ...filters, [key]: filters[key].filter((x) => x !== v) }) });
      }
    }
    if (filters.price_min || filters.price_max) {
      active.push({
        key: 'price',
        label: t('list.priceChip', {
          min: filters.price_min ? price(Number(filters.price_min)) : '—',
          max: filters.price_max ? price(Number(filters.price_max)) : '—',
        }),
        remove: () => apply({ ...filters, price_min: '', price_max: '' }),
      });
    }
  }

  let body;
  if (status === 403) {
    body = (
      <Card>
        <EmptyState icon="shield" title={t('home.teenTitle')} body={t('home.teenBody')} />
      </Card>
    );
  } else if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="shp-grid">
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (query.isError || !list) {
    body = (
      <Card>
        <EmptyState
          icon="warning"
          title={t('loadError')}
          action={
            <PrimaryButton onClick={() => void query.refetch()} loading={query.isFetching}>
              {t('retry')}
            </PrimaryButton>
          }
        />
      </Card>
    );
  } else {
    body = (
      <>
        {list.sub_categories.length ? (
          <ul className="shp-chips shp-subtabs" aria-label={t('home.categories')}>
            <li>
              <span className="shp-chip is-on" aria-current="page">
                {t('list.all')}
              </span>
            </li>
            {list.sub_categories.map((c) => (
              <li key={c.id}>
                <Link href={`/shop/${c.slug}`} className="shp-chip">
                  {c.title}
                </Link>
              </li>
            ))}
          </ul>
        ) : null}
        <div className="shp-resultbar">
          <span className="shp-count" aria-live="polite">
            {t('list.results', { n: list.meta.total })}
          </span>
          <label className="shp-sort">
            <span className="shp-sort-label">{t('list.sort')}</span>
            <select
              value={sort}
              onChange={(e) => {
                setSort(e.target.value as ShopSort);
                setPage(1);
              }}
            >
              {SHOP_SORTS.map((s) => (
                <option key={s} value={s}>
                  {t(`list.sorts.${s}`)}
                </option>
              ))}
            </select>
          </label>
        </div>
        {list.products.length === 0 ? (
          <Card>
            <EmptyState
              icon="search"
              title={t('list.emptyTitle')}
              body={t('list.emptyBody')}
              action={
                active.length || q ? (
                  <SecondaryButton
                    onClick={() => {
                      setQInput('');
                      apply(NO_FILTERS);
                    }}
                  >
                    {t('list.reset')}
                  </SecondaryButton>
                ) : undefined
              }
            />
          </Card>
        ) : (
          <ul className="shp-tiles">
            {list.products.map((p) => (
              <ProductTile key={p.id} p={p} />
            ))}
          </ul>
        )}
        {list.meta.last_page > 1 ? (
          <nav className="shp-pager" aria-label={t('products.pages')}>
            <SecondaryButton variant="text" disabled={page <= 1} onClick={() => setPage((n) => n - 1)}>
              {t('products.prev')}
            </SecondaryButton>
            <span>{t('products.page', { page, last: list.meta.last_page })}</span>
            <SecondaryButton variant="text" disabled={page >= list.meta.last_page} onClick={() => setPage((n) => n + 1)}>
              {t('products.next')}
            </SecondaryButton>
          </nav>
        ) : null}
        <p className="shp-note">{t('list.sizeNote')}</p>
        <FilterSheet open={sheet} onClose={() => setSheet(false)} list={list} value={filters} onApply={apply} />
      </>
    );
  }

  return (
    <div className="view shp-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={list?.category?.title ?? t('title')}
          subtitle={list ? t('list.itemCount', { n: list.category?.product_count ?? list.meta.total }) : undefined}
          onBack={() => router.push('/shop')}
          backLabel={t('back')}
        />
        <div className="shp-body">
          {status === 403 ? null : (
            <>
              <SearchField
                value={qInput}
                onValueChange={setQInput}
                label={t('list.search')}
                clearLabel={t('list.clear')}
                placeholder={list?.category ? t('list.searchIn', { name: list.category.title }) : t('list.search')}
              />
              <div className="shp-filterbar">
                <button type="button" className="shp-filterbtn" onClick={() => setSheet(true)} disabled={!list}>
                  <Icon name="filterLines" size={16} />
                  {t('list.filters')}
                  {active.length ? <span className="shp-filter-n">{formatNumber(active.length, locale)}</span> : null}
                </button>
                {active.map((a) => (
                  <button key={a.key} type="button" className="shp-activechip" onClick={a.remove} aria-label={t('list.removeFilter', { name: a.label })}>
                    {a.label}
                    <Icon name="x" size={12} strokeWidth={2.6} />
                  </button>
                ))}
              </div>
            </>
          )}
          {body}
          <p className="shp-note">{t('home.cod')}</p>
        </div>
        <div className="page-tail" />
      </div>
    </div>
  );
}
