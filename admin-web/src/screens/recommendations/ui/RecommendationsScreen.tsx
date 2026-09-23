'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { excerpt, formatNumber, optionLabel, useListParams } from '@/shared/lib';
import {
  Badge,
  CrudRowActions,
  DataTable,
  Icon,
  Pagination,
  Panel,
  Select,
  useRowCommands,
  type Column,
} from '@/shared/ui';

import { recommendationsApi, type Recommendation } from '../api/recommendations';

/** /recommendations — phase and type filters (Blade recommendations.index). */
export function RecommendationsScreen() {
  const t = useTranslations('recommendations');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const list = useListParams({ phase: '', type: '' });
  const query = recommendationsApi.useList(list.query);
  const options = recommendationsApi.useOptions().data;
  const commands = useRowCommands({ toggle: recommendationsApi.useAction('toggle'), remove: recommendationsApi.useRemove() });

  const columns: Column<Recommendation>[] = [
    { key: 'id', header: '#', cell: (r) => formatNumber(r.id, locale), className: 'cell-num w-14 text-muted' },
    { key: 'type', header: t('type'), cell: (r) => optionLabel(options?.types, r.type), className: 'whitespace-nowrap' },
    { key: 'text', header: t('text'), cell: (r) => excerpt(localize(r.text), 90) || '—', className: 'cell-wrap' },
    { key: 'phase', header: tc('phase'), cell: (r) => optionLabel(options?.phases, r.cycle_phase) || tc('allPhases') },
    {
      key: 'subphases',
      header: t('subphases'),
      cell: (r) =>
        r.cycle_subphases.length ? (
          <span className="flex flex-wrap gap-1">
            {r.cycle_subphases.map((s) => (
              <Badge key={s}>{optionLabel(options?.subphases, s)}</Badge>
            ))}
          </span>
        ) : (
          '—'
        ),
    },
    { key: 'trigger', header: t('trigger'), cell: (r) => optionLabel(options?.triggers, r.symptom_trigger) || '—' },
    {
      key: 'status',
      header: tc('status'),
      cell: (r) => (r.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge>{tc('inactive')}</Badge>),
    },
    { key: 'sort', header: tc('sortOrder'), cell: (r) => formatNumber(r.sort_order, locale), className: 'cell-num' },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (r) => <CrudRowActions id={r.id} editHref={`/recommendations/${r.id}`} commands={commands} active={r.is_active} />,
      className: 'cell-actions',
    },
  ];

  const filtered = Boolean(list.params.filters.phase || list.params.filters.type);

  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        <Link href="/recommendations/new" className="btn btn-primary btn-sm">
          <Icon name="plus" size={15} />
          {t('new')}
        </Link>
      }
    >
      <div className="filter-bar">
        <Select
          label={tc('phase')}
          className="w-48"
          value={list.params.filters.phase}
          onChange={(e) => list.setFilter('phase', e.target.value)}
          options={[
            { value: '', label: tc('allPhases') },
            { value: 'general', label: t('generalOnly') },
            ...(options?.phases ?? []),
          ]}
        />
        <Select
          label={t('type')}
          className="w-48"
          value={list.params.filters.type}
          onChange={(e) => list.setFilter('type', e.target.value)}
          options={[{ value: '', label: t('allTypes') }, ...(options?.types ?? [])]}
        />
      </div>
      <p className="panel-note">{t('intro')}</p>
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(r) => r.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={(r) => router.push(`/recommendations/${r.id}`)}
        emptyText={filtered ? t('emptyFiltered') : t('empty')}
        caption={t('title')}
      />
      <Pagination meta={query.data?.meta} onPage={list.setPage} />
    </Panel>
  );
}
