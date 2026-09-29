'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState, type DragEvent } from 'react';

import { isApiError } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { cn, formatNumber } from '@/shared/lib';
import { Badge, Button, ErrorState, Icon, Panel, RowActions, Skeleton, Switch, confirm, toast, useNotifyError } from '@/shared/ui';

import { careItemsApi, useReorderCareItems, type CareItem } from '../api/care-items';
import { moveItem } from '../lib/order';
import { useKindLabel } from './labels';

/** /pregnancy-care-plan — the visits / tests / scans / vaccines of the pregnancy care plan (admin-api.md §13). */
export function CarePlanScreen() {
  const t = useTranslations('pregnancyCarePlan');
  const query = careItemsApi.useList();
  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        <Link href="/pregnancy-care-plan/new" className="btn btn-primary btn-sm">
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
        <CareTable rows={query.data.items} />
      )}
    </Panel>
  );
}

function CareTable({ rows: serverRows }: { rows: CareItem[] }) {
  const t = useTranslations('pregnancyCarePlan');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const kindLabel = useKindLabel();
  const notifyError = useNotifyError();
  const reorder = useReorderCareItems();
  const toggle = careItemsApi.useAction('toggle');
  const remove = careItemsApi.useRemove();
  const [order, setOrder] = useState<number[] | null>(null);
  const [dragFrom, setDragFrom] = useState<number | null>(null);

  const byId = new Map(serverRows.map((r) => [r.id, r]));
  const rows = order ? order.map((id) => byId.get(id)).filter((r): r is CareItem => Boolean(r)) : serverRows;
  const n = (v: number) => formatNumber(v, locale);

  const move = (from: number, to: number) => {
    if (from === to || to < 0 || to >= rows.length) return;
    const next = moveItem(
      rows.map((r) => r.id),
      from,
      to,
    );
    setOrder(next);
    reorder.mutate(next, {
      onSuccess: () => toast.success(t('reordered')),
      onError: notifyError,
      onSettled: () => setOrder(null),
    });
  };

  const flip = (row: CareItem) =>
    toggle.mutate(
      { id: row.id },
      {
        onSuccess: () => toast.success(tc('statusChanged')),
        onError: notifyError,
      },
    );

  const del = async (row: CareItem) => {
    if (
      !(await confirm({
        message: tc('confirmDelete'),
        confirmLabel: tc('delete'),
        tone: 'danger',
      }))
    )
      return;
    remove.mutate(row.id, {
      onSuccess: () => toast.success(tc('deleted')),
      onError: async (e) => {
        // Appointments reference the key: offer deactivating instead (admin-api.md §13).
        if (isApiError(e) && e.code === 'in_use') {
          if (
            row.is_active &&
            (await confirm({
              message: t('inUseDeactivate'),
              confirmLabel: tc('deactivate'),
            }))
          )
            flip(row);
          return;
        }
        notifyError(e);
      },
    });
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
            <th scope="col">{t('kind')}</th>
            <th scope="col">{t('window')}</th>
            <th scope="col">{t('remindBefore')}</th>
            <th scope="col">{t('appointments')}</th>
            <th scope="col">{tc('status')}</th>
            <th scope="col">
              <span className="sr-only">{tc('actions')}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr
              key={r.id}
              data-clickable="true"
              draggable={!reorder.isPending}
              onDragStart={() => setDragFrom(i)}
              onDragOver={(e) => e.preventDefault()}
              onDrop={(e) => onDrop(e, i)}
              onDragEnd={() => setDragFrom(null)}
              className={cn(dragFrom === i && 'opacity-50')}
              onClick={() => router.push(`/pregnancy-care-plan/${r.id}`)}
            >
              <td className="cell-actions" onClick={(e) => e.stopPropagation()}>
                <RowActions>
                  <span className="cursor-grab text-muted" aria-hidden="true" title={t('dragHint')}>
                    ⋮⋮
                  </span>
                  <Button
                    size="sm"
                    icon
                    variant="ghost"
                    aria-label={t('moveUp')}
                    disabled={i === 0 || reorder.isPending}
                    onClick={() => move(i, i - 1)}
                  >
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
                <span className="flex flex-col">
                  <strong>{localize(r.title) || r.key}</strong>
                  <span dir="ltr" className="text-xs text-muted">
                    {r.key}
                  </span>
                </span>
              </td>
              <td>
                <Badge tone="brand">{kindLabel(r.kind)}</Badge>
              </td>
              <td>{t('weeks', { from: n(r.week_from), to: n(r.week_to) })}</td>
              <td>{t('days', { count: r.remind_before, n: n(r.remind_before) })}</td>
              <td className="cell-num">{n(r.appointments_count)}</td>
              <td onClick={(e) => e.stopPropagation()}>
                <Switch
                  label={r.is_active ? tc('active') : tc('inactive')}
                  checked={r.is_active}
                  disabled={toggle.isPending && toggle.variables?.id === r.id}
                  onChange={() => flip(r)}
                />
              </td>
              <td className="cell-actions" onClick={(e) => e.stopPropagation()}>
                <RowActions>
                  <Link href={`/pregnancy-care-plan/${r.id}`} className="btn btn-sm">
                    {tc('edit')}
                  </Link>
                  <Button
                    size="sm"
                    variant="danger"
                    title={r.appointments_count > 0 ? t('inUseHint', { count: n(r.appointments_count) }) : undefined}
                    loading={remove.isPending && remove.variables === r.id}
                    onClick={() => del(r)}
                  >
                    {tc('delete')}
                  </Button>
                </RowActions>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
