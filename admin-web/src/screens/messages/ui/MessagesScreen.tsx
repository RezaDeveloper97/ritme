'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { useCurrentAdmin } from '@/features/auth';
import { cn, excerpt, formatNumber, useListParams, useNumber } from '@/shared/lib';
import { Badge, Button, DataTable, Pagination, Panel, RowActions, Select, toast, useNotifyError, type Column } from '@/shared/ui';

import { messagesApi, type Message, type MissingMessage } from '../api/messages';
import { previewOf } from '../lib/payload';
import { groupByArea } from '../lib/where';
import { SUPER_ONLY_GROUPS, canWriteGroup } from '../lib/super-only';
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
  const role = useCurrentAdmin()?.role;
  const superOnly = query.data?.super_only_groups ?? SUPER_ONLY_GROUPS;
  // Clinical groups (postpartum copy) are super-admin writes (admin-api.md §18): editors only read them.
  const canWrite = (group: string) => canWriteGroup(group, role, superOnly);
  const busy = (m: { isPending: boolean; variables?: { id: number } }, id: number) => m.isPending && m.variables?.id === id;

  const columns: Column<Message>[] = [
    {
      key: 'group',
      header: t('group'),
      cell: (m) => <GroupCell group={m.group} />,
      className: 'cell-wrap',
    },
    {
      key: 'key',
      header: t('item'),
      cell: (m) => <ItemCell group={m.group} itemKey={m.item_key} />,
      className: 'cell-wrap',
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
            {canWrite(m.group) ? tc('edit') : t('view')}
          </Link>
          {canWrite(m.group) ? (
            <>
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
            </>
          ) : null}
        </RowActions>
      ),
      className: 'cell-actions',
    },
  ];

  const groups = query.data?.groups ?? [];
  const allGroups = [...new Set([...groups, ...(query.data?.registered_groups ?? [])])];
  const selectedGroup = list.params.filters.group ?? '';
  const locales = query.data?.locales ?? [];
  const filtered = Boolean(list.params.filters.group || list.params.filters.locale || list.params.filters.status);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex justify-end">
        <Link href="/messages/rules" className="btn btn-sm">
          {t('rulesLink')}
        </Link>
      </div>
      {selectedGroup ? (
        <WherePanel group={selectedGroup} />
      ) : (
        <WhereGuide groups={allGroups} onGroup={(g) => list.setFilter('group', g)} />
      )}
      {creating ? <MessageCreatePanel target={creating} onClose={() => setCreating(null)} /> : null}
      {query.data ? (
        <MissingPanel
          group={list.params.filters.group ?? ''}
          missing={query.data.missing}
          registered={query.data.registered_groups}
          onCreate={canWrite(list.params.filters.group ?? '') ? setCreating : undefined}
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
              // Ordered by where the app shows them, «area › group» so the same-named groups stay apart.
              ...groupByArea(allGroups).flatMap((b) => b.groups.map((g) => ({ value: g, label: `${labels.area(g)} › ${labels.group(g)}` }))),
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

/** The group's name, then where the app shows it. */
function GroupCell({ group }: { group: string }) {
  const t = useTranslations('smartMessages');
  const labels = useMessageLabels();
  const where = labels.where(group);
  return (
    <span className="flex flex-col gap-0.5">
      <span className="font-medium">{labels.group(group)}</span>
      {where ? <span className="text-xs text-muted">{where}</span> : null}
      {labels.shown(group) ? null : (
        <span>
          <Badge tone="neutral">{t('notShown')}</Badge>
        </span>
      )}
    </span>
  );
}

/** The readable name of the item (weeks, phases, symptoms…), with the stored key under it. */
function ItemCell({ group, itemKey }: { group: string; itemKey: string }) {
  const labels = useMessageLabels();
  const name = labels.item(group, itemKey);
  return (
    <span className="flex flex-col gap-0.5">
      {name ? <span>{name}</span> : null}
      <span dir="ltr" className={cn('cell-mono', name && 'text-xs text-muted')}>
        {itemKey}
      </span>
    </span>
  );
}

/** With a group picked: one line saying where its messages appear. */
function WherePanel({ group }: { group: string }) {
  const t = useTranslations('smartMessages');
  const labels = useMessageLabels();
  const where = labels.where(group);
  if (!where) return null;
  return (
    <Panel title={t('whereTitle')}>
      <p className="m-0 flex flex-wrap items-center gap-2">
        <Badge tone={labels.shown(group) ? 'brand' : 'neutral'}>{labels.area(group)}</Badge>
        <span>{where}</span>
      </p>
    </Panel>
  );
}

/** Without a filter: every group, bucketed by the part of the app it belongs to; a click filters the list. */
function WhereGuide({ groups, onGroup }: { groups: string[]; onGroup: (group: string) => void }) {
  const t = useTranslations('smartMessages');
  const labels = useMessageLabels();
  const n = useNumber();
  if (groups.length === 0) return null;
  return (
    <Panel title={t('guideTitle')}>
      <p className="field-hint mt-0">{t('guideHint')}</p>
      <p className="field-hint mt-0">{t('rulesNote')}</p>
      <div className="flex flex-col gap-2">
        {groupByArea(groups).map((bucket) => (
          <details key={bucket.area ?? 'other'} className="where-area">
            <summary className="where-area-summary">
              <span>{t(`areas.${bucket.area ?? 'other'}` as 'areas.other')}</span>
              <Badge tone={bucket.area === 'notShown' ? 'neutral' : 'brand'}>{n(bucket.groups.length)}</Badge>
            </summary>
            <ul className="m-0 flex list-none flex-col gap-1 p-0 pt-2">
              {bucket.groups.map((g) => (
                <li key={g}>
                  <button type="button" className="where-row" onClick={() => onGroup(g)}>
                    <span className="font-semibold">{labels.group(g)}</span>
                    <span className="text-xs text-muted">{labels.where(g) ?? g}</span>
                  </button>
                </li>
              ))}
            </ul>
          </details>
        ))}
      </div>
    </Panel>
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
  /** Absent: the viewer may not create rows of this group (read-only listing). */
  onCreate?: (target: CreateTarget) => void;
  onGroup: (group: string) => void;
}) {
  const t = useTranslations('smartMessages');
  const n = useNumber();
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
      {group === WEEK_TIP_GROUP && onCreate ? <WeekTipGrid missing={missing} onCreate={onCreate} /> : null}
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
              {onCreate ? (
                <Button size="sm" onClick={() => onCreate(m)}>
                  {t('create')}
                </Button>
              ) : null}
            </li>
          ))}
          {missing.length > MAX_MISSING_ROWS ? (
            <li className="text-muted">{t('moreMissing', { count: n(missing.length - MAX_MISSING_ROWS) })}</li>
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
                week: formatNumber(Number(w), uiLocale),
                locales: gaps.map((g) => labels.locale(g.locale)).join('، '),
              })
            : t('weekFilled', { week: formatNumber(Number(w), uiLocale) });
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
