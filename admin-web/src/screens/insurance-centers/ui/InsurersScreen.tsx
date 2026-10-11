'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { fieldError } from '@/shared/api';
import { blankToNull, toIntOrNull } from '@/shared/lib';
import { Badge, Button, confirm, DataTable, PageHeader, Panel, Switch, TextInput, toast, useNotifyError, type Column } from '@/shared/ui';

import { insurersApi, type Insurer } from '../api/insurance';

/** /insurance-centers/insurers — the insurer catalog. Deleting empties the link on policies and centres; deactivate to keep the name. */
export function InsurersScreen() {
  const t = useTranslations('insuranceCenters');
  const tc = useTranslations('crud');
  const notifyError = useNotifyError();
  const query = insurersApi.useList();
  const remove = insurersApi.useRemove();
  const [editing, setEditing] = useState<Insurer | 'new' | null>(null);

  const columns: Column<Insurer>[] = [
    { key: 'code', header: t('code'), cell: (r) => <span dir="ltr">{r.code}</span> },
    { key: 'name', header: t('name'), cell: (r) => r.name },
    { key: 'phone', header: t('phone'), cell: (r) => <span dir="ltr">{r.phone ?? '—'}</span> },
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
              if (!(await confirm({ message: t('confirmDeleteInsurer'), confirmLabel: tc('delete'), tone: 'danger' }))) return;
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
        title={t('insurers')}
        backHref="/insurance-centers"
        backLabel={t('backToCentres')}
        actions={
          <Button variant="primary" size="sm" onClick={() => setEditing('new')}>
            {t('newInsurer')}
          </Button>
        }
      />
      {editing ? <InsurerForm key={editing === 'new' ? 'new' : editing.id} row={editing === 'new' ? null : editing} onDone={() => setEditing(null)} /> : null}
      <Panel bodyClassName="">
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(r) => r.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={t('emptyInsurers')}
          caption={t('insurers')}
        />
      </Panel>
    </div>
  );
}

function InsurerForm({ row, onDone }: { row: Insurer | null; onDone: () => void }) {
  const t = useTranslations('insuranceCenters');
  const tc = useTranslations('crud');
  const notifyError = useNotifyError();
  const save = insurersApi.useSave(row?.id ?? null);
  const [code, setCode] = useState(row?.code ?? '');
  const [name, setName] = useState(row?.name ?? '');
  const [website, setWebsite] = useState(row?.website ?? '');
  const [phone, setPhone] = useState(row?.phone ?? '');
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? 0));
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (n: string) => fieldError(save.error, n);

  const submit = () =>
    save.mutate(
      { code: code.trim(), name: name.trim(), website: blankToNull(website), phone: blankToNull(phone), sort_order: toIntOrNull(sortOrder) ?? 0, is_active: active },
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
          <TextInput label={t('name')} value={name} onChange={(e) => setName(e.target.value)} maxLength={120} required error={err('name')} />
          <TextInput label={t('code')} hint={t('codeHint')} value={code} onChange={(e) => setCode(e.target.value)} dir="ltr" maxLength={40} required error={err('code')} />
          <TextInput label={t('website')} type="url" value={website} onChange={(e) => setWebsite(e.target.value)} dir="ltr" maxLength={255} error={err('website')} />
          <TextInput label={t('phone')} value={phone} onChange={(e) => setPhone(e.target.value)} dir="ltr" maxLength={32} error={err('phone')} />
          <TextInput label={tc('sortOrder')} hint={tc('sortOrderHint')} type="number" value={sortOrder} onChange={(e) => setSortOrder(e.target.value)} error={err('sort_order')} />
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
