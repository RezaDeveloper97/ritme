'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState, type DragEvent } from 'react';

import { useLocalized } from '@/shared/i18n';
import { cn, formatNumber } from '@/shared/lib';
import {
  Badge,
  Button,
  ErrorState,
  Icon,
  Panel,
  RowActions,
  Skeleton,
  Switch,
  confirm,
  toast,
  useNotifyError,
} from '@/shared/ui';

import { checkupTypesApi, useCheckupStats, useReorder, useUpdateRow, type CheckupType } from '../api/checkup-types';
import { checkupIcon } from '../lib/icon';
import { moveItem, rowToBody } from '../lib/payload';
import { toneClass } from '../lib/tone';
import { useCheckupLabels } from './labels';

/** /checkup-types — the shared checkup catalog: order, activity and usage (admin-api.md §12). */
export function CheckupTypesScreen() {
  const t = useTranslations('checkupTypes');
  const query = checkupTypesApi.useList({ per_page: 100 });
  const stats = useCheckupStats();
  const byId = new Map((stats.data?.items ?? []).map((s) => [s.id, s]));

  return (
    <div className="flex flex-col gap-4">
      <Panel
        title={t('title')}
        bodyClassName=""
        actions={
          <Link href="/checkup-types/new" className="btn btn-primary btn-sm">
            <Icon name="plus" size={15} />
            {t('new')}
          </Link>
        }
      >
        <p className="panel-note">{t('intro')}</p>
        {query.error && !query.data ? (
          <ErrorState error={query.error} onRetry={() => query.refetch()} />
        ) : !query.data ? (
          <div className="flex flex-col gap-2 p-4">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : (
          <CatalogTable rows={query.data.items} stats={byId} />
        )}
      </Panel>
      <StatsPanel rows={query.data?.items ?? []} stats={stats} />
    </div>
  );
}

type StatsById = Map<number, NonNullable<ReturnType<typeof useCheckupStats>['data']>['items'][number]>;

function CatalogTable({ rows: serverRows, stats }: { rows: CheckupType[]; stats: StatsById }) {
  const t = useTranslations('checkupTypes');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const { label, interval, age } = useCheckupLabels();
  const notifyError = useNotifyError();
  const reorder = useReorder();
  const remove = checkupTypesApi.useRemove();
  const update = useUpdateRow();
  const [pending, setPending] = useState<number | null>(null);
  const [order, setOrder] = useState<number[] | null>(null);
  const [dragFrom, setDragFrom] = useState<number | null>(null);

  const byId = new Map(serverRows.map((r) => [r.id, r]));
  const rows = order ? order.map((id) => byId.get(id)).filter((r): r is CheckupType => Boolean(r)) : serverRows;
  const n = (v: number) => formatNumber(v, locale);

  const move = (from: number, to: number) => {
    if (from === to || to < 0 || to >= rows.length) return;
    const next = moveItem(rows.map((r) => r.id), from, to);
    setOrder(next);
    reorder.mutate(next, {
      onSuccess: () => toast.success(t('reordered')),
      onError: (e) => {
        setOrder(null);
        notifyError(e);
      },
      onSettled: () => setOrder(null),
    });
  };

  const toggle = (row: CheckupType) => {
    setPending(row.id);
    // No toggle endpoint: a full PUT (booleans are absent = false on the API).
    update.mutate(
      { id: row.id, body: { ...rowToBody(row), is_active: !row.is_active } },
      {
        onSuccess: () => toast.success(tc('statusChanged')),
        onError: notifyError,
        onSettled: () => setPending(null),
      },
    );
  };

  const del = async (row: CheckupType) => {
    if (!(await confirm({ message: tc('confirmDelete'), confirmLabel: tc('delete'), tone: 'danger' }))) return;
    remove.mutate(row.id, { onSuccess: () => toast.success(tc('deleted')), onError: notifyError });
  };

  const onDrop = (e: DragEvent, to: number) => {
    e.preventDefault();
    if (dragFrom !== null) move(dragFrom, to);
    setDragFrom(null);
  };

  if (rows.length === 0) return <p className="cell-empty p-8 text-center text-muted">{t('empty')}</p>;

  return (
    <div className="table-wrap">
      <table className="data-table">
        <caption className="sr-only">{t('title')}</caption>
        <thead>
          <tr>
            <th scope="col">
              <span className="sr-only">{t('order')}</span>
            </th>
            <th scope="col">{t('titleField')}</th>
            <th scope="col">{t('category')}</th>
            <th scope="col">{t('interval')}</th>
            <th scope="col">{t('ageWindow')}</th>
            <th scope="col">{t('recordsCount')}</th>
            <th scope="col">{tc('status')}</th>
            <th scope="col">
              <span className="sr-only">{tc('actions')}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => {
            const records = stats.get(r.id)?.records_total ?? r.records_count;
            return (
              <tr
                key={r.id}
                data-clickable="true"
                draggable={!reorder.isPending}
                onDragStart={() => setDragFrom(i)}
                onDragOver={(e) => e.preventDefault()}
                onDrop={(e) => onDrop(e, i)}
                onDragEnd={() => setDragFrom(null)}
                className={cn(dragFrom === i && 'opacity-50')}
                onClick={() => router.push(`/checkup-types/${r.id}`)}
              >
                <td className="cell-actions">
                  <RowActions>
                    <span className="cursor-grab text-muted" aria-hidden="true" title={t('dragHint')}>
                      ⋮⋮
                    </span>
                    <Button size="sm" icon variant="ghost" aria-label={t('moveUp')} disabled={i === 0 || reorder.isPending} onClick={() => move(i, i - 1)}>
                      ↑
                    </Button>
                    <Button
                      size="sm"
                      icon
                      variant="ghost"
                      aria-label={t('moveDown')}
                      disabled={i === rows.length - 1 || reorder.isPending}
                      onClick={() => move(i, i + 1)}
                    >
                      ↓
                    </Button>
                  </RowActions>
                </td>
                <td className="cell-wrap">
                  <span className="flex items-center gap-2.5">
                    <span className={cn('grid size-8 shrink-0 place-items-center rounded-lg', toneClass(r.tone))}>
                      <Icon name={checkupIcon(r.icon, r.performed_by)} size={16} />
                    </span>
                    <span className="flex flex-col">
                      <strong>{localize(r.title) || r.key}</strong>
                      <span dir="ltr" className="text-xs text-muted">
                        {r.key}
                      </span>
                    </span>
                  </span>
                </td>
                <td>
                  <Badge tone="brand">{label('category', r.category)}</Badge>
                </td>
                <td>{interval(r)}</td>
                <td>{age(r)}</td>
                <td className="cell-num">{n(records)}</td>
                <td onClick={(e) => e.stopPropagation()}>
                  <Switch
                    label={r.is_active ? tc('active') : tc('inactive')}
                    checked={r.is_active}
                    disabled={pending === r.id}
                    onChange={() => toggle(r)}
                  />
                </td>
                <td className="cell-actions">
                  <RowActions>
                    <Link href={`/checkup-types/${r.id}`} className="btn btn-sm">
                      {tc('edit')}
                    </Link>
                    <span title={records > 0 ? t('inUseHint', { count: records }) : undefined}>
                      <Button
                        size="sm"
                        variant="danger"
                        disabled={records > 0}
                        loading={remove.isPending && remove.variables === r.id}
                        onClick={() => del(r)}
                      >
                        {tc('delete')}
                      </Button>
                    </span>
                  </RowActions>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
      {rows.some((r) => (stats.get(r.id)?.records_total ?? r.records_count) > 0) ? (
        <p className="field-hint px-4 pb-3">{t('inUseNote')}</p>
      ) : null}
    </div>
  );
}

function StatsPanel({ rows, stats }: { rows: CheckupType[]; stats: ReturnType<typeof useCheckupStats> }) {
  const t = useTranslations('checkupTypes');
  const locale = useLocale();
  const localize = useLocalized();
  const n = (v: number) => formatNumber(v, locale);
  const items = stats.data?.items ?? [];
  const titleOf = new Map(rows.map((r) => [r.id, localize(r.title) || r.key]));
  const sum = (k: 'users_with_records' | 'records_last_30_days' | 'overdue_users') => items.reduce((a, s) => a + s[k], 0);

  return (
    <Panel title={t('stats')} bodyClassName="">
      {stats.error && !stats.data ? (
        <ErrorState error={stats.error} onRetry={() => stats.refetch()} />
      ) : !stats.data ? (
        <div className="p-4">
          <Skeleton className="h-24 w-full" />
        </div>
      ) : (
        <>
          <div className="grid gap-3 p-4 sm:grid-cols-3">
            {(
              [
                ['users_with_records', t('usersWithRecords')],
                ['records_last_30_days', t('recordsLast30', { days: stats.data.window_days })],
                ['overdue_users', t('overdueUsers')],
              ] as const
            ).map(([k, text]) => (
              <div key={k} className="rounded-xl border border-line p-3">
                <div className="text-xs text-muted">{text}</div>
                <div className="text-xl font-extrabold">{n(sum(k))}</div>
              </div>
            ))}
          </div>
          <div className="table-wrap">
            <table className="data-table">
              <caption className="sr-only">{t('stats')}</caption>
              <thead>
                <tr>
                  <th scope="col">{t('titleField')}</th>
                  <th scope="col">{t('usersWithRecords')}</th>
                  <th scope="col">{t('recordsLast30', { days: stats.data.window_days })}</th>
                  <th scope="col">{t('overdueUsers')}</th>
                </tr>
              </thead>
              <tbody>
                {items.map((s) => (
                  <tr key={s.id}>
                    <td>{titleOf.get(s.id) ?? `#${s.id}`}</td>
                    <td className="cell-num">{n(s.users_with_records)}</td>
                    <td className="cell-num">{n(s.records_last_30_days)}</td>
                    <td className="cell-num">
                      {s.overdue_users > 0 ? <Badge tone="amber">{n(s.overdue_users)}</Badge> : n(0)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="field-hint px-4 pb-3">{t('overdueNote')}</p>
        </>
      )}
    </Panel>
  );
}
