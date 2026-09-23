'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { RequireSuper, useCurrentAdmin } from '@/features/auth';
import { RoleBadge, type Admin } from '@/entities/admin';
import { formatDateTime, formatNumber, useListParams } from '@/shared/lib';
import {
  Badge,
  CrudRowActions,
  DataTable,
  Icon,
  Pagination,
  Panel,
  useRowCommands,
  type Column,
} from '@/shared/ui';

import { adminsApi } from '../api/admins';

/** /admins — admin accounts (super admins; Blade admins.index). */
export function AdminsScreen() {
  return (
    <RequireSuper>
      <AdminsList />
    </RequireSuper>
  );
}

function AdminsList() {
  const t = useTranslations('admins');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const me = useCurrentAdmin();
  const list = useListParams();
  const query = adminsApi.useList(list.query);
  const commands = useRowCommands({ remove: adminsApi.useRemove(), confirmDelete: t('confirmDelete') });

  const columns: Column<Admin>[] = [
    { key: 'id', header: '#', cell: (a) => formatNumber(a.id, locale), className: 'cell-num w-14 text-muted' },
    {
      key: 'name',
      header: t('name'),
      cell: (a) => (
        <span className="flex items-center gap-2 font-semibold">
          {a.name}
          {a.id === me?.id ? <Badge>{t('you')}</Badge> : null}
        </span>
      ),
    },
    { key: 'email', header: t('email'), cell: (a) => <span dir="ltr">{a.email}</span> },
    { key: 'role', header: t('role'), cell: (a) => <RoleBadge role={a.role} /> },
    {
      key: 'status',
      header: tc('status'),
      cell: (a) => (a.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge tone="red">{tc('inactive')}</Badge>),
    },
    {
      key: 'login',
      header: t('lastLogin'),
      cell: (a) => formatDateTime(a.last_login_at, locale) || '—',
      className: 'cell-num whitespace-nowrap text-ink-3',
    },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (a) => <CrudRowActions id={a.id} editHref={`/admins/${a.id}`} commands={commands} canDelete={a.id !== me?.id} />,
      className: 'cell-actions',
    },
  ];

  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        <Link href="/admins/new" className="btn btn-primary btn-sm">
          <Icon name="plus" size={15} />
          {t('new')}
        </Link>
      }
    >
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(a) => a.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={(a) => router.push(`/admins/${a.id}`)}
        emptyText={t('empty')}
        caption={t('title')}
      />
      <Pagination meta={query.data?.meta} onPage={list.setPage} />
    </Panel>
  );
}
