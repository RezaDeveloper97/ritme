'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';

import { useLocalized } from '@/shared/i18n';
import { formatNumber, useListParams } from '@/shared/lib';
import { Badge, DataTable, LinkTabs, PageHeader, Pagination, Panel, SearchInput, type Column } from '@/shared/ui';

import { doctorsApi, type Doctor } from '../api/telemed';

const STATUSES = ['all', 'active', 'inactive'] as const;

/** /telemed/doctors — the directory (admin-api.md §19). */
export function DoctorsScreen() {
  const t = useTranslations('telemed.doctors');
  const locale = useLocale();
  const localize = useLocalized();
  const list = useListParams({ status: 'all' });
  const status = list.params.filters.status ?? 'all';
  const query = doctorsApi.useList(list.query);
  const n = (v: number) => formatNumber(v, locale);

  const columns: Column<Doctor>[] = [
    {
      key: 'name',
      header: t('name'),
      cell: (d) => (
        <span className="flex flex-col items-start">
          <Link href={`/telemed/doctors/${d.id}`} className="font-semibold text-brand no-underline hover:underline" dir="auto">
            {localize(d.name) || `#${d.id}`}
          </Link>
          <span className="text-xs text-ink-3">{t(`kind.${d.kind}` as 'kind.doctor')}</span>
        </span>
      ),
    },
    { key: 'specialty', header: t('specialty'), cell: (d) => d.specialty },
    {
      key: 'modes',
      header: t('modes'),
      cell: (d) =>
        d.modes.length ? (
          <span className="flex flex-wrap gap-1">
            {d.modes.map((m) => (
              <Badge key={m}>{t(`mode.${m}` as 'mode.video')}</Badge>
            ))}
          </span>
        ) : (
          <span className="text-muted">{t('noModes')}</span>
        ),
    },
    {
      key: 'rating',
      header: t('rating'),
      cell: (d) => (d.reviews_count ? `${n(Number(d.rating ?? 0))} (${n(d.reviews_count)})` : '—'),
      className: 'cell-num',
    },
    { key: 'visits', header: t('visits'), cell: (d) => n(d.visits_count), className: 'cell-num' },
    {
      key: 'status',
      header: t('status'),
      cell: (d) => <Badge tone={d.is_active ? 'green' : 'neutral'}>{d.is_active ? t('active') : t('inactive')}</Badge>,
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t('title')}
        actions={
          <Link href="/telemed/doctors/new" className="btn btn-primary">
            {t('new')}
          </Link>
        }
      />
      <LinkTabs
        label={t('title')}
        items={STATUSES.map((s) => ({
          key: s,
          href: s === 'all' ? '/telemed/doctors' : `/telemed/doctors?status=${s}`,
          label: t(`tabs.${s}`),
          active: s === status,
        }))}
      />
      <Panel bodyClassName="">
        <div className="filter-bar">
          <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('searchPlaceholder')} />
        </div>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(d) => d.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={list.params.q || status !== 'all' ? t('emptyFiltered') : t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}
