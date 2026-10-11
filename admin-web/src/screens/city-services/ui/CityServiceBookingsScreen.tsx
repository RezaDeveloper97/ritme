'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useLocalized } from '@/shared/i18n';
import { formatDateTime, formatNumber, useListParams } from '@/shared/lib';
import { Badge, Button, DataTable, PageHeader, Pagination, Panel, Select, TextInput, toast, useNotifyError, type BadgeTone, type Column } from '@/shared/ui';

import { bookingsApi, useCityServiceOptions, type CityBooking } from '../api/city-services';

const TONE: Record<string, BadgeTone> = { requested: 'amber', confirmed: 'green', rejected: 'red', cancelled: 'neutral', completed: 'data' };

/**
 * /city-service-bookings — the booking request queue (B-N10-05, capability support). It holds the contact name and
 * phone the user typed, the preferred day, an optional child AGE RANGE and a note: never any health data. The team
 * confirms by phone and moves the status here; the optional message is shown to the user.
 */
export function CityServiceBookingsScreen() {
  const t = useTranslations('cityServices.queue');
  const tb = useTranslations('cityServices.dir.booking');
  const locale = useLocale();
  const localize = useLocalized();
  const list = useListParams({ status: 'requested' });
  const status = list.params.filters.status ?? 'requested';
  const options = useCityServiceOptions();
  const query = bookingsApi.useList({ ...list.query, ...(status === 'all' ? {} : { status }) });

  const columns: Column<CityBooking>[] = [
    { key: 'created', header: t('createdAt'), cell: (b) => (b.created_at ? formatDateTime(b.created_at, locale) : '—'), className: 'cell-num whitespace-nowrap' },
    {
      key: 'venue',
      header: t('venue'),
      cell: (b) => (
        <span className="flex flex-col">
          <Link href={`/city-services/${b.listing.id}`} className="text-brand no-underline hover:underline">
            {b.listing.name}
          </Link>
          <span className="text-xs text-muted">{[localize(b.listing.category.name), localize(b.listing.city.name)].filter(Boolean).join(' · ')}</span>
        </span>
      ),
    },
    {
      key: 'contact',
      header: t('contact'),
      cell: (b) => (
        <span className="flex flex-col">
          <span>{b.contact_name}</span>
          <span dir="ltr" className="text-xs text-muted">
            {b.contact_phone}
          </span>
        </span>
      ),
    },
    {
      key: 'request',
      header: t('request'),
      cell: (b) => (
        <span className="flex flex-col text-xs">
          <span>{b.preferred_date}</span>
          {b.code ? <span dir="ltr">{tb('code', { code: b.code })}</span> : null}
          {b.service?.name ? <span className="text-muted">{tb('service', { name: b.service.name })}</span> : null}
          {b.slot?.starts_at ? <span className="text-muted">{tb('slot', { time: formatDateTime(b.slot.starts_at, locale) })}</span> : null}
          {b.child ? <span className="text-muted">{tb('child', { name: b.child.name, months: formatNumber(b.child.age_months, locale) })}</span> : null}
          {b.child_age_range ? <span className="text-muted">{t('age', { age: b.child_age_range })}</span> : null}
          {b.note ? <span className="text-muted">{b.note}</span> : null}
        </span>
      ),
    },
    { key: 'status', header: t('status'), cell: (b) => <Badge tone={TONE[b.status] ?? 'neutral'}>{t(`statusName.${b.status}` as 'statusName.requested')}</Badge> },
    { key: 'actions', header: <span className="sr-only">{t('actions')}</span>, cell: (b) => <MoveCell booking={b} />, className: 'cell-actions' },
  ];

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={t('title')} backHref="/city-services" backLabel={t('backToList')} />
      <Panel bodyClassName="">
        <div className="filter-bar">
          <Select
            aria-label={t('status')}
            value={status}
            onChange={(e) => list.setFilter('status', e.target.value)}
            options={[{ value: 'all', label: t('allStatuses') }, ...(options.data?.statuses ?? []).map((s) => ({ value: s, label: t(`statusName.${s}` as 'statusName.requested') }))]}
          />
        </div>
        <p className="panel-note">{t('privacyNote')}</p>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(b) => b.id}
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

/** The moves the server's status machine allows from the current status, with an optional message for the user. */
function MoveCell({ booking }: { booking: CityBooking }) {
  const t = useTranslations('cityServices.queue');
  const notifyError = useNotifyError();
  const save = bookingsApi.useSave(booking.id);
  const [note, setNote] = useState(booking.status_note ?? '');
  if (booking.allowed_statuses.length === 0) return null;
  const move = (to: string) => save.mutate({ status: to, status_note: note.trim() === '' ? null : note.trim() }, { onSuccess: () => toast.success(t('moved')), onError: notifyError });
  return (
    <div className="flex flex-col gap-2 items-start">
      <TextInput label={t('note')} value={note} onChange={(e) => setNote(e.target.value)} maxLength={300} />
      <div className="row-actions">
        {booking.allowed_statuses.map((to) => (
          <Button key={to} size="sm" variant={to === 'confirmed' || to === 'completed' ? 'primary' : 'default'} loading={save.isPending && save.variables !== undefined} onClick={() => move(to)}>
            {t(`moveTo.${to}` as 'moveTo.confirmed')}
          </Button>
        ))}
      </div>
    </div>
  );
}
