'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { formatNumber, toIntOrNull } from '@/shared/lib';
import {
  Badge,
  CrudRowActions,
  DataTable,
  FormPage,
  Icon,
  LoadGate,
  PageHeader,
  Panel,
  Switch,
  TextInput,
  toast,
  TranslatableField,
  useNotifyError,
  useRowCommands,
  type Column,
  type Translations,
} from '@/shared/ui';

import { sellersApi, type Seller } from '../api/commerce';

import { Money, ShopTabs } from './ShopTabs';

/** /shop/sellers — the internal shop's sellers: shipping terms, COD ceiling, return window (CB-SHOP-05). */
export function ShopSellersScreen() {
  const t = useTranslations('shopAdmin.sellers');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const query = sellersApi.useList();
  const commands = useRowCommands({ remove: sellersApi.useRemove() });

  const columns: Column<Seller>[] = [
    { key: 'name', header: t('name'), cell: (s) => localize(s.name) || s.slug },
    { key: 'ship', header: t('shipDays'), cell: (s) => t('days', { min: formatNumber(s.ship_min_days, locale), max: formatNumber(s.ship_max_days, locale) }) },
    { key: 'fee', header: t('shippingFee'), cell: (s) => <Money value={s.shipping_fee} /> },
    { key: 'cod', header: t('codMax'), cell: (s) => <Money value={s.cod_max_total} /> },
    { key: 'products', header: t('products'), cell: (s) => formatNumber(s.products_count, locale), className: 'cell-num' },
    {
      key: 'status',
      header: tc('status'),
      cell: (s) => (
        <span className="flex gap-1">
          {s.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge tone="amber">{tc('inactive')}</Badge>}
          {s.is_verified ? <Badge tone="brand">{t('verified')}</Badge> : null}
        </span>
      ),
    },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (s) => <CrudRowActions id={s.id} editHref={`/shop/sellers/${s.id}`} commands={commands} canDelete={s.products_count === 0} />,
      className: 'cell-actions',
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <ShopTabs />
      <Panel
        title={t('title')}
        bodyClassName=""
        actions={
          <Link href="/shop/sellers/new" className="btn btn-primary btn-sm">
            <Icon name="plus" size={15} />
            {t('new')}
          </Link>
        }
      >
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(s) => s.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          onRowClick={(s) => router.push(`/shop/sellers/${s.id}`)}
          emptyText={t('empty')}
          caption={t('title')}
        />
      </Panel>
    </div>
  );
}

/** /shop/sellers/new and /shop/sellers/:id. */
export function ShopSellerFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('shopAdmin.sellers');
  const detail = sellersApi.useDetail(id);
  return (
    <LoadGate queries={[detail]} header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/shop/sellers" backLabel={t('back')} />}>
      {() => <SellerForm id={id} row={detail.data?.seller ?? null} />}
    </LoadGate>
  );
}

function SellerForm({ id, row }: { id: number | null; row: Seller | null }) {
  const t = useTranslations('shopAdmin.sellers');
  const tc = useTranslations('crud');
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = sellersApi.useSave(id);
  const [name, setName] = useState<Translations>(row?.name ?? {});
  const [slug, setSlug] = useState(row?.slug ?? '');
  const [minDays, setMinDays] = useState(String(row?.ship_min_days ?? 1));
  const [maxDays, setMaxDays] = useState(String(row?.ship_max_days ?? 3));
  const [fee, setFee] = useState(String(row?.shipping_fee ?? 0));
  const [freeOver, setFreeOver] = useState(row?.free_shipping_over === null || row === null ? '' : String(row.free_shipping_over));
  const [codMax, setCodMax] = useState(row?.cod_max_total === null || row === null ? '' : String(row.cod_max_total));
  const [returnDays, setReturnDays] = useState(String(row?.return_days ?? 7));
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? 0));
  const [verified, setVerified] = useState(row?.is_verified ?? false);
  const [discreet, setDiscreet] = useState(row?.discreet_packaging ?? false);
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (f: string) => fieldError(save.error, f);

  const submit = () =>
    save.mutate(
      {
        name,
        slug: slug.trim(),
        ship_min_days: toIntOrNull(minDays),
        ship_max_days: toIntOrNull(maxDays),
        shipping_fee: toIntOrNull(fee),
        free_shipping_over: toIntOrNull(freeOver),
        cod_max_total: toIntOrNull(codMax),
        return_days: toIntOrNull(returnDays),
        sort_order: toIntOrNull(sortOrder) ?? 0,
        is_verified: verified,
        discreet_packaging: discreet,
        is_active: active,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/shop/sellers');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('new') : localize(row?.name) || t('edit')}
      backHref="/shop/sellers"
      backLabel={t('back')}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <TranslatableField name="name" label={t('name')} value={name} onChange={setName} maxLength={120} errors={fieldErrorsOf(save.error)} />
      <div className="form-grid">
        <TextInput label={t('slug')} hint={t('slugHint')} value={slug} onChange={(e) => setSlug(e.target.value)} dir="ltr" maxLength={64} required error={err('slug')} />
        <TextInput label={tc('sortOrder')} type="number" value={sortOrder} onChange={(e) => setSortOrder(e.target.value)} error={err('sort_order')} />
        <TextInput label={t('shipMin')} type="number" value={minDays} onChange={(e) => setMinDays(e.target.value)} required error={err('ship_min_days')} />
        <TextInput label={t('shipMax')} type="number" value={maxDays} onChange={(e) => setMaxDays(e.target.value)} required error={err('ship_max_days')} />
        <TextInput label={t('shippingFee')} hint={t('tomanHint')} type="number" value={fee} onChange={(e) => setFee(e.target.value)} required error={err('shipping_fee')} />
        <TextInput label={t('freeOver')} hint={t('optionalToman')} type="number" value={freeOver} onChange={(e) => setFreeOver(e.target.value)} error={err('free_shipping_over')} />
        <TextInput label={t('codMax')} hint={t('codMaxHint')} type="number" value={codMax} onChange={(e) => setCodMax(e.target.value)} error={err('cod_max_total')} />
        <TextInput label={t('returnDays')} type="number" value={returnDays} onChange={(e) => setReturnDays(e.target.value)} required error={err('return_days')} />
      </div>
      <Switch label={t('verified')} checked={verified} onChange={setVerified} />
      <Switch label={t('discreet')} hint={t('discreetHint')} checked={discreet} onChange={setDiscreet} />
      <Switch label={tc('isActive')} checked={active} onChange={setActive} />
    </FormPage>
  );
}
