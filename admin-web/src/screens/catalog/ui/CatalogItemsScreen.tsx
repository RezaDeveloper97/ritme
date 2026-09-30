'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState, type DragEvent } from 'react';

import { useLocalized } from '@/shared/i18n';
import { cn, excerpt, useListParams, useNumber } from '@/shared/lib';
import {
  Badge,
  Button,
  ErrorState,
  Icon,
  PageHeader,
  Pagination,
  Panel,
  RowActions,
  SearchInput,
  Select,
  Skeleton,
  Switch,
  confirm,
  toast,
  useNotifyError,
} from '@/shared/ui';

import {
  useCatalogGroups,
  useCatalogItems,
  usePatchItem,
  useRemoveItem,
  useReorderItems,
  type CatalogItem,
} from '../api/catalog';
import { moveItem } from '../lib/payload';
import { MetaHint } from './MetaHint';

/** A group holds a short list; one page of the API's maximum keeps the whole list reorderable. */
const PER_PAGE = 100;

/** /catalog/:group — the ordered item list of one group (catalog.md §3). */
export function CatalogItemsScreen({ group }: { group: string }) {
  const t = useTranslations('catalog');
  const tc = useTranslations('crud');
  const router = useRouter();
  const list = useListParams({ status: 'all' });
  const query = useCatalogItems(group, { ...list.query, per_page: PER_PAGE });
  const groups = useCatalogGroups();
  const filtered = list.params.q !== '' || list.params.filters.status !== 'all';
  const complete = (query.data?.meta.last_page ?? 1) <= 1;

  const groupOptions = [...new Set([...(groups.data?.items ?? []).map((g) => g.group), group])]
    .sort()
    .map((g) => ({ value: g, label: g }));

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={<span dir="ltr">{group}</span>}
        backHref="/catalog"
        backLabel={t('backToGroups')}
        actions={
          <Link href={`/catalog/${group}/new`} className="btn btn-primary btn-sm">
            <Icon name="plus" size={15} />
            {t('newItem')}
          </Link>
        }
      />
      <MetaHint group={group} compact />
      <Panel bodyClassName="">
        <div className="flex flex-wrap items-end gap-3 border-b border-line p-4">
          <Select
            className="min-w-48"
            label={t('group')}
            value={group}
            dir="ltr"
            options={groupOptions}
            onChange={(e) => router.push(`/catalog/${e.target.value}`)}
          />
          <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('searchPlaceholder')} />
          <Select
            className="min-w-36"
            label={tc('status')}
            value={list.params.filters.status}
            options={[
              { value: 'all', label: t('statusAll') },
              { value: 'active', label: tc('active') },
              { value: 'inactive', label: tc('inactive') },
            ]}
            onChange={(e) => list.setFilter('status', e.target.value)}
          />
        </div>
        {query.error && !query.data ? (
          <ErrorState error={query.error} onRetry={() => query.refetch()} />
        ) : !query.data ? (
          <div className="flex flex-col gap-2 p-4">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : (
          <ItemsTable
            key={`${group}:${JSON.stringify(list.query)}`}
            group={group}
            rows={query.data.items}
            reorderable={!filtered && complete}
            emptyText={filtered ? t('emptyFiltered') : t('empty')}
          />
        )}
        {filtered ? <p className="field-hint px-4 pb-3">{t('reorderFiltered')}</p> : null}
        {complete ? null : <Pagination meta={query.data?.meta} onPage={list.setPage} />}
      </Panel>
    </div>
  );
}

function ItemsTable({
  group,
  rows: serverRows,
  reorderable,
  emptyText,
}: {
  group: string;
  rows: CatalogItem[];
  reorderable: boolean;
  emptyText: string;
}) {
  const t = useTranslations('catalog');
  const tc = useTranslations('crud');
  const n = useNumber();
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const reorder = useReorderItems(group);
  const patch = usePatchItem(group);
  const remove = useRemoveItem(group);
  const [order, setOrder] = useState<number[] | null>(null);
  const [pending, setPending] = useState<number | null>(null);
  const [dragFrom, setDragFrom] = useState<number | null>(null);

  const byId = new Map(serverRows.map((r) => [r.id, r]));
  const rows = order ? order.map((id) => byId.get(id)).filter((r): r is CatalogItem => Boolean(r)) : serverRows;
  const busy = reorder.isPending;

  const move = (from: number, to: number) => {
    if (!reorderable || busy || from === to || to < 0 || to >= rows.length) return;
    const next = moveItem(rows, from, to);
    setOrder(next.map((r) => r.id));
    reorder.mutate(next, {
      onSuccess: () => toast.success(t('reordered')),
      onError: notifyError,
      onSettled: () => setOrder(null),
    });
  };

  const toggle = (row: CatalogItem) => {
    setPending(row.id);
    patch.mutate(
      { row, patch: { is_active: !row.is_active } },
      { onSuccess: () => toast.success(tc('statusChanged')), onError: notifyError, onSettled: () => setPending(null) },
    );
  };

  const del = async (row: CatalogItem) => {
    if (!(await confirm({ message: t('confirmDelete'), confirmLabel: tc('delete'), tone: 'danger' }))) return;
    remove.mutate(row.id, { onSuccess: () => toast.success(tc('deleted')), onError: notifyError });
  };

  const onDrop = (e: DragEvent, to: number) => {
    e.preventDefault();
    if (dragFrom !== null) move(dragFrom, to);
    setDragFrom(null);
  };

  if (rows.length === 0) return <p className="cell-empty p-8 text-center text-muted">{emptyText}</p>;

  return (
    <div className="table-wrap">
      <table className="data-table">
        <caption className="sr-only">{t('itemsCaption', { group })}</caption>
        <thead>
          <tr>
            <th scope="col">{t('order')}</th>
            <th scope="col">{t('titleField')}</th>
            <th scope="col">{t('audiences')}</th>
            <th scope="col">{t('review')}</th>
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
              draggable={reorderable && !busy}
              onDragStart={() => setDragFrom(i)}
              onDragOver={(e) => e.preventDefault()}
              onDrop={(e) => onDrop(e, i)}
              onDragEnd={() => setDragFrom(null)}
              className={cn(dragFrom === i && 'opacity-50')}
              onClick={() => router.push(`/catalog/${group}/${r.id}`)}
            >
              <td className="cell-actions" onClick={(e) => e.stopPropagation()}>
                <RowActions>
                  {reorderable ? (
                    <>
                      <span className="cursor-grab text-muted" aria-hidden="true" title={t('dragHint')}>
                        ⋮⋮
                      </span>
                      <Button size="sm" icon variant="ghost" aria-label={t('moveUp')} disabled={i === 0 || busy} onClick={() => move(i, i - 1)}>
                        ↑
                      </Button>
                      <Button
                        size="sm"
                        icon
                        variant="ghost"
                        aria-label={t('moveDown')}
                        disabled={i === rows.length - 1 || busy}
                        onClick={() => move(i, i + 1)}
                      >
                        ↓
                      </Button>
                    </>
                  ) : null}
                  <span className="cell-num text-xs text-muted">{n(r.sort_order)}</span>
                </RowActions>
              </td>
              <td className="cell-wrap">
                <span className="flex flex-col">
                  <strong>{localize(r.title) || r.code}</strong>
                  {localize(r.body) ? <span className="text-xs text-muted">{excerpt(localize(r.body), 90)}</span> : null}
                  <span dir="ltr" className="text-xs text-muted">
                    {r.code}
                  </span>
                </span>
              </td>
              <td>
                {r.audiences.length ? (
                  <span className="flex flex-wrap gap-1" dir="ltr">
                    {r.audiences.map((a) => (
                      <Badge key={a} tone="data">
                        {a}
                      </Badge>
                    ))}
                  </span>
                ) : (
                  <span className="text-muted">{t('everyone')}</span>
                )}
              </td>
              <td>{r.needs_review ? <Badge tone="amber">{t('needsReview')}</Badge> : <Badge tone="green">{t('reviewed')}</Badge>}</td>
              <td onClick={(e) => e.stopPropagation()}>
                <Switch
                  label={r.is_active ? tc('active') : tc('inactive')}
                  checked={r.is_active}
                  disabled={pending === r.id || busy}
                  onChange={() => toggle(r)}
                />
              </td>
              <td className="cell-actions" onClick={(e) => e.stopPropagation()}>
                <RowActions>
                  <Link href={`/catalog/${group}/${r.id}`} className="btn btn-sm">
                    {tc('edit')}
                  </Link>
                  <Button
                    size="sm"
                    variant="danger"
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
      {reorderable ? <p className="field-hint px-4 pb-3">{t('reorderHint')}</p> : null}
    </div>
  );
}
