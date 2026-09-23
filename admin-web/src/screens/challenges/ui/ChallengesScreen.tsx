'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { formatNumber, useListParams } from '@/shared/lib';
import {
  Badge,
  CrudRowActions,
  DataTable,
  Icon,
  Pagination,
  Panel,
  SearchInput,
  Select,
  TextInput,
  useRowCommands,
  type Column,
} from '@/shared/ui';

import { challengesApi, type Challenge } from '../api/challenges';
import { useDayLabel } from './day-label';

/** /challenges — search, status and cycle-day filters (Blade challenges.index). */
export function ChallengesScreen() {
  const t = useTranslations('challenges');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const dayLabel = useDayLabel();
  const list = useListParams({ status: 'all', cycle_day: '' });
  const query = challengesApi.useList(list.query);
  const maxDay = challengesApi.useOptions().data?.max_cycle_day ?? 35;
  const commands = useRowCommands({ toggle: challengesApi.useAction('toggle'), remove: challengesApi.useRemove() });
  const filtered = Boolean(list.params.q || list.params.filters.cycle_day || list.params.filters.status !== 'all');

  const columns: Column<Challenge>[] = [
    { key: 'id', header: '#', cell: (c) => formatNumber(c.id, locale), className: 'cell-num w-14 text-muted' },
    { key: 'title', header: t('titleField'), cell: (c) => localize(c.title) || '—', className: 'cell-wrap font-semibold' },
    {
      key: 'days',
      header: t('cycleDays'),
      cell: (c) => (
        <Badge tone={c.cycle_day_from !== null || c.cycle_day_to !== null ? 'brand' : 'neutral'}>
          {dayLabel(c.cycle_day_from, c.cycle_day_to)}
        </Badge>
      ),
    },
    { key: 'category', header: tc('category'), cell: (c) => c.category || '—' },
    { key: 'sort', header: tc('sortOrder'), cell: (c) => formatNumber(c.sort_order, locale), className: 'cell-num' },
    {
      key: 'status',
      header: tc('status'),
      cell: (c) => (c.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge>{tc('inactive')}</Badge>),
    },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (c) => <CrudRowActions id={c.id} editHref={`/challenges/${c.id}`} commands={commands} active={c.is_active} />,
      className: 'cell-actions',
    },
  ];

  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        <>
          <Link href="/challenge-completions" className="btn btn-sm">
            {t('completionsReport')}
          </Link>
          <Link href="/challenges/new" className="btn btn-primary btn-sm">
            <Icon name="plus" size={15} />
            {t('new')}
          </Link>
        </>
      }
    >
      <div className="filter-bar">
        <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('searchPlaceholder')} />
        <TextInput
          label={t('cycleDayFilter')}
          className="w-32"
          type="number"
          min={1}
          max={maxDay}
          value={list.params.filters.cycle_day}
          onChange={(e) => list.setFilter('cycle_day', e.target.value)}
        />
        <Select
          label={tc('status')}
          className="w-36"
          value={list.params.filters.status}
          onChange={(e) => list.setFilter('status', e.target.value)}
          options={[
            { value: 'all', label: t('statusAll') },
            { value: 'active', label: tc('active') },
            { value: 'inactive', label: tc('inactive') },
          ]}
        />
      </div>
      {list.params.filters.cycle_day ? <p className="panel-note">{t('cycleDayFilterHint')}</p> : null}
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(c) => c.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={(c) => router.push(`/challenges/${c.id}`)}
        emptyText={filtered ? t('emptyFiltered') : t('empty')}
        caption={t('title')}
      />
      <Pagination meta={query.data?.meta} onPage={list.setPage} />
    </Panel>
  );
}
