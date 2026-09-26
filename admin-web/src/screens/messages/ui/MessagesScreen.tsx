'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { cn, excerpt, formatNumber, useListParams } from '@/shared/lib';
import { Badge, Button, DataTable, Pagination, Panel, RowActions, Select, toast, useNotifyError, type Column } from '@/shared/ui';

import { messagesApi, type Message, type MissingMessage } from '../api/messages';
import { previewOf } from '../lib/payload';
import { useMessageLabels } from './labels';
import { MessageCreatePanel, type CreateTarget } from './MessageCreatePanel';

/** /messages — group / locale / approval filters, approve and toggle in place (Blade messages.index). */
export function MessagesScreen() {
  const t = useTranslations('smartMessages');
  const tc = useTranslations('crud');
  const router = useRouter();
  const labels = useMessageLabels();
  const notifyError = useNotifyError();
  const list = useListParams({ group: '', locale: '', status: '' });
  const query = messagesApi.useList(list.query);
  const approve = messagesApi.useAction('approve');
  const toggle = messagesApi.useAction('toggle');
  const [creating, setCreating] = useState<CreateTarget | null>(null);
  const busy = (m: { isPending: boolean; variables?: { id: number } }, id: number) => m.isPending && m.variables?.id === id;

  const columns: Column<Message>[] = [
    {
      key: 'group',
      header: t('group'),
      cell: (m) => labels.group(m.group),
      className: 'cell-wrap',
    },
    {
      key: 'key',
      header: t('key'),
      cell: (m) => <span dir="ltr">{m.item_key}</span>,
      className: 'cell-mono',
    },
    {
      key: 'locale',
      header: t('locale'),
      cell: (m) => labels.locale(m.locale),
      className: 'whitespace-nowrap',
    },
    {
      key: 'preview',
      header: t('preview'),
      cell: (m) => (
        <span dir={labels.direction(m.locale)} className="text-ink-3">
          {excerpt(previewOf(m.payload, t('listJoiner')), 80) || '—'}
        </span>
      ),
      className: 'cell-wrap',
    },
    {
      key: 'status',
      header: tc('status'),
      cell: (m) => (
        <span className="flex flex-wrap gap-1">
          {m.is_approved ? <Badge tone="green">{t('approved')}</Badge> : <Badge tone="amber">{t('pending')}</Badge>}
          {m.is_active ? null : <Badge tone="red">{tc('inactive')}</Badge>}
        </span>
      ),
    },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (m) => (
        <RowActions>
          <Link href={`/messages/${m.id}`} className="btn btn-sm">
            {tc('edit')}
          </Link>
          <Button
            size="sm"
            loading={busy(approve, m.id)}
            onClick={() =>
              approve.mutate(
                { id: m.id },
                {
                  onSuccess: () => toast.success(m.is_approved ? t('unapprovedToast') : t('approvedToast')),
                  onError: notifyError,
                },
              )
            }
          >
            {m.is_approved ? t('unapprove') : t('approve')}
          </Button>
          <Button
            size="sm"
            loading={busy(toggle, m.id)}
            onClick={() =>
              toggle.mutate(
                { id: m.id },
                {
                  onSuccess: () => toast.success(tc('statusChanged')),
                  onError: notifyError,
                },
              )
            }
          >
            {m.is_active ? tc('deactivate') : tc('activate')}
          </Button>
        </RowActions>
      ),
      className: 'cell-actions',
    },
  ];

  const groups = query.data?.groups ?? [];
  const locales = query.data?.locales ?? [];
  const filtered = Boolean(list.params.filters.group || list.params.filters.locale || list.params.filters.status);

  return (
    <div className="flex flex-col gap-4">
      {creating ? <MessageCreatePanel target={creating} onClose={() => setCreating(null)} /> : null}
      {query.data ? (
        <MissingPanel
          group={list.params.filters.group ?? ''}
          missing={query.data.missing}
          registered={query.data.registered_groups}
          onCreate={setCreating}
          onGroup={(g) => list.setFilter('group', g)}
        />
      ) : null}
      <Panel title={t('title')} bodyClassName="">
        <div className="filter-bar">
          <Select
            label={t('group')}
            className="min-w-48 flex-1 sm:max-w-72"
            value={list.params.filters.group}
            onChange={(e) => list.setFilter('group', e.target.value)}
            options={[
              { value: '', label: t('allGroups') },
              ...[...new Set([...groups, ...(query.data?.registered_groups ?? [])])].map((g) => ({ value: g, label: labels.group(g) })),
            ]}
          />
          <Select
            label={t('locale')}
            className="w-40"
            value={list.params.filters.locale}
            onChange={(e) => list.setFilter('locale', e.target.value)}
            options={[{ value: '', label: t('allLocales') }, ...locales.map((l) => ({ value: l, label: labels.locale(l) }))]}
          />
          <Select
            label={tc('status')}
            className="w-44"
            value={list.params.filters.status}
            onChange={(e) => list.setFilter('status', e.target.value)}
            options={[
              { value: '', label: t('allStatuses') },
              { value: 'approved', label: t('approved') },
              { value: 'pending', label: t('pendingLong') },
            ]}
          />
        </div>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(m) => m.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          onRowClick={(m) => router.push(`/messages/${m.id}`)}
          emptyText={filtered ? t('emptyFiltered') : t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}

const WEEK_TIP_GROUP = 'pregnancy_week_tip';
const MAX_MISSING_ROWS = 60;

/**
 * Registered rows (admin-api.md §13) that have no message yet in some language, with «ایجاد».
 * Without a group filter: a count per group; the week tips get a 1–42 filled / missing grid.
 */
function MissingPanel({
  group,
  missing,
  registered,
  onCreate,
  onGroup,
}: {
  group: string;
  missing: MissingMessage[];
  registered: string[];
  onCreate: (target: CreateTarget) => void;
  onGroup: (group: string) => void;
}) {
  const t = useTranslations('smartMessages');
  const labels = useMessageLabels();
  if (group && !registered.includes(group)) return null;

  if (!group) {
    const counts = new Map<string, number>();
    for (const m of missing) counts.set(m.group, (counts.get(m.group) ?? 0) + 1);
    if (counts.size === 0) return null;
    return (
      <Panel title={t('missingTitle')}>
        <p className="field-hint mt-0">{t('missingHint')}</p>
        <div className="flex flex-wrap gap-2">
          {[...counts].map(([g, n]) => (
            <Button key={g} size="sm" onClick={() => onGroup(g)}>
              {labels.group(g)} <Badge tone="amber">{n}</Badge>
            </Button>
          ))}
        </div>
      </Panel>
    );
  }

  return (
    <Panel title={t('missingTitle')}>
      {group === WEEK_TIP_GROUP ? <WeekTipGrid missing={missing} onCreate={onCreate} /> : null}
      {missing.length === 0 ? (
        <p className="m-0 text-ink-3">{t('noMissing')}</p>
      ) : group === WEEK_TIP_GROUP ? null : (
        <ul className="m-0 flex list-none flex-col gap-1.5 p-0">
          {missing.slice(0, MAX_MISSING_ROWS).map((m) => (
            <li key={`${m.item_key}/${m.locale}`} className="flex flex-wrap items-center gap-2">
              <span dir="ltr" className="cell-mono">
                {m.item_key}
              </span>
              <Badge>{labels.locale(m.locale)}</Badge>
              <Button size="sm" onClick={() => onCreate(m)}>
                {t('create')}
              </Button>
            </li>
          ))}
          {missing.length > MAX_MISSING_ROWS ? (
            <li className="text-muted">{t('moreMissing', { count: missing.length - MAX_MISSING_ROWS })}</li>
          ) : null}
        </ul>
      )}
    </Panel>
  );
}

function WeekTipGrid({ missing, onCreate }: { missing: MissingMessage[]; onCreate: (target: CreateTarget) => void }) {
  const t = useTranslations('smartMessages');
  const labels = useMessageLabels();
  const uiLocale = useLocale();
  const byWeek = new Map<string, MissingMessage[]>();
  for (const m of missing) byWeek.set(m.item_key, [...(byWeek.get(m.item_key) ?? []), m]);
  const weeks = Array.from({ length: 42 }, (_, i) => String(i + 1));
  return (
    <div className="mb-4 flex flex-col gap-2">
      <p className="field-hint m-0">{t('weekTipGridHint')}</p>
      <div className="grid grid-cols-6 gap-1.5 sm:grid-cols-7 lg:grid-cols-14">
        {weeks.map((w) => {
          const gaps = byWeek.get(w) ?? [];
          const first = gaps[0];
          const title = first
            ? t('weekMissing', {
                week: w,
                locales: gaps.map((g) => labels.locale(g.locale)).join('، '),
              })
            : t('weekFilled', { week: w });
          return (
            <button
              key={w}
              type="button"
              title={title}
              aria-label={title}
              disabled={!first}
              onClick={() => first && onCreate(first)}
              className={cn(
                'grid h-10 place-items-center rounded-lg border text-sm font-semibold',
                first
                  ? 'cursor-pointer border-[var(--amber-deep)] bg-[var(--amber-soft)] text-[var(--amber-deep)]'
                  : 'border-transparent bg-[var(--green-soft)] text-[var(--green-deep)]',
              )}
            >
              {formatNumber(Number(w), uiLocale)}
            </button>
          );
        })}
      </div>
    </div>
  );
}
