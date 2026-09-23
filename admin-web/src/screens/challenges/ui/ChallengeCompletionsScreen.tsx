'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';

import { useErrorMessage, useLocalized } from '@/shared/i18n';
import { formatDate, formatNumber, useListParams } from '@/shared/lib';
import {
  Button,
  DataTable,
  ErrorState,
  Pagination,
  Panel,
  SearchInput,
  Select,
  Skeleton,
  TextInput,
  type Column,
} from '@/shared/ui';

import { useCompletions, type Completion, type PerChallenge } from '../api/challenges';

const FILTERS = { challenge_id: '', from: '', to: '' };

/** /challenge-completions — who completed which challenge, when (Blade challenges.completions). */
export function ChallengeCompletionsScreen() {
  const t = useTranslations('completions');
  const locale = useLocale();
  const localize = useLocalized();
  const describe = useErrorMessage();
  const list = useListParams(FILTERS);
  const query = useCompletions(list.query);
  const data = query.data;
  const n = (v: number) => formatNumber(v, locale);
  const filtered = Boolean(list.params.q || list.params.filters.challenge_id || list.params.filters.from || list.params.filters.to);

  const perColumns: Column<PerChallenge>[] = [
    { key: 'title', header: t('challenge'), cell: (r) => localize(r.title) || `#${n(r.challenge_id)}`, className: 'cell-wrap' },
    { key: 'completions', header: t('completions'), cell: (r) => n(r.completions), className: 'cell-num' },
    { key: 'users', header: t('uniqueUsers'), cell: (r) => n(r.users), className: 'cell-num' },
  ];

  const columns: Column<Completion>[] = [
    { key: 'id', header: '#', cell: (r) => n(r.id), className: 'cell-num w-14 text-muted' },
    {
      key: 'user',
      header: t('user'),
      cell: (r) => (
        <Link href={`/users/${r.user.id}`} className="font-semibold text-brand no-underline hover:underline">
          {r.user.name || t('userFallback', { id: n(r.user.id) })}
        </Link>
      ),
    },
    {
      key: 'mobile',
      header: t('mobile'),
      cell: (r) => (r.user.mobile ? <span dir="ltr">{r.user.mobile}</span> : '—'),
      className: 'cell-num',
    },
    { key: 'challenge', header: t('challenge'), cell: (r) => localize(r.challenge.title) || '—', className: 'cell-wrap' },
    {
      key: 'date',
      header: t('date'),
      cell: (r) => formatDate(r.completion_date, locale),
      className: 'cell-num whitespace-nowrap text-ink-3',
    },
  ];

  const challengeOptions = [
    { value: '', label: t('allChallenges') },
    ...(data?.challenges ?? []).map((c) => ({ value: String(c.id), label: localize(c.title) || `#${n(c.id)}` })),
  ];

  return (
    <div className="flex flex-col gap-5">
      <Panel
        title={t('title')}
        bodyClassName=""
        actions={
          <Link href="/challenges" className="btn btn-sm">
            {t('manageChallenges')}
          </Link>
        }
      >
        <div className="filter-bar">
          <Select
            label={t('challenge')}
            className="min-w-48 flex-1 sm:max-w-72"
            value={list.params.filters.challenge_id}
            onChange={(e) => list.setFilter('challenge_id', e.target.value)}
            options={challengeOptions}
          />
          <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('searchPlaceholder')} />
          <TextInput
            label={t('from')}
            type="date"
            className="w-40"
            value={list.params.filters.from}
            max={list.params.filters.to || undefined}
            onChange={(e) => list.setFilter('from', e.target.value)}
          />
          <TextInput
            label={t('to')}
            type="date"
            className="w-40"
            value={list.params.filters.to}
            min={list.params.filters.from || undefined}
            onChange={(e) => list.setFilter('to', e.target.value)}
          />
          {filtered ? (
            <Button
              size="sm"
              variant="ghost"
              onClick={list.reset}
            >
              {t('clearFilters')}
            </Button>
          ) : null}
        </div>
        {query.error && !data ? (
          <ErrorState error={query.error} onRetry={() => query.refetch()} />
        ) : (
          <dl className="ledger m-0">
            {(
              [
                ['total', data?.stats.total],
                ['users', data?.stats.users],
                ['today', data?.stats.today],
              ] as const
            ).map(([key, value]) => (
              <div key={key} className="ledger-cell">
                <dt className="text-[13px] font-semibold text-ink-3">{t(`stats.${key}`)}</dt>
                <dd className="m-0 mt-1">
                  {value === undefined ? <Skeleton className="h-8 w-16" /> : <span className="ledger-value">{n(value)}</span>}
                </dd>
              </div>
            ))}
          </dl>
        )}
        {query.error && data ? <p className="panel-note text-danger-deep">{describe(query.error)}</p> : null}
      </Panel>

      <Panel title={t('perChallenge')} bodyClassName="">
        <DataTable
          columns={perColumns}
          items={data?.per_challenge}
          rowKey={(r) => r.challenge_id}
          loading={query.isPending}
          emptyText={t('empty')}
          skeletonRows={3}
          caption={t('perChallenge')}
        />
      </Panel>

      <Panel title={t('latest')} bodyClassName="">
        <DataTable
          columns={columns}
          items={data?.items}
          rowKey={(r) => r.id}
          loading={query.isPending}
          emptyText={filtered ? t('emptyFiltered') : t('empty')}
          caption={t('latest')}
        />
        <Pagination meta={data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}
