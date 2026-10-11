'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { formatNumber, useListParams } from '@/shared/lib';
import { Badge, CrudRowActions, DataTable, Icon, Pagination, Panel, SearchInput, Select, useRowCommands, type Column } from '@/shared/ui';

import { listingsApi, useCityServiceOptions, type Listing } from '../api/city-services';

/** /city-services — the venue listings (B-N10-05); cities, categories and the request queue are linked from here. */
export function CityServicesScreen() {
  const t = useTranslations('cityServices');
  const td = useTranslations('cityServices.dir');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const list = useListParams({ status: 'all', city: 'all', category: 'all' });
  const { status, city, category } = { status: 'all', city: 'all', category: 'all', ...list.params.filters };
  const options = useCityServiceOptions();
  const query = listingsApi.useList({
    ...list.query,
    ...(list.params.q ? { q: list.params.q } : {}),
    ...(status !== 'all' ? { status } : {}),
    ...(city !== 'all' ? { city_id: city } : {}),
    ...(category !== 'all' ? { category_id: category } : {}),
  });
  const commands = useRowCommands({ toggle: listingsApi.useAction('toggle'), remove: listingsApi.useRemove() });

  const columns: Column<Listing>[] = [
    { key: 'id', header: '#', cell: (l) => formatNumber(l.id, locale), className: 'cell-num w-14 text-muted' },
    {
      key: 'photo',
      header: t('photo'),
      // eslint-disable-next-line @next/next/no-img-element -- API-hosted upload
      cell: (l) => ((l.file_photo_urls[0] ?? l.photo_urls[0]) ? <img src={l.file_photo_urls[0] ?? l.photo_urls[0]} alt="" className="thumb" loading="lazy" /> : '—'),
    },
    { key: 'name', header: t('name'), cell: (l) => l.name },
    { key: 'category', header: t('category'), cell: (l) => localize(l.category.name) || l.category.slug, className: 'whitespace-nowrap' },
    { key: 'city', header: t('city'), cell: (l) => [localize(l.city.name) || l.city.slug, l.district].filter(Boolean).join(' · '), className: 'whitespace-nowrap' },
    { key: 'mode', header: t('bookingMode'), cell: (l) => <Badge tone="brand">{t(`modes.${l.booking_mode}` as 'modes.link')}</Badge> },
    {
      key: 'status',
      header: tc('status'),
      cell: (l) => (
        <span className="flex flex-wrap gap-1">
          {l.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge tone="amber">{tc('inactive')}</Badge>}
          {l.status !== 'published' ? <Badge tone="neutral">{td(`placeStatus.${l.status}` as 'placeStatus.published')}</Badge> : null}
          {l.verified ? <Badge tone="brand">{td('verified')}</Badge> : null}
        </span>
      ),
    },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (l) => <CrudRowActions id={l.id} editHref={`/city-services/${l.id}`} commands={commands} active={l.is_active} />,
      className: 'cell-actions',
    },
  ];

  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        <>
          <Link href="/city-service-bookings" className="btn btn-sm">
            {t('queueLink')}
          </Link>
          <Link href="/city-services/applications" className="btn btn-sm">
            {td('applicationsLink')}
          </Link>
          <Link href="/city-services/reviews" className="btn btn-sm">
            {td('reviewsLink')}
          </Link>
          <Link href="/city-services/reports" className="btn btn-sm">
            {td('reportsLink')}
          </Link>
          <Link href="/city-services/cities" className="btn btn-sm">
            {t('cities')}
          </Link>
          <Link href="/city-services/categories" className="btn btn-sm">
            {t('categories')}
          </Link>
          <Link href="/city-services/new" className="btn btn-primary btn-sm">
            <Icon name="plus" size={15} />
            {t('new')}
          </Link>
        </>
      }
    >
      <div className="filter-bar">
        <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('searchPlaceholder')} />
        <Select
          aria-label={t('city')}
          value={city}
          onChange={(e) => list.setFilter('city', e.target.value)}
          options={[{ value: 'all', label: t('allCities') }, ...(options.data?.cities ?? []).map((c) => ({ value: String(c.id), label: localize(c.name) || c.slug }))]}
        />
        <Select
          aria-label={t('category')}
          value={category}
          onChange={(e) => list.setFilter('category', e.target.value)}
          options={[{ value: 'all', label: t('allCategories') }, ...(options.data?.categories ?? []).map((c) => ({ value: String(c.id), label: localize(c.name) || c.slug }))]}
        />
        <Select
          aria-label={tc('status')}
          value={status}
          onChange={(e) => list.setFilter('status', e.target.value)}
          options={[
            { value: 'all', label: t('allStatuses') },
            { value: 'active', label: tc('active') },
            { value: 'inactive', label: tc('inactive') },
          ]}
        />
      </div>
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(l) => l.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={(l) => router.push(`/city-services/${l.id}`)}
        emptyText={t('empty')}
        caption={t('title')}
      />
      <Pagination meta={query.data?.meta} onPage={list.setPage} />
    </Panel>
  );
}
