'use client';

import { useLocale, useTranslations } from 'next-intl';

import { formatDate, formatNumber, useListParams } from '@/shared/lib';
import { Badge, DataTable, Pagination, Panel, Select, Skeleton, type BadgeTone, type Column } from '@/shared/ui';

import { companionLinksApi, type CompanionLink, type StatusCounts } from '../api/companions';

const STATUSES = ['all', 'invited', 'active', 'revoked'] as const;
const TYPES = ['all', 'partner', 'spouse'] as const;
const STATUS_TONE: Record<string, BadgeTone> = { active: 'green', invited: 'brand', revoked: 'neutral' };

function StatusBadge({ status }: { status: string }) {
  const t = useTranslations('companions.links.status');
  return <Badge tone={STATUS_TONE[status] ?? 'neutral'}>{t.has(status as 'active') ? t(status as 'active') : status}</Badge>;
}

function Person({ person }: { person: { name: string | null; mobile: string | null } | null }) {
  const t = useTranslations('companions.links');
  if (!person) return <span className="text-muted">{t('notJoined')}</span>;
  return (
    <span className="flex flex-col items-start">
      <span className="font-semibold" dir="auto">
        {person.name ?? '•••'}
      </span>
      {person.mobile ? (
        <span dir="ltr" className="text-xs text-muted">
          {person.mobile}
        </span>
      ) : null}
    </span>
  );
}

/** /companions/links — masked, read-only overview of companion links with counts (B-N4-07). */
export function CompanionLinksScreen() {
  const t = useTranslations('companions.links');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const list = useListParams({ status: 'all', type: 'all' });
  const query = companionLinksApi.useList(list.query);
  const counts = query.data?.counts;
  const n = (v: number) => formatNumber(v, locale);
  const filtered = list.params.filters.status !== 'all' || list.params.filters.type !== 'all';

  const columns: Column<CompanionLink>[] = [
    { key: 'id', header: '#', cell: (l) => n(l.id), className: 'cell-num w-14 text-muted' },
    { key: 'owner', header: t('owner'), cell: (l) => <Person person={l.owner} /> },
    {
      key: 'companion',
      header: t('companion'),
      cell: (l) => (
        <span className="flex flex-col items-start gap-0.5">
          <Person person={l.companion} />
          {l.label ? <span className="text-xs text-ink-3">{t('labelAs', { label: l.label })}</span> : null}
        </span>
      ),
    },
    { key: 'type', header: t('type'), cell: (l) => t(`types.${l.type === 'spouse' ? 'spouse' : 'partner'}`) },
    {
      key: 'status',
      header: tc('status'),
      cell: (l) => (
        <span className="flex flex-col items-start gap-1">
          <StatusBadge status={l.status} />
          {l.invite ? (
            <span className="text-xs text-muted">
              {l.invite.expired ? t('inviteExpired') : t('inviteUntil', { date: formatDate(l.invite.expires_at, locale) })}
              {l.invite.phone ? (
                <>
                  {' · '}
                  <span dir="ltr">{l.invite.phone}</span>
                </>
              ) : null}
            </span>
          ) : null}
          {l.status === 'revoked' && l.revoked_by ? (
            <span className="text-xs text-muted">{t(`revokedBy.${l.revoked_by === 'companion' ? 'companion' : 'owner'}`)}</span>
          ) : null}
        </span>
      ),
    },
    {
      key: 'grants',
      header: t('grants'),
      cell: (l) => (l.status === 'active' ? t('grantsCount', { count: l.grants_count }) : '—'),
      className: 'whitespace-nowrap text-ink-3',
    },
    {
      key: 'dates',
      header: t('dates'),
      cell: (l) => (
        <span className="flex flex-col text-xs tabular-nums text-ink-3">
          <span>{t('invitedOn', { date: formatDate(l.invited_at ?? l.created_at, locale) })}</span>
          {l.accepted_at ? <span>{t('acceptedOn', { date: formatDate(l.accepted_at, locale) })}</span> : null}
          {l.revoked_at ? <span>{t('revokedOn', { date: formatDate(l.revoked_at, locale) })}</span> : null}
        </span>
      ),
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <Counts counts={counts?.by_status} byType={counts?.by_type} format={n} />
      <Panel title={t('title')} bodyClassName="">
        <div className="filter-bar">
          <Select
            label={tc('status')}
            className="w-44"
            value={list.params.filters.status}
            onChange={(e) => list.setFilter('status', e.target.value)}
            options={STATUSES.map((s) => ({
              value: s,
              label: counts ? `${t(`filter.${s}`)} (${n(counts.by_status[s])})` : t(`filter.${s}`),
            }))}
          />
          <Select
            label={t('type')}
            className="w-40"
            value={list.params.filters.type}
            onChange={(e) => list.setFilter('type', e.target.value)}
            options={TYPES.map((v) => ({ value: v, label: v === 'all' ? t('allTypes') : t(`types.${v}`) }))}
          />
        </div>
        <p className="panel-note">{t('maskNote')}</p>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(l) => l.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={filtered ? t('emptyFiltered') : t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}

function Counts({
  counts,
  byType,
  format,
}: {
  counts: StatusCounts | undefined;
  byType: Record<string, StatusCounts> | undefined;
  format: (v: number) => string;
}) {
  const t = useTranslations('companions.links');
  const cells = (['all', 'active', 'invited', 'revoked'] as const).map((s) => ({
    key: s,
    label: t(`count.${s}`),
    value: counts?.[s],
    sub:
      byType && s !== 'all'
        ? t('countSub', { partner: format(byType.partner?.[s] ?? 0), spouse: format(byType.spouse?.[s] ?? 0) })
        : byType
          ? t('countSub', { partner: format(byType.partner?.all ?? 0), spouse: format(byType.spouse?.all ?? 0) })
          : ' ',
  }));
  return (
    <dl className="ledger m-0">
      {cells.map((c) => (
        <div key={c.key} className="ledger-cell">
          <dt className="text-[13px] font-semibold text-ink-3">{c.label}</dt>
          <dd className="m-0 mt-1">
            {c.value === undefined ? <Skeleton className="h-8 w-20" /> : <span className="ledger-value">{format(c.value)}</span>}
            <span className="mt-0.5 block text-xs text-muted">{c.sub}</span>
          </dd>
        </div>
      ))}
    </dl>
  );
}
