'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { formatDateTime, formatNumber, useListParams } from '@/shared/lib';
import {
  Button,
  confirm,
  DataTable,
  Icon,
  Pagination,
  Panel,
  RowActions,
  SearchInput,
  Select,
  toast,
  useNotifyError,
  type Column,
} from '@/shared/ui';

import { discountsApi, plansApi, useRemoveOrDeactivate, type Discount } from '../api/billing';
import { StatusBadge, Toman, useCanManage } from './parts';

/** /plus/discount-codes — codes with their usage, limits and window (B-N2-09). */
export function DiscountsScreen() {
  const t = useTranslations('plus.discounts');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const canManage = useCanManage();
  const notifyError = useNotifyError();
  const list = useListParams({ status: 'all' });
  const query = discountsApi.useList(list.query);
  const plans = plansApi.useList().data?.items ?? [];
  const remove = useRemoveOrDeactivate(discountsApi.path, discountsApi.keys.all);
  const n = (v: number) => formatNumber(v, locale);
  const planName = (id: number) => {
    const p = plans.find((x) => x.id === id);
    return p ? localize(p.title) || p.code : `#${n(id)}`;
  };

  const onRemove = async (d: Discount) => {
    const used = d.uses.paid + d.uses.pending > 0;
    const ok = await confirm({
      message: used ? t('confirmDeactivate') : tc('confirmDelete'),
      confirmLabel: used ? tc('deactivate') : tc('delete'),
      tone: 'danger',
    });
    if (!ok) return;
    remove.mutate(d.id, {
      onSuccess: (res) => toast.success(res.deleted ? tc('deleted') : t('deactivated')),
      onError: notifyError,
    });
  };

  const columns: Column<Discount>[] = [
    { key: 'code', header: t('code'), cell: (d) => <code dir="ltr" className="font-semibold">{d.code}</code> },
    {
      key: 'value',
      header: t('value'),
      cell: (d) => (d.kind === 'percent' ? t('percentValue', { value: n(d.value) }) : <Toman rials={d.value} />),
      className: 'whitespace-nowrap',
    },
    {
      key: 'plans',
      header: t('plans'),
      cell: (d) => (d.plan_ids && d.plan_ids.length ? d.plan_ids.map(planName).join('، ') : t('allPlans')),
      className: 'cell-wrap text-ink-3',
    },
    {
      key: 'uses',
      header: t('uses'),
      cell: (d) => (
        <span className="flex flex-col text-xs">
          <span className="font-semibold tabular-nums">
            {d.max_redemptions ? t('usesOf', { used: n(d.uses.paid), max: n(d.max_redemptions) }) : n(d.uses.paid)}
          </span>
          {d.uses.pending ? <span className="text-muted">{t('pending', { count: d.uses.pending })}</span> : null}
          <span className="text-muted">{d.per_user_limit ? t('perUser', { count: d.per_user_limit }) : t('perUserUnlimited')}</span>
        </span>
      ),
    },
    {
      key: 'window',
      header: t('window'),
      cell: (d) => (
        <span className="flex flex-col gap-0.5 text-xs text-ink-3 tabular-nums">
          <span>{d.starts_at ? formatDateTime(d.starts_at, locale) : t('fromNow')}</span>
          <span>{d.expires_at ? t('until', { date: formatDateTime(d.expires_at, locale) }) : t('noEnd')}</span>
        </span>
      ),
    },
    { key: 'state', header: tc('status'), cell: (d) => <StatusBadge status={d.state} /> },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (d) =>
        canManage ? (
          <RowActions>
            <Link href={`/plus/discount-codes/${d.id}`} className="btn btn-sm">
              {tc('edit')}
            </Link>
            {d.is_active || d.uses.paid + d.uses.pending === 0 ? (
              <Button size="sm" variant="danger" onClick={() => void onRemove(d)} loading={remove.isPending && remove.variables === d.id}>
                {d.uses.paid + d.uses.pending > 0 ? tc('deactivate') : tc('delete')}
              </Button>
            ) : null}
          </RowActions>
        ) : null,
      className: 'cell-actions',
    },
  ];

  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        canManage ? (
          <Link href="/plus/discount-codes/new" className="btn btn-primary btn-sm">
            <Icon name="plus" size={15} />
            {t('new')}
          </Link>
        ) : null
      }
    >
      <div className="filter-bar">
        <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('searchPlaceholder')} />
        <Select
          label={tc('status')}
          className="w-40"
          value={list.params.filters.status}
          onChange={(e) => list.setFilter('status', e.target.value)}
          options={[
            { value: 'all', label: t('statusAll') },
            { value: 'active', label: tc('active') },
            { value: 'inactive', label: tc('inactive') },
          ]}
        />
      </div>
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(d) => d.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={canManage ? (d) => router.push(`/plus/discount-codes/${d.id}`) : undefined}
        emptyText={list.params.q || list.params.filters.status !== 'all' ? t('emptyFiltered') : t('empty')}
        caption={t('title')}
      />
      <Pagination meta={query.data?.meta} onPage={list.setPage} />
    </Panel>
  );
}
