'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { blankToNull, formatNumber, toIntOrNull } from '@/shared/lib';
import {
  Badge,
  Button,
  confirm,
  DataTable,
  PageHeader,
  Panel,
  Switch,
  TextInput,
  toast,
  TranslatableField,
  useNotifyError,
  type Column,
  type Translations,
} from '@/shared/ui';

import { categoriesApi, citiesApi, type Taxon } from '../api/city-services';

/**
 * /city-services/cities and /city-services/categories — a short admin-managed list with an inline add / edit form.
 * A city or category that listings use cannot be deleted (422 in_use): deactivate it instead.
 */
export function CityServiceTaxaScreen({ kind }: { kind: 'cities' | 'categories' }) {
  const api = kind === 'cities' ? citiesApi : categoriesApi;
  const t = useTranslations('cityServices');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const query = api.useList();
  const remove = api.useRemove();
  const [editing, setEditing] = useState<Taxon | 'new' | null>(null);

  const columns: Column<Taxon>[] = [
    { key: 'slug', header: t('slug'), cell: (r) => <span dir="ltr">{r.slug}</span> },
    { key: 'name', header: t('name'), cell: (r) => localize(r.name) || '—' },
    ...(kind === 'categories' ? [{ key: 'icon', header: t('icon'), cell: (r: Taxon) => r.icon ?? '—' } satisfies Column<Taxon>] : []),
    { key: 'count', header: t('listingsCount'), cell: (r) => formatNumber(r.listings_count, locale), className: 'cell-num' },
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
        title={kind === 'cities' ? t('cities') : t('categories')}
        backHref="/city-services"
        backLabel={t('backToList')}
        actions={
          <Button variant="primary" size="sm" onClick={() => setEditing('new')}>
            {kind === 'cities' ? t('newCity') : t('newCategory')}
          </Button>
        }
      />
      {editing ? <TaxonForm key={editing === 'new' ? 'new' : editing.id} kind={kind} row={editing === 'new' ? null : editing} onDone={() => setEditing(null)} /> : null}
      <Panel bodyClassName="">
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(r) => r.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={t('emptyTaxa')}
          caption={kind === 'cities' ? t('cities') : t('categories')}
        />
      </Panel>
    </div>
  );
}

function TaxonForm({ kind, row, onDone }: { kind: 'cities' | 'categories'; row: Taxon | null; onDone: () => void }) {
  const api = kind === 'cities' ? citiesApi : categoriesApi;
  const t = useTranslations('cityServices');
  const tc = useTranslations('crud');
  const notifyError = useNotifyError();
  const save = api.useSave(row?.id ?? null);
  const [slug, setSlug] = useState(row?.slug ?? '');
  const [name, setName] = useState<Translations>(row?.name ?? {});
  const [icon, setIcon] = useState(row?.icon ?? '');
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? 0));
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (n: string) => fieldError(save.error, n);

  const submit = () =>
    save.mutate(
      { slug: slug.trim(), name, ...(kind === 'categories' ? { icon: blankToNull(icon) } : {}), sort_order: toIntOrNull(sortOrder) ?? 0, is_active: active },
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
        <TranslatableField name="name" label={t('name')} value={name} onChange={setName} maxLength={120} errors={fieldErrorsOf(save.error)} />
        <div className="form-grid">
          <TextInput label={t('slug')} hint={t('slugHint')} value={slug} onChange={(e) => setSlug(e.target.value)} dir="ltr" maxLength={64} required error={err('slug')} />
          {kind === 'categories' ? <TextInput label={t('icon')} hint={t('iconHint')} value={icon} onChange={(e) => setIcon(e.target.value)} dir="ltr" maxLength={32} error={err('icon')} /> : null}
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
