'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { excerpt, useListParams } from '@/shared/lib';
import {
  Badge,
  Button,
  DataTable,
  Pagination,
  Panel,
  RowActions,
  Select,
  toast,
  useNotifyError,
  type Column,
} from '@/shared/ui';

import { messagesApi, type Message } from '../api/messages';
import { previewOf } from '../lib/payload';
import { useMessageLabels } from './labels';

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
  const busy = (m: { isPending: boolean; variables?: { id: number } }, id: number) => m.isPending && m.variables?.id === id;

  const columns: Column<Message>[] = [
    { key: 'group', header: t('group'), cell: (m) => labels.group(m.group), className: 'cell-wrap' },
    { key: 'key', header: t('key'), cell: (m) => <span dir="ltr">{m.item_key}</span>, className: 'cell-mono' },
    { key: 'locale', header: t('locale'), cell: (m) => labels.locale(m.locale), className: 'whitespace-nowrap' },
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
                { onSuccess: () => toast.success(m.is_approved ? t('unapprovedToast') : t('approvedToast')), onError: notifyError },
              )
            }
          >
            {m.is_approved ? t('unapprove') : t('approve')}
          </Button>
          <Button
            size="sm"
            loading={busy(toggle, m.id)}
            onClick={() => toggle.mutate({ id: m.id }, { onSuccess: () => toast.success(tc('statusChanged')), onError: notifyError })}
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
    <Panel title={t('title')} bodyClassName="">
      <div className="filter-bar">
        <Select
          label={t('group')}
          className="min-w-48 flex-1 sm:max-w-72"
          value={list.params.filters.group}
          onChange={(e) => list.setFilter('group', e.target.value)}
          options={[{ value: '', label: t('allGroups') }, ...groups.map((g) => ({ value: g, label: labels.group(g) }))]}
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
  );
}
