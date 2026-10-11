'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { fieldError } from '@/shared/api';
import { blankToNull, toIntOrNull } from '@/shared/lib';
import { Badge, Button, confirm, DataTable, PageHeader, Panel, Select, Switch, TextInput, toast, useNotifyError, type Column } from '@/shared/ui';

import { CENTRE_TYPES, centresApi, insurersApi, type Centre } from '../api/insurance';

/** /insurance-centers — contracted centres (CRUD). An empty insurer means the centre accepts every insurer. */
export function InsuranceCentresScreen() {
  const t = useTranslations('insuranceCenters');
  const tc = useTranslations('crud');
  const notifyError = useNotifyError();
  const query = centresApi.useList();
  const remove = centresApi.useRemove();
  const [editing, setEditing] = useState<Centre | 'new' | null>(null);

  const columns: Column<Centre>[] = [
    { key: 'name', header: t('name'), cell: (r) => r.name },
    { key: 'type', header: t('type'), cell: (r) => t(`types.${r.type}` as never) },
    { key: 'insurer', header: t('insurer'), cell: (r) => r.insurer_name ?? t('allInsurers') },
    { key: 'city', header: t('city'), cell: (r) => r.city ?? '—' },
    { key: 'direct', header: t('isDirect'), cell: (r) => (r.is_direct ? <Badge tone="green">{t('direct')}</Badge> : '—') },
    { key: 'status', header: tc('status'), cell: (r) => (r.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge tone="amber">{tc('inactive')}</Badge>) },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (r) => (
        <div className="row-actions">
          <Button size="sm" onClick={() => setEditing(r)}>
            {tc('edit')}
          </Button>
          <Button
            size="sm"
            variant="danger"
            onClick={async () => {
              if (!(await confirm({ message: tc('confirmDelete'), confirmLabel: tc('delete'), tone: 'danger' }))) return;
              remove.mutate(r.id, { onSuccess: () => toast.success(tc('deleted')), onError: notifyError });
            }}
          >
            {tc('delete')}
          </Button>
        </div>
      ),
      className: 'cell-actions',
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t('title')}
        actions={
          <div className="row-actions">
            <Link href="/insurance-centers/insurers" className="btn btn-sm">
              {t('insurers')}
            </Link>
            <Button variant="primary" size="sm" onClick={() => setEditing('new')}>
              {t('newCentre')}
            </Button>
          </div>
        }
      />
      {editing ? <CentreForm key={editing === 'new' ? 'new' : editing.id} row={editing === 'new' ? null : editing} types={query.data?.types ?? [...CENTRE_TYPES]} onDone={() => setEditing(null)} /> : null}
      <Panel bodyClassName="">
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(r) => r.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={t('emptyCentres')}
          caption={t('title')}
        />
      </Panel>
    </div>
  );
}

function CentreForm({ row, types, onDone }: { row: Centre | null; types: string[]; onDone: () => void }) {
  const t = useTranslations('insuranceCenters');
  const tc = useTranslations('crud');
  const notifyError = useNotifyError();
  const insurers = insurersApi.useList();
  const save = centresApi.useSave(row?.id ?? null);
  const [name, setName] = useState(row?.name ?? '');
  const [type, setType] = useState(row?.type ?? types[0] ?? 'other');
  const [insurerId, setInsurerId] = useState(row?.insurer_id ? String(row.insurer_id) : '');
  const [services, setServices] = useState(row?.services ?? '');
  const [city, setCity] = useState(row?.city ?? '');
  const [address, setAddress] = useState(row?.address ?? '');
  const [phone, setPhone] = useState(row?.phone ?? '');
  const [lat, setLat] = useState(row?.lat ?? '');
  const [lng, setLng] = useState(row?.lng ?? '');
  const [direct, setDirect] = useState(row?.is_direct ?? false);
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? 0));
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (n: string) => fieldError(save.error, n);

  const submit = () =>
    save.mutate(
      {
        name: name.trim(),
        type,
        insurer_id: toIntOrNull(insurerId),
        services: blankToNull(services),
        city: blankToNull(city),
        address: blankToNull(address),
        phone: blankToNull(phone),
        lat: blankToNull(lat),
        lng: blankToNull(lng),
        is_direct: direct,
        is_active: active,
        sort_order: toIntOrNull(sortOrder) ?? 0,
      },
      {
        onSuccess: () => {
          toast.success(row ? tc('saved') : tc('created'));
          onDone();
        },
        onError: notifyError,
      },
    );

  return (
    <Panel title={row ? tc('edit') : tc('create')}>
      <form
        className="flex flex-col gap-3"
        onSubmit={(e) => {
          e.preventDefault();
          submit();
        }}
      >
        <div className="form-grid">
          <TextInput label={t('name')} value={name} onChange={(e) => setName(e.target.value)} maxLength={160} required error={err('name')} />
          <Select label={t('type')} value={type} onChange={(e) => setType(e.target.value)} options={types.map((v) => ({ value: v, label: t(`types.${v}` as never) }))} error={err('type')} />
          <Select
            label={t('insurer')}
            hint={t('insurerHint')}
            value={insurerId}
            onChange={(e) => setInsurerId(e.target.value)}
            options={[{ value: '', label: t('allInsurers') }, ...(insurers.data?.items ?? []).map((i) => ({ value: String(i.id), label: i.name }))]}
            error={err('insurer_id')}
          />
          <TextInput label={t('services')} value={services} onChange={(e) => setServices(e.target.value)} maxLength={160} error={err('services')} />
          <TextInput label={t('city')} value={city} onChange={(e) => setCity(e.target.value)} maxLength={80} error={err('city')} />
          <TextInput label={t('phone')} value={phone} onChange={(e) => setPhone(e.target.value)} dir="ltr" maxLength={32} error={err('phone')} />
          <TextInput label={t('address')} value={address} onChange={(e) => setAddress(e.target.value)} maxLength={300} error={err('address')} />
          <TextInput label={t('lat')} hint={t('coordHint')} type="number" step="any" value={lat} onChange={(e) => setLat(e.target.value)} dir="ltr" error={err('lat')} />
          <TextInput label={t('lng')} type="number" step="any" value={lng} onChange={(e) => setLng(e.target.value)} dir="ltr" error={err('lng')} />
          <TextInput label={tc('sortOrder')} hint={tc('sortOrderHint')} type="number" value={sortOrder} onChange={(e) => setSortOrder(e.target.value)} error={err('sort_order')} />
          <Switch label={t('isDirect')} checked={direct} onChange={setDirect} />
          <Switch label={tc('isActive')} checked={active} onChange={setActive} />
        </div>
        <div className="row-actions">
          <Button type="submit" variant="primary" loading={save.isPending}>
            {tc('saveChanges')}
          </Button>
          <Button onClick={onDone}>{tc('cancel')}</Button>
        </div>
      </form>
    </Panel>
  );
}
