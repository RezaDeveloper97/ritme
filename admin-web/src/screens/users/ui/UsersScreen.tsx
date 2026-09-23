'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { formatDate, formatNumber, useListParams } from '@/shared/lib';
import { Badge, DataTable, Pagination, Panel, SearchInput, Select, type Column } from '@/shared/ui';

import { usersApi, type UserRow } from '../api/users';
import { useUserLabels } from './labels';

/** /users — search by name/mobile/email, filter by status (Blade users.index). */
export function UsersScreen() {
  const t = useTranslations('users');
  const tc = useTranslations('common');
  const locale = useLocale();
  const router = useRouter();
  const labels = useUserLabels();
  const list = useListParams({ status: 'all' });
  const query = usersApi.useList(list.query);

  const columns: Column<UserRow>[] = [
    { key: 'id', header: '#', cell: (u) => formatNumber(u.id, locale), className: 'cell-num w-16 text-muted' },
    { key: 'name', header: t('name'), cell: (u) => u.name || '—' },
    {
      key: 'mobile',
      header: t('mobile'),
      cell: (u) => (u.mobile ? <span dir="ltr">{u.mobile}</span> : '—'),
      className: 'cell-num',
    },
    {
      key: 'subscription',
      header: t('subscription'),
      cell: (u) => (
        <Badge tone={u.subscription_type === 'premium' ? 'brand' : 'neutral'}>{labels.subscription(u.subscription_type)}</Badge>
      ),
    },
    { key: 'goal', header: t('goal'), cell: (u) => labels.goal(u.user_goal) },
    {
      key: 'status',
      header: t('status'),
      cell: (u) => (u.is_blocked ? <Badge tone="red">{tc('blocked')}</Badge> : <Badge tone="green">{tc('active')}</Badge>),
    },
    {
      key: 'joined',
      header: t('joined'),
      cell: (u) => formatDate(u.created_at, locale),
      className: 'cell-num whitespace-nowrap text-ink-3',
    },
    {
      key: 'open',
      header: <span className="sr-only">{t('actions')}</span>,
      cell: (u) => (
        <Link href={`/users/${u.id}`} className="btn btn-sm" onClick={(e) => e.stopPropagation()}>
          {t('view')}
        </Link>
      ),
      className: 'cell-actions',
    },
  ];

  return (
    <Panel title={t('title')} bodyClassName="">
      <div className="filter-bar">
        <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('searchPlaceholder')} />
        <Select
          label={t('status')}
          className="w-40"
          value={list.params.filters.status}
          onChange={(e) => list.setFilter('status', e.target.value)}
          options={[
            { value: 'all', label: t('statusAll') },
            { value: 'active', label: tc('active') },
            { value: 'blocked', label: tc('blocked') },
          ]}
        />
      </div>
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(u) => u.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={(u) => router.push(`/users/${u.id}`)}
        emptyText={list.params.q || list.params.filters.status !== 'all' ? t('emptyFiltered') : t('empty')}
        caption={t('title')}
      />
      <Pagination meta={query.data?.meta} onPage={list.setPage} />
    </Panel>
  );
}
