'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { formatDateTime, formatNumber, useListParams } from '@/shared/lib';
import { DataTable, Pagination, Panel, SearchInput, Select, Skeleton, TextInput, type Column } from '@/shared/ui';

import { paymentsApi, type PaymentRow } from '../api/billing';
import { rialsToToman } from '../lib/money';
import { StatusBadge, Toman, UserCell } from './parts';

const STATUSES = ['all', 'paid', 'pending', 'failed', 'expired', 'refunded'] as const;

/** /plus/payments — the payment log: invoices and verified receipts (no card data) (B-N2-09). */
export function PaymentsScreen() {
  const t = useTranslations('plus.payments');
  const tp = useTranslations('plus');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const list = useListParams({ status: 'all', gateway: '', from: '', to: '' });
  const query = paymentsApi.useList(list.query);
  const summary = query.data?.summary;
  const n = (v: number) => formatNumber(v, locale);

  const columns: Column<PaymentRow>[] = [
    {
      key: 'ref',
      header: t('reference'),
      cell: (p) => (
        <code dir="ltr" className="text-xs font-semibold">
          {p.reference}
        </code>
      ),
    },
    { key: 'user', header: t('payer'), cell: (p) => <UserCell user={p.user} /> },
    { key: 'plan', header: t('plan'), cell: (p) => (p.plan ? localize(p.plan.title) || p.plan.code : '—') },
    {
      key: 'amount',
      header: t('amount'),
      cell: (p) => (
        <span className="flex flex-col items-end">
          <Toman rials={p.total_rials} />
          {p.discount_code ? (
            <span dir="ltr" className="text-xs text-muted">
              {p.discount_code}
            </span>
          ) : null}
        </span>
      ),
      className: 'cell-num',
    },
    { key: 'gateway', header: t('gateway'), cell: (p) => (p.gateway ? <span dir="ltr">{p.gateway}</span> : t('noGateway')), className: 'text-ink-3' },
    {
      key: 'bankRef',
      header: t('bankRef'),
      cell: (p) => (p.receipt ? <code dir="ltr" className="text-xs">{p.receipt.ref_id}</code> : '—'),
    },
    { key: 'status', header: tc('status'), cell: (p) => <StatusBadge status={p.effective_status} /> },
    { key: 'date', header: t('date'), cell: (p) => formatDateTime(p.created_at, locale), className: 'cell-num whitespace-nowrap text-ink-3' },
  ];

  const gateways = query.data?.gateways ?? [];
  const filtered = Boolean(list.params.q) || Object.entries(list.params.filters).some(([k, v]) => (k === 'status' ? v !== 'all' : v !== ''));
  const cells = [
    { key: 'paid', label: t('summaryPaid'), value: summary ? tp('toman', { amount: n(rialsToToman(summary.paid_rials)) }) : undefined },
    { key: 'count', label: t('summaryCount'), value: summary ? n(summary.paid_count) : undefined },
    { key: 'refunded', label: t('summaryRefunded'), value: summary ? tp('toman', { amount: n(rialsToToman(summary.refunded_rials)) }) : undefined },
  ];

  return (
    <div className="flex flex-col gap-4">
      <Panel bodyClassName="">
        <dl className="ledger m-0">
          {cells.map((c) => (
            <div key={c.key} className="ledger-cell">
              <dt className="text-[13px] font-semibold text-ink-3">{c.label}</dt>
              <dd className="m-0 mt-1">{c.value === undefined ? <Skeleton className="h-8 w-24" /> : <span className="ledger-value">{c.value}</span>}</dd>
            </div>
          ))}
        </dl>
      </Panel>
      <Panel title={t('title')} bodyClassName="">
        <div className="filter-bar">
          <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('searchPlaceholder')} />
          <Select
            label={tc('status')}
            className="w-40"
            value={list.params.filters.status}
            onChange={(e) => list.setFilter('status', e.target.value)}
            options={STATUSES.map((s) => ({ value: s, label: s === 'all' ? t('statusAll') : tp(`status.${s}`) }))}
          />
          <Select
            label={t('gateway')}
            className="w-36"
            value={list.params.filters.gateway}
            onChange={(e) => list.setFilter('gateway', e.target.value)}
            options={[{ value: '', label: t('allGateways') }, ...gateways.map((g) => ({ value: g, label: g }))]}
          />
          <TextInput label={t('dateFrom')} type="date" className="w-40" value={list.params.filters.from} onChange={(e) => list.setFilter('from', e.target.value)} />
          <TextInput label={t('dateTo')} type="date" className="w-40" value={list.params.filters.to} onChange={(e) => list.setFilter('to', e.target.value)} />
        </div>
        <p className="panel-note">{t('privacyNote')}</p>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(p) => p.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          onRowClick={(p) => router.push(`/plus/payments/${p.id}`)}
          emptyText={filtered ? t('emptyFiltered') : t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}
