'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useState, type FormEvent } from 'react';

import { useCurrentAdmin } from '@/features/auth';
import { useLocalized } from '@/shared/i18n';
import { formatDateTime, formatNumber, useListParams } from '@/shared/lib';
import {
  Badge,
  Button,
  DataTable,
  LinkTabs,
  PageHeader,
  Pagination,
  Panel,
  Select,
  SearchInput,
  TextInput,
  toast,
  useNotifyError,
  type Column,
} from '@/shared/ui';

import { bookingsApi, useMarkRefunded, type AdminBooking } from '../api/telemed';
import { BOOKING_TABS, bookingTone, paymentTone, rialsToToman } from '../lib/format';

/** /telemed/bookings and ?tab=refunds — visits with payment state; the manual refund queue (admin-api.md §19c). */
export function BookingsScreen() {
  const t = useTranslations('telemed.bookings');
  const locale = useLocale();
  const localize = useLocalized();
  const isSuper = useCurrentAdmin()?.role === 'super';
  const list = useListParams({ tab: 'all', status: 'all' });
  const tab = list.params.filters.tab === 'refunds' ? 'refunds' : 'all';
  const status = list.params.filters.status ?? 'all';
  const base = { page: list.query.page, per_page: list.query.per_page, ...(list.params.q ? { q: list.params.q } : {}) };
  const query = bookingsApi.useList(
    tab === 'refunds' ? { ...base, payment_status: 'refund_pending' } : { ...base, ...(status === 'all' ? {} : { status }) },
  );
  const pending = query.data?.counts?.refund_pending;
  const n = (v: number) => formatNumber(v, locale);
  const [refund, setRefund] = useState<AdminBooking | null>(null);

  const columns: Column<AdminBooking>[] = [
    {
      key: 'ref',
      header: t('reference'),
      cell: (b) => (
        <span dir="ltr" className="font-mono text-xs">
          {b.reference}
        </span>
      ),
    },
    {
      key: 'doctor',
      header: t('doctor'),
      cell: (b) => (
        <Link href={`/telemed/doctors/${b.doctor.id}`} className="text-brand no-underline hover:underline" dir="auto">
          {localize(b.doctor.name) || `#${b.doctor.id}`}
        </Link>
      ),
    },
    {
      key: 'patient',
      header: t('patient'),
      cell: (b) => (
        <Link href={`/users/${b.user_id}`} className="text-brand no-underline hover:underline" dir="ltr">
          {b.user_mobile ?? `#${b.user_id}`}
        </Link>
      ),
    },
    { key: 'mode', header: t('modeHeader'), cell: (b) => t(`mode.${b.mode}` as 'mode.video') },
    { key: 'starts', header: t('startsAt'), cell: (b) => formatDateTime(b.starts_at, locale), className: 'cell-num whitespace-nowrap' },
    {
      key: 'status',
      header: t('status'),
      cell: (b) => <Badge tone={bookingTone(b.status)}>{t(`statusName.${b.status}` as 'statusName.held')}</Badge>,
    },
    {
      key: 'payment',
      header: t('payment'),
      cell: (b) => (
        <span className="flex flex-col items-start gap-1">
          <Badge tone={paymentTone(b.payment_status)}>{t(`paymentName.${b.payment_status}` as 'paymentName.paid')}</Badge>
          {b.refund_ref ? (
            <span dir="ltr" className="text-xs text-muted">
              {b.refund_ref}
            </span>
          ) : null}
        </span>
      ),
    },
    {
      key: 'total',
      header: t('total'),
      cell: (b) => `${n(rialsToToman(b.total_rials))} ${t('toman')}`,
      className: 'cell-num whitespace-nowrap',
    },
    {
      key: 'actions',
      header: <span className="sr-only">{t('actions')}</span>,
      cell: (b) =>
        isSuper && b.payment_status === 'refund_pending' ? (
          <div className="row-actions">
            <Button size="sm" variant="primary" onClick={() => setRefund(b)}>
              {t('markRefunded')}
            </Button>
          </div>
        ) : null,
      className: 'cell-actions',
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={t('title')} />
      <LinkTabs
        label={t('title')}
        items={BOOKING_TABS.map((k) => ({
          key: k,
          href: k === 'all' ? '/telemed/bookings' : '/telemed/bookings?tab=refunds',
          label: k === 'refunds' && pending !== undefined ? `${t('tabs.refunds')} (${n(pending)})` : t(`tabs.${k}`),
          active: k === tab,
        }))}
      />
      <Panel bodyClassName="">
        <div className="filter-bar">
          <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('searchPlaceholder')} />
          {tab === 'all' ? (
            <Select
              aria-label={t('status')}
              value={status}
              onChange={(e) => list.setFilter('status', e.target.value)}
              options={[
                { value: 'all', label: t('allStatuses') },
                ...(query.data?.statuses ?? []).map((s) => ({ value: s, label: t(`statusName.${s}` as 'statusName.held') })),
              ]}
            />
          ) : null}
        </div>
        {tab === 'refunds' ? <p className="panel-note">{t('refundsIntro')}</p> : null}
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(b) => b.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={tab === 'refunds' ? t('emptyRefunds') : t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
      <RefundDialog booking={refund} onClose={() => setRefund(null)} />
    </div>
  );
}

function RefundDialog({ booking, onClose }: { booking: AdminBooking | null; onClose: () => void }) {
  const t = useTranslations('telemed.bookings');
  const tc = useTranslations("confirm");
  const notifyError = useNotifyError();
  const mark = useMarkRefunded();
  const [ref, setRef] = useState('');
  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!booking || mark.isPending) return;
    mark.mutate(
      { id: booking.id, refundRef: ref.trim() },
      {
        onSuccess: () => {
          toast.success(t('refundedToast'));
          setRef('');
          onClose();
        },
        onError: notifyError,
      },
    );
  };
  return (
    <dialog
      className="dialog"
      aria-label={t('markRefunded')}
      ref={(el) => {
        if (!el) return;
        if (booking && !el.open) el.showModal();
        if (!booking && el.open) el.close();
      }}
      onCancel={(e) => {
        e.preventDefault();
        onClose();
      }}
    >
      {booking ? (
        <form className="flex flex-col gap-4 p-5" onSubmit={submit}>
          <h2 className="m-0 text-base font-bold">{t('markRefunded')}</h2>
          <p className="m-0 text-ink-3">{t('refundMessage', { reference: booking.reference })}</p>
          <TextInput label={t('refundRef')} hint={t('refundRefHint')} value={ref} maxLength={64} dir="ltr" onChange={(e) => setRef(e.target.value)} />
          <div className="flex justify-end gap-2">
            <Button onClick={onClose}>{tc('cancel')}</Button>
            <Button type="submit" variant="primary" loading={mark.isPending}>
              {t('markRefunded')}
            </Button>
          </div>
        </form>
      ) : null}
    </dialog>
  );
}
