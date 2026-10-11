'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useLocalized } from '@/shared/i18n';
import { formatDate, formatDateTime, formatNumber, useListParams } from '@/shared/lib';
import {
  Badge,
  Button,
  confirm,
  DataTable,
  LoadGate,
  PageHeader,
  Pagination,
  Panel,
  SearchInput,
  Select,
  Switch,
  TextInput,
  toast,
  useNotifyError,
  type Column,
} from '@/shared/ui';

import { ordersApi, returnsApi, shipmentsApi, useCommerceOptions, type AdminOrder, type ReturnRow, type ShipmentRow } from '../api/commerce';

import { Money, ShopTabs, StatusBadge } from './ShopTabs';

/**
 * CB-SHOP-05 order screens (capability `support`): the shipment queue, one order and the returns. They show the
 * recipient's name, phone and address for delivery only — nothing here is copied elsewhere or logged.
 */

function slotText(slot: ShipmentRow['slot'], locale: string) {
  return slot ? `${formatDate(slot.date, locale)} · ${slot.starts_at}–${slot.ends_at}` : '—';
}

/** POST /shop/shipments/:id/move — the next statuses the server allows, with the tracking code for «shipped». */
function MoveButtons({ id, allowed }: { id: number; allowed: readonly string[] }) {
  const t = useTranslations('shopAdmin.orders');
  const notifyError = useNotifyError();
  const move = shipmentsApi.useAction('move');
  const [tracking, setTracking] = useState('');
  if (allowed.length === 0) return null;
  const go = (status: string) =>
    move.mutate(
      { id, body: { status, tracking_code: status === 'shipped' && tracking.trim() !== '' ? tracking.trim() : null } },
      { onSuccess: () => toast.success(t('moved')), onError: notifyError },
    );
  return (
    <div className="flex flex-col items-start gap-2" onClick={(e) => e.stopPropagation()}>
      {allowed.includes('shipped') ? (
        <TextInput aria-label={t('tracking')} placeholder={t('tracking')} value={tracking} onChange={(e) => setTracking(e.target.value)} dir="ltr" maxLength={64} />
      ) : null}
      <div className="row-actions">
        {allowed.map((to) => (
          <Button key={to} size="sm" variant={to === 'delivered' ? 'primary' : 'default'} loading={move.isPending} onClick={() => go(to)}>
            {t(`moveTo.${to}` as 'moveTo.confirmed')}
          </Button>
        ))}
      </div>
    </div>
  );
}

/** /shop/shipments — the queue, one row per seller shipment, newest first. */
export function ShopShipmentsScreen() {
  const t = useTranslations('shopAdmin.orders');
  const locale = useLocale();
  const localize = useLocalized();
  const options = useCommerceOptions();
  const list = useListParams({ status: 'awaiting_seller', seller_id: '' });
  const status = list.params.filters.status ?? 'awaiting_seller';
  const query = shipmentsApi.useList(list.query);
  const ts = useTranslations('shopAdmin.status');

  const columns: Column<ShipmentRow>[] = [
    {
      key: 'code',
      header: t('code'),
      cell: (s) => (
        <span className="flex flex-col">
          <Link href={`/shop/orders/${s.order_id}`} className="text-brand no-underline hover:underline" dir="ltr">
            {s.order_code}
          </Link>
          <span className="text-xs text-muted">{formatDateTime(s.created_at, locale)}</span>
        </span>
      ),
    },
    { key: 'seller', header: t('seller'), cell: (s) => localize(s.seller.name) || '—' },
    {
      key: 'recipient',
      header: t('recipient'),
      cell: (s) => (
        <span className="flex flex-col">
          <span>{s.recipient_name}</span>
          <span className="text-xs text-muted">{s.city}</span>
        </span>
      ),
    },
    { key: 'slot', header: t('slot'), cell: (s) => slotText(s.slot, locale), className: 'whitespace-nowrap' },
    { key: 'total', header: t('payable'), cell: (s) => <Money value={s.payable_total} /> },
    {
      key: 'status',
      header: t('status'),
      cell: (s) => (
        <span className="flex flex-wrap gap-1">
          <StatusBadge status={s.status} />
          {s.discreet ? <Badge tone="neutral">{t('discreet')}</Badge> : null}
        </span>
      ),
    },
    { key: 'actions', header: <span className="sr-only">{t('actions')}</span>, cell: (s) => <MoveButtons id={s.id} allowed={s.allowed_statuses} />, className: 'cell-actions' },
  ];

  return (
    <div className="flex flex-col gap-4">
      <ShopTabs group="orders" />
      <Panel title={t('title')} bodyClassName="">
        <div className="filter-bar">
          <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('search')} />
          <Select
            aria-label={t('status')}
            value={status}
            onChange={(e) => list.setFilter('status', e.target.value)}
            options={[{ value: 'all', label: t('allStatuses') }, ...(options.data?.shipment_statuses ?? []).map((s) => ({ value: s, label: ts(s as 'pending') }))]}
          />
          <Select
            aria-label={t('seller')}
            value={list.params.filters.seller_id ?? ''}
            onChange={(e) => list.setFilter('seller_id', e.target.value)}
            options={[{ value: '', label: t('allSellers') }, ...(options.data?.sellers ?? []).map((s) => ({ value: String(s.id), label: localize(s.name) || s.slug }))]}
          />
        </div>
        <p className="panel-note">{t('privacyNote')}</p>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(s) => s.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}

/** /shop/orders/:id — recipient, address, shipments with their moves, items, timeline and returns. */
export function ShopOrderScreen({ id }: { id: number }) {
  const t = useTranslations('shopAdmin.orders');
  const detail = ordersApi.useDetail(id);
  return (
    <LoadGate queries={[detail]} header={<PageHeader title={t('order')} backHref="/shop/shipments" backLabel={t('back')} />}>
      {() => (detail.data ? <OrderDetail order={detail.data.order} /> : null)}
    </LoadGate>
  );
}

function OrderDetail({ order }: { order: AdminOrder }) {
  const t = useTranslations('shopAdmin.orders');
  const tr = useTranslations('shopAdmin.returns');
  const locale = useLocale();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const cancel = ordersApi.useAction('cancel');

  const doCancel = async () => {
    if (!(await confirm({ message: t('cancelConfirm'), confirmLabel: t('cancel'), tone: 'danger' }))) return;
    cancel.mutate({ id: order.id }, { onSuccess: () => toast.success(t('cancelled')), onError: notifyError });
  };

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={<span dir="ltr">{order.code}</span>}
        backHref="/shop/shipments"
        backLabel={t('back')}
        meta={<StatusBadge status={order.status} />}
        actions={
          order.can_cancel ? (
            <Button variant="danger" size="sm" loading={cancel.isPending} onClick={() => void doCancel()}>
              {t('cancel')}
            </Button>
          ) : null
        }
      />
      <Panel title={t('recipient')}>
        <dl className="grid gap-2 md:grid-cols-2">
          <div>
            <dt className="text-xs text-muted">{t('recipientName')}</dt>
            <dd>{order.recipient_name}</dd>
          </div>
          <div>
            <dt className="text-xs text-muted">{t('phone')}</dt>
            <dd dir="ltr" className="text-start">
              {order.recipient_phone}
            </dd>
          </div>
          <div className="md:col-span-2">
            <dt className="text-xs text-muted">{t('address')}</dt>
            <dd>
              {order.city} — {order.address_text}
              {order.postal_code ? <span dir="ltr"> ({order.postal_code})</span> : null}
            </dd>
          </div>
          <div>
            <dt className="text-xs text-muted">{t('payment')}</dt>
            <dd>
              {t('cod')} · {order.discreet ? t('discreet') : t('normalPackaging')}
            </dd>
          </div>
          <div>
            <dt className="text-xs text-muted">{t('totals')}</dt>
            <dd className="flex flex-wrap gap-2">
              <span>
                {t('items')}: <Money value={order.items_total} />
              </span>
              <span>
                {t('shipping')}: <Money value={order.shipping_total} />
              </span>
              {order.discount_total > 0 ? (
                <span>
                  {t('discount')}: <Money value={order.discount_total} />
                </span>
              ) : null}
              <strong>
                {t('payable')}: <Money value={order.payable_total} />
              </strong>
            </dd>
          </div>
        </dl>
      </Panel>
      {order.shipments.map((s) => (
        <Panel key={s.id} title={localize(s.seller.name) || t('shipment')} actions={<StatusBadge status={s.status} />}>
          <p className="text-sm text-muted">
            {t('slot')}: {slotText(s.slot, locale)}
            {s.tracking_code ? (
              <>
                {' · '}
                {t('tracking')}: <span dir="ltr">{s.tracking_code}</span>
              </>
            ) : null}
            {s.delivered_at ? ` · ${t('deliveredAt', { at: formatDateTime(s.delivered_at, locale) })}` : null}
          </p>
          <ul className="my-3 flex flex-col gap-1">
            {s.items.map((it) => (
              <li key={it.id} className="flex flex-wrap justify-between gap-2">
                <span>
                  {localize(it.title)}
                  {localize(it.variant_label) ? <span className="text-muted"> · {localize(it.variant_label)}</span> : null}
                  {' × '}
                  {formatNumber(it.quantity, locale)}
                </span>
                <Money value={it.line_total} />
              </li>
            ))}
          </ul>
          <MoveButtons id={s.id} allowed={s.allowed_statuses} />
        </Panel>
      ))}
      <Panel title={t('timeline')}>
        <ol className="flex flex-col gap-1">
          {order.events.map((e) => (
            <li key={e.id} className="flex gap-2">
              <span className="text-xs text-muted">{formatDateTime(e.created_at, locale)}</span>
              <StatusBadge status={e.status} />
            </li>
          ))}
        </ol>
      </Panel>
      {order.returns.length > 0 ? (
        <Panel title={tr('title')}>
          <ul className="flex flex-col gap-1">
            {order.returns.map((r) => (
              <li key={r.id} className="flex gap-2">
                <StatusBadge status={r.status} />
                <span>
                  {tr(`reason.${r.reason}` as 'reason.other')} × {formatNumber(r.quantity, locale)}
                </span>
              </li>
            ))}
          </ul>
          <Link href="/shop/returns" className="btn btn-sm mt-2">
            {tr('open')}
          </Link>
        </Panel>
      ) : null}
    </div>
  );
}

/** /shop/returns — return requests; approve (optionally putting the item back in stock) or reject. */
export function ShopReturnsScreen() {
  const t = useTranslations('shopAdmin.returns');
  const ts = useTranslations('shopAdmin.status');
  const locale = useLocale();
  const localize = useLocalized();
  const list = useListParams({ status: 'requested' });
  const status = list.params.filters.status ?? 'requested';
  const query = returnsApi.useList(list.query);

  const columns: Column<ReturnRow>[] = [
    {
      key: 'order',
      header: t('order'),
      cell: (r) => (
        <span className="flex flex-col">
          <Link href={`/shop/orders/${r.order_id}`} className="text-brand no-underline hover:underline" dir="ltr">
            {r.order_code}
          </Link>
          <span className="text-xs text-muted">{formatDateTime(r.created_at, locale)}</span>
        </span>
      ),
    },
    {
      key: 'item',
      header: t('item'),
      cell: (r) => `${localize(r.item_title)}${localize(r.variant_label) ? ` · ${localize(r.variant_label)}` : ''} × ${formatNumber(r.quantity, locale)}`,
    },
    {
      key: 'reason',
      header: t('reasonLabel'),
      cell: (r) => (
        <span className="flex flex-col">
          <span>{t(`reason.${r.reason}` as 'reason.other')}</span>
          {r.note ? <span className="text-xs text-muted">{r.note}</span> : null}
        </span>
      ),
    },
    { key: 'status', header: t('status'), cell: (r) => <StatusBadge status={r.status} /> },
    { key: 'actions', header: <span className="sr-only">{t('actions')}</span>, cell: (r) => <Decide row={r} />, className: 'cell-actions' },
  ];

  return (
    <div className="flex flex-col gap-4">
      <ShopTabs group="orders" />
      <Panel title={t('title')} bodyClassName="">
        <div className="filter-bar">
          <Select
            aria-label={t('status')}
            value={status}
            onChange={(e) => list.setFilter('status', e.target.value)}
            options={[{ value: 'all', label: t('allStatuses') }, ...['requested', 'approved', 'rejected'].map((s) => ({ value: s, label: ts(s as 'pending') }))]}
          />
        </div>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(r) => r.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}

function Decide({ row }: { row: ReturnRow }) {
  const t = useTranslations('shopAdmin.returns');
  const notifyError = useNotifyError();
  const approve = returnsApi.useAction('approve');
  const reject = returnsApi.useAction('reject');
  const [restock, setRestock] = useState(row.can_restock);
  if (row.status !== 'requested') return null;
  const done = { onSuccess: () => toast.success(t('decided')), onError: notifyError };
  return (
    <div className="flex flex-col items-start gap-2">
      {row.can_restock ? <Switch label={t('restock')} checked={restock} onChange={setRestock} /> : null}
      <div className="row-actions">
        <Button size="sm" variant="primary" loading={approve.isPending} onClick={() => approve.mutate({ id: row.id, body: { restock } }, done)}>
          {t('approve')}
        </Button>
        <Button size="sm" variant="danger" loading={reject.isPending} onClick={() => reject.mutate({ id: row.id }, done)}>
          {t('reject')}
        </Button>
      </div>
    </div>
  );
}
