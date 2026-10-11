'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { formatDateTime, formatNumber, useListParams } from '@/shared/lib';
import { Badge, DataTable, PageHeader, Pagination, Panel, SearchInput, Select, type BadgeTone, type Column } from '@/shared/ui';

import { useCityServiceOptions } from '../api/city-services';
import { applicationsApi, type Application } from '../api/directory';

export const APP_TONE: Record<string, BadgeTone> = { submitted: 'amber', in_review: 'brand', needs_info: 'amber', approved: 'green', rejected: 'red' };

/**
 * /city-services/applications — the business join review queue (CB-DIR-05, capability content). Open work first;
 * drafts never appear. Contact details are only sent to admins who also hold «support».
 */
export function CityServiceApplicationsScreen() {
  const t = useTranslations('cityServices.dir.apps');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const options = useCityServiceOptions();
  const list = useListParams({ status: 'all' });
  const status = list.params.filters.status ?? 'all';
  const query = applicationsApi.useList({
    ...list.query,
    ...(list.params.q ? { q: list.params.q } : {}),
    ...(status === 'all' ? {} : { status }),
  });
  const cityName = (id: number | null) => {
    const c = options.data?.cities.find((x) => x.id === id);
    return c ? localize(c.name) || c.slug : '—';
  };
  const catName = (slug: string) => {
    const c = options.data?.categories.find((x) => x.slug === slug);
    return c ? localize(c.name) || slug : slug;
  };

  const columns: Column<Application>[] = [
    { key: 'submitted', header: t('submittedAt'), cell: (a) => (a.submitted_at ? formatDateTime(a.submitted_at, locale) : '—'), className: 'cell-num whitespace-nowrap' },
    {
      key: 'business',
      header: t('business'),
      cell: (a) => (
        <span className="flex flex-col">
          <span>{a.business_name ?? '—'}</span>
          {a.tracking_code ? (
            <span dir="ltr" className="text-xs text-muted">
              {a.tracking_code}
            </span>
          ) : null}
        </span>
      ),
    },
    { key: 'categories', header: t('categories'), cell: (a) => a.category_slugs.map(catName).join('، ') || '—' },
    { key: 'city', header: t('city'), cell: (a) => [cityName(a.city_id), a.district].filter(Boolean).join(' · '), className: 'whitespace-nowrap' },
    { key: 'photos', header: t('photos'), cell: (a) => formatNumber(a.photos_count, locale), className: 'cell-num' },
    { key: 'status', header: t('status'), cell: (a) => <Badge tone={APP_TONE[a.status] ?? 'neutral'}>{t(`statusName.${a.status}` as 'statusName.submitted')}</Badge> },
  ];

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={t('title')} backHref="/city-services" backLabel={t('backToList')} />
      <Panel bodyClassName="">
        <div className="filter-bar">
          <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('searchPlaceholder')} />
          <Select
            aria-label={t('status')}
            value={status}
            onChange={(e) => list.setFilter('status', e.target.value)}
            options={[
              { value: 'all', label: t('allStatuses') },
              ...(options.data?.application_statuses ?? []).map((s) => ({ value: s, label: t(`statusName.${s}` as 'statusName.submitted') })),
            ]}
          />
        </div>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(a) => a.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          onRowClick={(a) => router.push(`/city-services/applications/${a.id}`)}
          emptyText={t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}
