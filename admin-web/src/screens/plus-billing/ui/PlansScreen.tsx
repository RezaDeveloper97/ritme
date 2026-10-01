'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib';
import { Badge, Button, confirm, DataTable, Icon, Panel, RowActions, toast, useNotifyError, type Column } from '@/shared/ui';

import { plansApi, useRemoveOrDeactivate, type Plan } from '../api/billing';
import { StatusBadge, Toman, useCanManage } from './parts';

/** /plus/plans — Ritme Plus plans in paywall order (B-N2-09). */
export function PlansScreen() {
  const t = useTranslations('plus.plans');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const canManage = useCanManage();
  const notifyError = useNotifyError();
  const query = plansApi.useList();
  const remove = useRemoveOrDeactivate(plansApi.path, plansApi.keys.all);
  const n = (v: number) => formatNumber(v, locale);

  const onRemove = async (plan: Plan) => {
    const used = plan.invoices_count + plan.subscriptions_count > 0;
    const ok = await confirm({
      message: used ? t('confirmDeactivate') : tc('confirmDelete'),
      confirmLabel: used ? tc('deactivate') : tc('delete'),
      tone: 'danger',
    });
    if (!ok) return;
    remove.mutate(plan.id, {
      onSuccess: (res) => toast.success(res.deleted ? tc('deleted') : t('deactivated')),
      onError: notifyError,
    });
  };

  const columns: Column<Plan>[] = [
    { key: 'sort', header: tc('sortOrder'), cell: (p) => n(p.sort_order), className: 'cell-num w-14 text-muted' },
    {
      key: 'title',
      header: t('plan'),
      cell: (p) => (
        <span className="flex flex-col items-start gap-1">
          <span className="font-semibold">{localize(p.title) || p.code}</span>
          <span className="flex flex-wrap gap-1">
            {p.is_highlighted ? <Badge tone="brand">{t('highlighted')}</Badge> : null}
            {localize(p.badge) ? <Badge tone="data">{localize(p.badge)}</Badge> : null}
          </span>
        </span>
      ),
    },
    { key: 'code', header: t('code'), cell: (p) => <code dir="ltr">{p.code}</code> },
    { key: 'duration', header: t('duration'), cell: (p) => t('months', { count: p.duration_months }), className: 'whitespace-nowrap' },
    { key: 'price', header: t('price'), cell: (p) => <Toman rials={p.price_rials} />, className: 'cell-num' },
    {
      key: 'monthly',
      header: t('monthly'),
      cell: (p) => (
        <span className="flex flex-col items-end text-xs text-ink-3">
          <Toman rials={p.monthly_price_rials} />
          {p.monthly_display_rials ? <span>{t('monthlyOverride')}</span> : null}
        </span>
      ),
      className: 'cell-num',
    },
    { key: 'subs', header: t('activeSubscriptions'), cell: (p) => n(p.active_subscriptions), className: 'cell-num' },
    { key: 'status', header: tc('status'), cell: (p) => <StatusBadge status={p.is_active ? 'active' : 'inactive'} /> },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (p) =>
        canManage ? (
          <RowActions>
            <Link href={`/plus/plans/${p.id}`} className="btn btn-sm" onClick={(e) => e.stopPropagation()}>
              {tc('edit')}
            </Link>
            <Button
              size="sm"
              variant="danger"
              onClick={(e) => {
                e.stopPropagation();
                void onRemove(p);
              }}
              loading={remove.isPending && remove.variables === p.id}
            >
              {p.invoices_count + p.subscriptions_count > 0 ? tc('deactivate') : tc('delete')}
            </Button>
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
          <Link href="/plus/plans/new" className="btn btn-primary btn-sm">
            <Icon name="plus" size={15} />
            {t('new')}
          </Link>
        ) : null
      }
    >
      <p className="panel-note">{t('intro')}</p>
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(p) => p.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={canManage ? (p) => router.push(`/plus/plans/${p.id}`) : undefined}
        emptyText={t('empty')}
        caption={t('title')}
      />
    </Panel>
  );
}
