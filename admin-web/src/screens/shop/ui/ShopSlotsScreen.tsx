'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { fieldError } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { formatDate, formatNumber, toIntOrNull, useListParams } from '@/shared/lib';
import {
  Badge,
  Button,
  CheckboxGrid,
  DataTable,
  Panel,
  RowActions,
  Select,
  TextInput,
  toast,
  useNotifyError,
  useRowCommands,
  type Column,
} from '@/shared/ui';

import { slotsApi, useCommerceOptions, useGenerateSlots, type Slot } from '../api/commerce';

import { ShopTabs } from './ShopTabs';

// Saturday first (the Iranian week); values are JS weekdays (0 = Sunday), as the API takes them.
const WEEKDAYS = ['6', '0', '1', '2', '3', '4', '5'] as const;

/**
 * /shop/slots — a seller's dated delivery windows (CB-SHOP-05). The template generates the chosen weekdays × windows
 * over a date range (existing slots are kept); a booked slot can be shrunk to its bookings or deactivated, not deleted.
 */
export function ShopSlotsScreen() {
  const t = useTranslations('shopAdmin.slots');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const localize = useLocalized();
  const options = useCommerceOptions();
  const list = useListParams({ seller_id: '' });
  const sellerId = list.params.filters.seller_id ?? '';
  const query = slotsApi.useList({ seller_id: sellerId, from: list.params.filters.from, to: list.params.filters.to }, { enabled: sellerId !== '' });
  const commands = useRowCommands({ remove: slotsApi.useRemove() });

  const columns: Column<Slot>[] = [
    { key: 'date', header: t('date'), cell: (s) => formatDate(s.slot_date, locale), className: 'whitespace-nowrap' },
    { key: 'window', header: t('window'), cell: (s) => <span dir="ltr">{`${s.starts_at}–${s.ends_at}`}</span> },
    {
      key: 'capacity',
      header: t('booked'),
      cell: (s) =>
        s.capacity === null
          ? t('bookedOpen', { booked: formatNumber(s.booked, locale) })
          : t('bookedOf', { booked: formatNumber(s.booked, locale), capacity: formatNumber(s.capacity, locale) }),
    },
    {
      key: 'status',
      header: tc('status'),
      cell: (s) => (s.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge tone="amber">{tc('inactive')}</Badge>),
    },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (s) => <SlotActions slot={s} onDelete={() => void commands.remove(s.id)} />,
      className: 'cell-actions',
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <ShopTabs />
      <Panel title={t('title')} bodyClassName="">
        <div className="filter-bar">
          <Select
            aria-label={t('seller')}
            value={sellerId}
            onChange={(e) => list.setFilter('seller_id', e.target.value)}
            options={[{ value: '', label: t('chooseSeller') }, ...(options.data?.sellers ?? []).map((s) => ({ value: String(s.id), label: localize(s.name) || s.slug }))]}
          />
          <TextInput aria-label={t('from')} type="date" dir="ltr" value={list.params.filters.from ?? ''} onChange={(e) => list.setFilter('from', e.target.value)} />
          <TextInput aria-label={t('to')} type="date" dir="ltr" value={list.params.filters.to ?? ''} onChange={(e) => list.setFilter('to', e.target.value)} />
        </div>
        {sellerId === '' ? (
          <p className="panel-note">{t('pickSeller')}</p>
        ) : (
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
        )}
      </Panel>
      {sellerId !== '' ? <GenerateForm sellerId={Number(sellerId)} /> : null}
    </div>
  );
}

function SlotActions({ slot, onDelete }: { slot: Slot; onDelete: () => void }) {
  const t = useTranslations('shopAdmin.slots');
  const tc = useTranslations('crud');
  const notifyError = useNotifyError();
  const save = slotsApi.useSave(slot.id);
  const [capacity, setCapacity] = useState(slot.capacity === null ? '' : String(slot.capacity));
  const put = (body: { capacity: number | null; is_active: boolean }) =>
    save.mutate(body, { onSuccess: () => toast.success(tc('saved')), onError: notifyError });
  return (
    <RowActions>
      <TextInput aria-label={t('capacity')} type="number" value={capacity} onChange={(e) => setCapacity(e.target.value)} className="w-20" placeholder="∞" />
      <Button size="sm" loading={save.isPending} onClick={() => put({ capacity: toIntOrNull(capacity), is_active: slot.is_active })}>
        {tc('saveChanges')}
      </Button>
      <Button size="sm" variant="ghost" onClick={() => put({ capacity: slot.capacity, is_active: !slot.is_active })}>
        {slot.is_active ? tc('deactivate') : tc('activate')}
      </Button>
      {slot.booked === 0 ? (
        <Button size="sm" variant="danger" onClick={onDelete}>
          {tc('delete')}
        </Button>
      ) : null}
    </RowActions>
  );
}

function GenerateForm({ sellerId }: { sellerId: number }) {
  const t = useTranslations('shopAdmin.slots');
  const notifyError = useNotifyError();
  const generate = useGenerateSlots();
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [days, setDays] = useState<string[]>([...WEEKDAYS]);
  const [windows, setWindows] = useState([{ starts_at: '09:00', ends_at: '13:00', capacity: '' }]);
  const err = (f: string) => fieldError(generate.error, f);
  const setWindow = (i: number, patch: Partial<(typeof windows)[number]>) =>
    setWindows((ws) => ws.map((w, j) => (j === i ? { ...w, ...patch } : w)));

  const submit = () =>
    generate.mutate(
      {
        seller_id: sellerId,
        from,
        to,
        weekdays: days.map(Number),
        windows: windows.map((w) => ({ starts_at: w.starts_at, ends_at: w.ends_at, capacity: toIntOrNull(w.capacity) })),
      },
      { onSuccess: (res) => toast.success(t('generated', { count: res.created })), onError: notifyError },
    );

  return (
    <Panel title={t('template')}>
      <div className="flex flex-col gap-3">
        <div className="form-grid">
          <TextInput label={t('from')} type="date" dir="ltr" value={from} onChange={(e) => setFrom(e.target.value)} required error={err('from')} />
          <TextInput label={t('to')} type="date" dir="ltr" value={to} onChange={(e) => setTo(e.target.value)} required error={err('to')} />
        </div>
        <CheckboxGrid
          label={t('weekdays')}
          options={WEEKDAYS.map((d) => ({ value: d, label: t(`weekday.${d}` as 'weekday.0') }))}
          value={days}
          onChange={setDays}
        />
        {windows.map((w, i) => (
          <div key={i} className="form-grid">
            <TextInput label={t('startsAt')} type="time" dir="ltr" value={w.starts_at} onChange={(e) => setWindow(i, { starts_at: e.target.value })} />
            <TextInput label={t('endsAt')} type="time" dir="ltr" value={w.ends_at} onChange={(e) => setWindow(i, { ends_at: e.target.value })} />
            <TextInput label={t('capacity')} hint={t('capacityHint')} type="number" value={w.capacity} onChange={(e) => setWindow(i, { capacity: e.target.value })} />
          </div>
        ))}
        {err('windows') ? <p className="field-error">{err('windows')}</p> : null}
        <div className="row-actions">
          {windows.length < 6 ? (
            <Button size="sm" variant="ghost" onClick={() => setWindows((ws) => [...ws, { starts_at: '17:00', ends_at: '21:00', capacity: '' }])}>
              {t('addWindow')}
            </Button>
          ) : null}
          {windows.length > 1 ? (
            <Button size="sm" variant="ghost" onClick={() => setWindows((ws) => ws.slice(0, -1))}>
              {t('removeWindow')}
            </Button>
          ) : null}
          <Button variant="primary" loading={generate.isPending} onClick={submit}>
            {t('generate')}
          </Button>
        </div>
      </div>
    </Panel>
  );
}
