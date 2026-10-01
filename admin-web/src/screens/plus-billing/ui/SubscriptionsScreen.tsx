'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { fieldError } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { formatDate, formatNumber, toIntOrNull, useListParams } from '@/shared/lib';
import {
  Button,
  DataTable,
  Pagination,
  Panel,
  RowActions,
  SearchInput,
  Select,
  TextArea,
  TextInput,
  toast,
  useNotifyError,
  type Column,
} from '@/shared/ui';

import { plansApi, subscriptionsApi, type Subscription } from '../api/billing';
import { FormDialog } from './FormDialog';
import { StatusBadge, useCanManage, UserCell } from './parts';

const STATUSES = ['all', 'active', 'canceled', 'expired', 'refunded'] as const;

/** /plus/subscriptions — every subscription period, filterable; super admins extend running ones (B-N2-09). */
export function SubscriptionsScreen() {
  const t = useTranslations('plus.subscriptions');
  const tp = useTranslations('plus');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const localize = useLocalized();
  const canManage = useCanManage();
  const list = useListParams({ status: 'all', plan_id: '', from: '', to: '' });
  const query = subscriptionsApi.useList(list.query);
  const plans = plansApi.useList().data?.items ?? [];
  const counts = query.data?.counts;
  const n = (v: number) => formatNumber(v, locale);
  const [extending, setExtending] = useState<Subscription | null>(null);

  const statusLabel = (s: (typeof STATUSES)[number]) => {
    const label = s === 'all' ? t('statusAll') : tp(`status.${s}`);
    return counts && s !== 'all' ? `${label} (${n(counts[s])})` : label;
  };

  const columns: Column<Subscription>[] = [
    { key: 'id', header: '#', cell: (s) => n(s.id), className: 'cell-num w-14 text-muted' },
    { key: 'user', header: t('subscriber'), cell: (s) => <UserCell user={s.user} /> },
    { key: 'plan', header: t('plan'), cell: (s) => (s.plan ? localize(s.plan.title) || s.plan.code : '—') },
    {
      key: 'status',
      header: tc('status'),
      cell: (s) => (
        <span className="flex flex-col items-start gap-1">
          <StatusBadge status={s.effective_status} />
          {s.source === 'admin' ? <span className="text-xs text-muted">{t('sourceAdmin')}</span> : null}
        </span>
      ),
    },
    {
      key: 'period',
      header: t('period'),
      cell: (s) => (
        <span className="flex flex-col text-xs tabular-nums text-ink-3">
          <span>{t('from', { date: formatDate(s.starts_at, locale) })}</span>
          <span>{t('to', { date: formatDate(s.ends_at, locale) })}</span>
        </span>
      ),
    },
    {
      key: 'left',
      header: t('daysLeft'),
      cell: (s) => (s.days_left > 0 ? t('days', { count: s.days_left }) : '—'),
      className: 'cell-num whitespace-nowrap',
    },
    { key: 'renew', header: t('autoRenew'), cell: (s) => (s.auto_renew ? t('renewOn') : t('renewOff')), className: 'text-ink-3' },
    {
      key: 'invoice',
      header: t('invoice'),
      cell: (s) => (s.invoice_reference ? <code dir="ltr" className="text-xs">{s.invoice_reference}</code> : '—'),
    },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (s) =>
        canManage && (s.effective_status === 'active' || s.effective_status === 'canceled') ? (
          <RowActions>
            <Button size="sm" onClick={() => setExtending(s)}>
              {t('extend')}
            </Button>
          </RowActions>
        ) : null,
      className: 'cell-actions',
    },
  ];

  const filtered = Boolean(list.params.q) || Object.entries(list.params.filters).some(([k, v]) => (k === 'status' ? v !== 'all' : v !== ''));

  return (
    <Panel title={t('title')} bodyClassName="">
      <div className="filter-bar">
        <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('searchPlaceholder')} />
        <Select
          label={tc('status')}
          className="w-44"
          value={list.params.filters.status}
          onChange={(e) => list.setFilter('status', e.target.value)}
          options={STATUSES.map((s) => ({ value: s, label: statusLabel(s) }))}
        />
        <Select
          label={t('plan')}
          className="w-40"
          value={list.params.filters.plan_id}
          onChange={(e) => list.setFilter('plan_id', e.target.value)}
          options={[{ value: '', label: t('allPlans') }, ...plans.map((p) => ({ value: String(p.id), label: localize(p.title) || p.code }))]}
        />
        <TextInput label={t('startFrom')} type="date" className="w-40" value={list.params.filters.from} onChange={(e) => list.setFilter('from', e.target.value)} />
        <TextInput label={t('startTo')} type="date" className="w-40" value={list.params.filters.to} onChange={(e) => list.setFilter('to', e.target.value)} />
      </div>
      <p className="panel-note">{t('maskNote')}</p>
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(s) => s.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        emptyText={filtered ? t('emptyFiltered') : t('empty')}
        caption={t('title')}
      />
      <Pagination meta={query.data?.meta} onPage={list.setPage} />
      <ExtendDialog subscription={extending} onClose={() => setExtending(null)} />
    </Panel>
  );
}

function ExtendDialog({ subscription, onClose }: { subscription: Subscription | null; onClose: () => void }) {
  const t = useTranslations('plus.subscriptions');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const extend = subscriptionsApi.useAction('extend');
  const [days, setDays] = useState('7');
  const [note, setNote] = useState('');
  const err = (name: string) => fieldError(extend.error, name);

  const close = () => {
    extend.reset();
    setDays('7');
    setNote('');
    onClose();
  };
  const submit = () => {
    if (!subscription) return;
    extend.mutate(
      { id: subscription.id, body: { days: toIntOrNull(days), note: note.trim() } },
      {
        onSuccess: () => {
          toast.success(t('extended'));
          close();
        },
        onError: notifyError,
      },
    );
  };

  return (
    <FormDialog
      open={subscription !== null}
      title={t('extendTitle', { id: formatNumber(subscription?.id ?? 0, locale) })}
      onClose={close}
      onSubmit={submit}
      submitLabel={t('extend')}
      saving={extend.isPending}
    >
      <p className="m-0 text-sm text-ink-3">{t('extendIntro', { date: formatDate(subscription?.ends_at, locale) })}</p>
      <TextInput label={t('extendDays')} hint={t('extendDaysHint')} type="number" value={days} onChange={(e) => setDays(e.target.value)} required error={err('days')} />
      <TextArea label={t('note')} hint={t('noteHint')} value={note} onChange={(e) => setNote(e.target.value)} rows={3} maxLength={500} required error={err('note')} />
    </FormDialog>
  );
}
