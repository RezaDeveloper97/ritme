'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';

import { formatDateTime, formatNumber, useListParams } from '@/shared/lib';
import { Badge, Button, DataTable, PageHeader, Pagination, Panel, SearchInput, Select, toast, useNotifyError, type BadgeTone, type Column } from '@/shared/ui';

import { labRequestsApi, useSetLabStatus, type LabRequestRow } from '../api/partnerLab';

const TONE: Record<string, BadgeTone> = {
  requested: 'amber',
  accepted: 'data',
  sampled: 'data',
  result_ready: 'green',
  rejected: 'red',
  cancelled: 'neutral',
};

/** What the console may do next for the fake partner (the same moves the server's state machine allows). */
const NEXT: Record<string, readonly string[]> = {
  requested: ['accepted', 'rejected'],
  accepted: ['sampled', 'rejected'],
  sampled: ['result_ready'],
};

/**
 * /lab-requests — the home lab / screening request queue (B-N10-04). It shows no name, address, note or result: only
 * the reference, the masked account mobile, status, service and the preferred day. Statuses of the fake partner are
 * advanced here; real partners report their own.
 */
export function LabRequestsScreen() {
  const t = useTranslations('partnerLab.requests');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const list = useListParams({ status: 'all' });
  const status = list.params.filters.status ?? 'all';
  const query = labRequestsApi.useList({
    page: list.query.page,
    per_page: list.query.per_page,
    ...(list.params.q ? { q: list.params.q } : {}),
    ...(status === 'all' ? {} : { status }),
  });
  const set = useSetLabStatus();
  const move = (r: LabRequestRow, to: string) =>
    set.mutate({ id: r.id, status: to }, { onSuccess: () => toast.success(t('moved')), onError: notifyError });

  const columns: Column<LabRequestRow>[] = [
    {
      key: 'ref',
      header: t('reference'),
      cell: (r) => (
        <span dir="ltr" className="font-mono text-xs">
          {r.reference}
        </span>
      ),
    },
    { key: 'service', header: t('service'), cell: (r) => r.service },
    {
      key: 'user',
      header: t('user'),
      cell: (r) => (
        <Link href={`/users/${r.user_id}`} className="text-brand no-underline hover:underline" dir="ltr">
          {r.user_mobile ?? `#${r.user_id}`}
        </Link>
      ),
    },
    { key: 'day', header: t('preferred'), cell: (r) => `${r.preferred_date} · ${t(`window.${r.preferred_window}` as 'window.morning')}`, className: 'whitespace-nowrap' },
    { key: 'status', header: t('status'), cell: (r) => <Badge tone={TONE[r.status] ?? 'neutral'}>{t(`statusName.${r.status}` as 'statusName.requested')}</Badge> },
    {
      key: 'flags',
      header: t('flags'),
      cell: (r) => (
        <span className="flex flex-col items-start gap-1 text-xs text-muted">
          {r.consent_import ? <span>{t('importConsent')}</span> : null}
          {r.imported ? <span>{t('imported')}</span> : null}
          {r.purged ? <span>{t('purged')}</span> : null}
        </span>
      ),
    },
    { key: 'created', header: t('createdAt'), cell: (r) => (r.created_at ? formatDateTime(r.created_at, locale) : '—'), className: 'cell-num whitespace-nowrap' },
    {
      key: 'actions',
      header: <span className="sr-only">{t('actions')}</span>,
      cell: (r) =>
        r.manual_status && NEXT[r.status] ? (
          <div className="row-actions">
            {NEXT[r.status]!.map((to) => (
              <Button key={to} size="sm" variant={to === 'rejected' ? 'default' : 'primary'} loading={set.isPending && set.variables?.id === r.id} onClick={() => move(r, to)}>
                {t(`moveTo.${to}` as 'moveTo.accepted')}
              </Button>
            ))}
          </div>
        ) : null,
      className: 'cell-actions',
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t('title')}
        actions={
          <Link href="/lab-services" className="btn btn-sm">
            {t('catalog')}
          </Link>
        }
      />
      <Panel bodyClassName="">
        <div className="filter-bar">
          <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('searchPlaceholder')} />
          <Select
            aria-label={t('status')}
            value={status}
            onChange={(e) => list.setFilter('status', e.target.value)}
            options={[{ value: 'all', label: t('allStatuses') }, ...(query.data?.statuses ?? []).map((s) => ({ value: s, label: t(`statusName.${s}` as 'statusName.requested') }))]}
          />
        </div>
        <p className="panel-note">{t('privacyNote')}</p>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(r) => r.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
      <span className="sr-only">{formatNumber(query.data?.items.length ?? 0, locale)}</span>
    </div>
  );
}
