'use client';

import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';
import { z } from 'zod';

import { api, listSchema } from '@/shared/api';
import { useErrorMessage } from '@/shared/i18n';
import { formatDateTime, formatNumber, useListParams } from '@/shared/lib';
import {
  Badge,
  Button,
  confirm,
  DataTable,
  ImageUpload,
  Pagination,
  Panel,
  SearchInput,
  Select,
  Switch,
  TextInput,
  toast,
  TranslatableField,
  type Column,
  type Translations,
} from '@/shared/ui';

/**
 * Living catalogue of the shared building blocks, wired the way T-M2-23 screens
 * should use them. Dev-only in the sidebar; reachable at /ui-kit.
 */
export function UiKitScreen() {
  const t = useTranslations('uiKit');
  const tr = useTranslations('roles');
  const [title, setTitle] = useState<Translations>({});
  const [body, setBody] = useState<Translations>({});
  const [image, setImage] = useState<File | null>(null);
  const [name, setName] = useState('');
  const [role, setRole] = useState('editor');
  const [enabled, setEnabled] = useState(true);

  return (
    <div className="flex flex-col gap-5">
      <p className="m-0 text-ink-3">{t('intro')}</p>

      <Panel title={t('translatable')}>
        <div className="flex flex-col gap-5">
          <TranslatableField name="title" label={t('titleLabel')} value={title} onChange={setTitle} required maxLength={255} />
          <TranslatableField name="body" label={t('bodyLabel')} value={body} onChange={setBody} kind="rich" required />
          <details className="text-ink-3">
            <summary className="cursor-pointer text-[13px] font-semibold">{t('richOutput')}</summary>
            <pre className="mt-2 overflow-x-auto rounded-[10px] bg-surface-2 p-3 text-xs" dir="ltr">
              {JSON.stringify({ title, body }, null, 2)}
            </pre>
          </details>
        </div>
      </Panel>

      <div className="grid gap-5 lg:grid-cols-2">
        <Panel title={t('upload')}>
          <ImageUpload label={t('upload')} value={image} onChange={setImage} />
        </Panel>
        <Panel title={t('fields')}>
          <div className="flex flex-col gap-4">
            <TextInput label={t('name')} value={name} onChange={(e) => setName(e.target.value)} required />
            <Select
              label={t('role')}
              value={role}
              onChange={(e) => setRole(e.target.value)}
              options={[
                { value: 'editor', label: tr('editor') },
                { value: 'super', label: tr('super') },
              ]}
            />
            <Switch label={t('enabled')} checked={enabled} onChange={setEnabled} />
          </div>
        </Panel>
      </div>

      <Panel title={t('feedback')}>
        <div className="flex flex-wrap gap-2">
          <Button variant="primary" onClick={() => toast.success(t('saved'))}>
            {t('toastSuccess')}
          </Button>
          <Button onClick={() => toast.error(t('failed'))}>{t('toastError')}</Button>
          <Button
            variant="danger"
            onClick={async () => {
              if (await confirm({ message: t('confirmDelete'), tone: 'danger' })) toast.success(t('deleted'));
            }}
          >
            {t('askDelete')}
          </Button>
        </div>
      </Panel>

      <ServerListSample />
    </div>
  );
}

// ── Server list pattern: URL state → query key → zod list schema → table + pager ──

const userRowSchema = z.object({
  id: z.number(),
  name: z.string().nullable(),
  mobile: z.string().nullable(),
  is_blocked: z.boolean(),
  created_at: z.string().nullable(),
});
type UserRow = z.infer<typeof userRowSchema>;
const usersListSchema = listSchema(userRowSchema, z.object({ q: z.string(), status: z.string() }).partial());

function ServerListSample() {
  const t = useTranslations('uiKit');
  const tc = useTranslations('common');
  const locale = useLocale();
  const describe = useErrorMessage();
  const list = useListParams({ status: 'all' });
  const query = useQuery({
    queryKey: ['ui-kit', 'users', list.query],
    queryFn: ({ signal }) => api.get('/users', { query: list.query, schema: usersListSchema, signal }),
    placeholderData: keepPreviousData,
  });

  const columns: Column<UserRow>[] = [
    { key: 'id', header: '#', cell: (u) => formatNumber(u.id, locale), className: 'cell-num w-16 text-muted' },
    { key: 'name', header: t('name'), cell: (u) => u.name || '—' },
    { key: 'mobile', header: t('mobile'), cell: (u) => (u.mobile ? <span dir="ltr">{u.mobile}</span> : '—') },
    {
      key: 'status',
      header: t('status'),
      cell: (u) => (u.is_blocked ? <Badge tone="red">{tc('blocked')}</Badge> : <Badge tone="green">{tc('active')}</Badge>),
    },
    { key: 'created', header: '', cell: (u) => formatDateTime(u.created_at, locale), className: 'cell-num text-ink-3' },
  ];

  return (
    <Panel
      title={t('serverList')}
      bodyClassName=""
      actions={query.isFetching && query.data ? <span className="text-xs text-muted">{tc('loading')}</span> : null}
    >
      <div className="flex flex-wrap items-end gap-3 border-b border-line px-4 py-3">
        <SearchInput value={list.params.q} onChange={list.setSearch} />
        <Select
          label={t('status')}
          className="w-40"
          value={list.params.filters.status}
          onChange={(e) => list.setFilter('status', e.target.value)}
          options={[
            { value: 'all', label: t('statusAll') },
            { value: 'active', label: t('statusActive') },
            { value: 'blocked', label: t('statusBlocked') },
          ]}
        />
      </div>
      {query.error && query.data ? <p className="m-0 px-4 py-2 text-danger-deep">{describe(query.error)}</p> : null}
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(u) => u.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
      />
      <Pagination meta={query.data?.meta} onPage={list.setPage} />
    </Panel>
  );
}
