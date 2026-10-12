'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import { useMemo, useState } from 'react';

import {
  parseShopListState,
  SHOP_SORTS,
  shopListParams,
  shopListSearch,
  type ShopCatalogFacet,
  type ShopCatalogList,
  type ShopListState,
  useShopCatalogProducts,
  useShopPrice,
} from '@/entities/shop';
import { getApiErrorStatus } from '@/shared/api';
import { Link, type Locale, usePathname, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Card, EmptyState, Icon, Skeleton, SkeletonGroup } from '@/shared/ui';

import { APP_SHOP, FACET_KEYS, FACET_SOURCE, type FacetKey, type Filters, NO_FILTERS, STORE_HOME, storeCategoryHref } from '../model/store';
import { ShopWebShell, WebTile } from './ShopWebShell';

const SKIN_KEYS = ['oily', 'combination', 'normal', 'dry', 'sensitive'];
const digits = (v: string) => v.replace(/\D/g, '');

/**
 * CB-SHOP-10 — `W_Shop_List`: breadcrumb, title, filter sidebar (sub-categories, facets, price), sort pills, product grid,
 * numbered pager. Filters are server-side (`/shop/catalog/products`); the first page arrives server-rendered. COD only.
 */
export function WebShopList({
  category,
  initial,
  initialSearch = '',
}: {
  category: string;
  initial?: ShopCatalogList | null;
  /** The query string the server rendered `initial` for (`shopListSearch` of the request's own state). */
  initialSearch?: string;
}) {
  const t = useTranslations('shop');
  const w = useTranslations('webShop');
  const locale = useLocale() as Locale;
  const price = useShopPrice();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  // Filters, sort and page live in the query string (2026-10-12): a reload, a shared link or a back from a product
  // restores the list, and the server renders the same first page. `go` rewrites it with router.replace.
  const state = useMemo(() => parseShopListState((k) => searchParams.get(k)), [searchParams]);
  const { filters, sort } = state;
  const [priceDraft, setPriceDraft] = useState({ min: filters.price_min, max: filters.price_max });

  const go = (next: Partial<ShopListState>) => {
    router.replace(`${pathname}${shopListSearch({ ...state, ...next })}`, { scroll: false });
  };
  const setPage = (n: number) => go({ page: n });

  const query = useShopCatalogProducts(shopListParams(category, state), {
    anon: true,
    // The server-rendered page is only the right data for the exact state it was rendered for.
    initialData: shopListSearch(state) === initialSearch ? (initial ?? undefined) : undefined,
  });
  const status = query.isError ? getApiErrorStatus(query.error) : undefined;
  const list = query.data;

  const apply = (f: Filters) => go({ filters: f, page: 1 });
  const toggle = (key: FacetKey, v: string) => apply({ ...filters, [key]: filters[key].includes(v) ? filters[key].filter((x) => x !== v) : [...filters[key], v] });
  const label = (key: FacetKey, f: ShopCatalogFacet) => (key === 'skin_type' && SKIN_KEYS.includes(f.value) ? t(`skin.${f.value as 'oily'}`) : f.label);
  const activeCount = FACET_KEYS.reduce((n, k) => n + filters[k].length, 0) + (filters.price_min || filters.price_max ? 1 : 0);
  const clear = () => {
    setPriceDraft({ min: '', max: '' });
    apply(NO_FILTERS);
  };

  const chips: { key: string; text: string; remove: () => void }[] = [];
  if (list) {
    for (const key of FACET_KEYS) {
      for (const v of filters[key]) {
        const f = list.filters[FACET_SOURCE[key]].find((x) => x.value === v);
        chips.push({ key: `${key}:${v}`, text: f ? label(key, f) : v, remove: () => toggle(key, v) });
      }
    }
    if (filters.price_min || filters.price_max) {
      chips.push({
        key: 'price',
        text: t('list.priceChip', {
          min: filters.price_min ? price(Number(filters.price_min)) : '—',
          max: filters.price_max ? price(Number(filters.price_max)) : '—',
        }),
        remove: () => {
          setPriceDraft({ min: '', max: '' });
          apply({ ...filters, price_min: '', price_max: '' });
        },
      });
    }
  }

  let content;
  if (status === 403 || status === 404 || (query.isError && !list)) {
    content = (
      <Card>
        <EmptyState
          icon={status === 403 ? 'shield' : status === 404 ? 'bag' : 'warning'}
          title={status === 403 ? t('home.teenTitle') : status === 404 ? t('product.notFoundTitle') : t('loadError')}
          body={status === 403 ? t('home.teenBody') : status === 404 ? t('product.notFoundBody') : undefined}
          action={
            <Link href={STORE_HOME} className="wsh-link-btn">
              {w('list.crumbRoot')}
            </Link>
          }
        />
      </Card>
    );
  } else if (!list) {
    content = (
      <SkeletonGroup label={t('loading')} className="wsh-skel">
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else {
    const groups = FACET_KEYS.map((key) => ({ key, items: list.filters[FACET_SOURCE[key]] })).filter((g) => g.items.length > 0);
    const { min, max } = list.filters.price;
    content = (
      <>
        <header className="wsh-listhead">
          <nav aria-label={w('list.crumbLabel')} className="wsh-crumbs">
            <Link href={STORE_HOME}>{w('list.crumbRoot')}</Link>
            <span aria-hidden>/</span>
            <span aria-current="page">{list.category?.title ?? t('title')}</span>
          </nav>
          <h1 className="wsh-h1">{list.category?.title ?? t('title')}</h1>
          <p className="wsh-lead">{t('list.itemCount', { n: list.category?.product_count ?? list.meta.total })}</p>
        </header>
        <div className="wsh-cols">
          <aside className="wsh-side" aria-label={w('list.sidebar')}>
            <div className="wsh-side-head">
              <h2 className="wsh-h3">{t('list.filters')}</h2>
              {activeCount > 0 ? (
                <button type="button" className="wsh-link-btn" onClick={clear}>
                  {w('list.clearAll')}
                </button>
              ) : null}
            </div>
            {list.sub_categories.length > 0 ? (
              <nav className="wsh-group" aria-label={w('list.categories')}>
                <h3 className="wsh-group-title">{w('list.categories')}</h3>
                <ul className="wsh-sublist">
                  {list.sub_categories.map((c) => (
                    <li key={c.id}>
                      <Link href={storeCategoryHref(c.slug)} className="wsh-sub">
                        <span>{c.title}</span>
                        <span className="wsh-sub-n">{formatNumber(c.product_count ?? 0, locale)}</span>
                      </Link>
                    </li>
                  ))}
                </ul>
              </nav>
            ) : null}
            {groups.map((g) => (
              <div key={g.key} role="group" aria-labelledby={`wsh-g-${g.key}`} className="wsh-group">
                <h3 id={`wsh-g-${g.key}`} className="wsh-group-title">
                  {t(`list.facet.${g.key}`)}
                </h3>
                {g.items.map((f) => (
                  <label key={f.value} className="wsh-check">
                    <input type="checkbox" checked={filters[g.key].includes(f.value)} onChange={() => toggle(g.key, f.value)} />
                    <span>{label(g.key, f)}</span>
                    <span className="wsh-sub-n">{formatNumber(f.count, locale)}</span>
                  </label>
                ))}
              </div>
            ))}
            <div role="group" aria-labelledby="wsh-g-price" className="wsh-group">
              <h3 id="wsh-g-price" className="wsh-group-title">
                {t('list.facet.price')}
              </h3>
              <div className="shp-price-range">
                <label className="shp-price-field">
                  <span>{t('list.priceMin')}</span>
                  <input
                    inputMode="numeric"
                    dir="ltr"
                    className="shp-price-input"
                    value={priceDraft.min}
                    placeholder={min !== null ? formatNumber(min, locale) : ''}
                    onChange={(e) => setPriceDraft((d) => ({ ...d, min: digits(e.target.value) }))}
                  />
                </label>
                <label className="shp-price-field">
                  <span>{t('list.priceMax')}</span>
                  <input
                    inputMode="numeric"
                    dir="ltr"
                    className="shp-price-input"
                    value={priceDraft.max}
                    placeholder={max !== null ? formatNumber(max, locale) : ''}
                    onChange={(e) => setPriceDraft((d) => ({ ...d, max: digits(e.target.value) }))}
                  />
                </label>
              </div>
              <p className="wsh-hint">{t('list.priceUnit')}</p>
              <button type="button" className="wsh-apply" onClick={() => apply({ ...filters, price_min: priceDraft.min, price_max: priceDraft.max })}>
                {w('list.applyPrice')}
              </button>
            </div>
          </aside>

          <div className="wsh-results">
            <div className="wsh-sortbar">
              <span className="wsh-sort-label" id="wsh-sort-l">
                {w('list.sortLabel')}
              </span>
              <div role="group" aria-labelledby="wsh-sort-l" className="wsh-sorts">
                {SHOP_SORTS.map((s) => (
                  <button
                    key={s}
                    type="button"
                    className={clsx('wsh-pill', s === sort && 'on')}
                    aria-pressed={s === sort}
                    onClick={() => go({ sort: s, page: 1 })}
                  >
                    {t(`list.sorts.${s}`)}
                  </button>
                ))}
              </div>
              <span className="wsh-count" aria-live="polite">
                {t('list.results', { n: list.meta.total })}
              </span>
            </div>
            {chips.length > 0 ? (
              <div className="wsh-chips">
                {chips.map((c) => (
                  <button key={c.key} type="button" className="shp-activechip" onClick={c.remove} aria-label={t('list.removeFilter', { name: c.text })}>
                    {c.text}
                    <Icon name="x" size={12} strokeWidth={2.6} />
                  </button>
                ))}
              </div>
            ) : null}
            {list.products.length === 0 ? (
              <Card>
                <EmptyState
                  icon="search"
                  title={t('list.emptyTitle')}
                  body={t('list.emptyBody')}
                  action={
                    activeCount > 0 ? (
                      <button type="button" className="wsh-link-btn" onClick={clear}>
                        {t('list.reset')}
                      </button>
                    ) : undefined
                  }
                />
              </Card>
            ) : (
              <ul className="wsh-tiles wsh-tiles-list">
                {list.products.map((p) => (
                  <WebTile key={p.id} p={p} />
                ))}
              </ul>
            )}
            {list.meta.last_page > 1 ? (
              <nav className="wsh-pager" aria-label={t('products.pages')}>
                {Array.from({ length: list.meta.last_page }, (_, i) => i + 1).map((n) => (
                  <button
                    key={n}
                    type="button"
                    className={clsx('wsh-page', n === list.meta.current_page && 'on')}
                    aria-current={n === list.meta.current_page ? 'page' : undefined}
                    aria-label={w('list.pageN', { n: formatNumber(n, locale) })}
                    onClick={() => setPage(n)}
                  >
                    {formatNumber(n, locale)}
                  </button>
                ))}
              </nav>
            ) : null}
            <p className="shp-note wsh-note">{t('list.sizeNote')}</p>
          </div>
        </div>
      </>
    );
  }

  return (
    <ShopWebShell appHref={`${APP_SHOP}/${category}`}>
      <div className="wsh">
        {content}
        <p className="shp-note wsh-note">{t('home.cod')}</p>
      </div>
    </ShopWebShell>
  );
}
