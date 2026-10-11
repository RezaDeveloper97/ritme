'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { formatDateTime, formatNumber, isoToLocalInput, localInputToApi, toIntOrNull, useListParams } from '@/shared/lib';
import {
  Badge,
  Button,
  CrudRowActions,
  DataTable,
  FormPage,
  Icon,
  LoadGate,
  PageHeader,
  Pagination,
  Panel,
  RowActions,
  Select,
  Switch,
  TextInput,
  toast,
  TranslatableField,
  useNotifyError,
  useRowCommands,
  type Column,
  type Translations,
} from '@/shared/ui';

import {
  discountCodesApi,
  layetteApi,
  reviewsApi,
  useCommerceOptions,
  type CommerceOptions,
  type DiscountCode,
  type LayetteItem,
  type Review,
} from '../api/commerce';

import { Money, ShopTabs, StatusBadge } from './ShopTabs';

// ─── reviews ───────────────────────────────────────────────────────────────────────────────────────────────────────

/** /shop/reviews — verified-purchase reviews to moderate; publishing or hiding refreshes the product rating. */
export function ShopReviewsScreen() {
  const t = useTranslations('shopAdmin.reviews');
  const ts = useTranslations('shopAdmin.status');
  const locale = useLocale();
  const localize = useLocalized();
  const list = useListParams({ status: 'pending' });
  const status = list.params.filters.status ?? 'pending';
  const query = reviewsApi.useList(list.query);
  const notifyError = useNotifyError();
  const moderate = reviewsApi.useAction('moderate');
  const commands = useRowCommands({ remove: reviewsApi.useRemove() });
  const set = (id: number, to: string) =>
    moderate.mutate({ id, body: { status: to } }, { onSuccess: () => toast.success(t('moderated')), onError: notifyError });

  const columns: Column<Review>[] = [
    {
      key: 'product',
      header: t('product'),
      cell: (r) => (
        <span className="flex flex-col">
          <Link href={`/shop/${r.product_id}/commerce`} className="text-brand no-underline hover:underline">
            {localize(r.product_title)}
          </Link>
          <span className="text-xs text-muted">{formatDateTime(r.created_at, locale)}</span>
        </span>
      ),
    },
    {
      key: 'review',
      header: t('review'),
      cell: (r) => (
        <span className="flex flex-col gap-1">
          <span>
            {'★'.repeat(r.rating)}
            <span className="text-muted">{'★'.repeat(5 - r.rating)}</span> · {r.author_name}
          </span>
          {r.body ? <span className="text-sm">{r.body}</span> : null}
        </span>
      ),
    },
    { key: 'status', header: t('status'), cell: (r) => <StatusBadge status={r.status} /> },
    {
      key: 'actions',
      header: <span className="sr-only">{t('actions')}</span>,
      cell: (r) => (
        <RowActions>
          {r.status !== 'published' ? (
            <Button size="sm" variant="primary" onClick={() => set(r.id, 'published')}>
              {t('publish')}
            </Button>
          ) : null}
          {r.status !== 'rejected' ? (
            <Button size="sm" onClick={() => set(r.id, 'rejected')}>
              {t('hide')}
            </Button>
          ) : null}
          <Button size="sm" variant="danger" onClick={() => void commands.remove(r.id)}>
            {t('delete')}
          </Button>
        </RowActions>
      ),
      className: 'cell-actions',
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <ShopTabs />
      <Panel title={t('title')} bodyClassName="">
        <div className="filter-bar">
          <Select
            aria-label={t('status')}
            value={status}
            onChange={(e) => list.setFilter('status', e.target.value)}
            options={[{ value: 'all', label: t('allStatuses') }, ...['pending', 'published', 'rejected'].map((s) => ({ value: s, label: ts(s as 'pending') }))]}
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

// ─── discount codes ────────────────────────────────────────────────────────────────────────────────────────────────

/** /shop/discount-codes — shop cart codes (percent / fixed, caps, window, limits). */
export function ShopDiscountCodesScreen() {
  const t = useTranslations('shopAdmin.codes');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const list = useListParams();
  const query = discountCodesApi.useList(list.query);
  const commands = useRowCommands({ remove: discountCodesApi.useRemove() });

  const columns: Column<DiscountCode>[] = [
    { key: 'code', header: t('code'), cell: (d) => <span dir="ltr">{d.code}</span> },
    {
      key: 'value',
      header: t('value'),
      cell: (d) => (d.kind === 'percent' ? t('percentValue', { value: formatNumber(d.value, locale) }) : <Money value={d.value} />),
    },
    {
      key: 'uses',
      header: t('uses'),
      cell: (d) =>
        d.max_uses === null
          ? formatNumber(d.used_count, locale)
          : t('usesOf', { used: formatNumber(d.used_count, locale), max: formatNumber(d.max_uses, locale) }),
    },
    {
      key: 'window',
      header: t('window'),
      cell: (d) => [formatDateTime(d.starts_at, locale), formatDateTime(d.ends_at, locale)].filter(Boolean).join(' → ') || '—',
    },
    {
      key: 'status',
      header: tc('status'),
      cell: (d) => (d.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge tone="amber">{tc('inactive')}</Badge>),
    },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (d) => <CrudRowActions id={d.id} editHref={`/shop/discount-codes/${d.id}`} commands={commands} />,
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
          <Link href="/shop/discount-codes/new" className="btn btn-primary btn-sm">
            <Icon name="plus" size={15} />
            {t('new')}
          </Link>
        }
      >
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(d) => d.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          onRowClick={(d) => router.push(`/shop/discount-codes/${d.id}`)}
          emptyText={t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}

/** /shop/discount-codes/new and /:id. */
export function ShopDiscountCodeFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('shopAdmin.codes');
  const detail = discountCodesApi.useDetail(id);
  return (
    <LoadGate queries={[detail]} header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/shop/discount-codes" backLabel={t('back')} />}>
      {() => <CodeForm id={id} row={detail.data?.discount_code ?? null} />}
    </LoadGate>
  );
}

const numText = (n: number | null | undefined) => (n === null || n === undefined ? '' : String(n));

function CodeForm({ id, row }: { id: number | null; row: DiscountCode | null }) {
  const t = useTranslations('shopAdmin.codes');
  const tc = useTranslations('crud');
  const router = useRouter();
  const notifyError = useNotifyError();
  const save = discountCodesApi.useSave(id);
  const [code, setCode] = useState(row?.code ?? '');
  const [kind, setKind] = useState(row?.kind ?? 'percent');
  const [value, setValue] = useState(numText(row?.value));
  const [maxDiscount, setMaxDiscount] = useState(numText(row?.max_discount));
  const [minTotal, setMinTotal] = useState(numText(row?.min_items_total ?? 0));
  const [maxUses, setMaxUses] = useState(numText(row?.max_uses));
  const [perUser, setPerUser] = useState(numText(row ? row.per_user_limit : 1));
  const [startsAt, setStartsAt] = useState(isoToLocalInput(row?.starts_at ?? null));
  const [endsAt, setEndsAt] = useState(isoToLocalInput(row?.ends_at ?? null));
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (f: string) => fieldError(save.error, f);

  const submit = () =>
    save.mutate(
      {
        code: code.trim(),
        kind,
        value: toIntOrNull(value),
        max_discount: toIntOrNull(maxDiscount),
        min_items_total: toIntOrNull(minTotal),
        max_uses: toIntOrNull(maxUses),
        per_user_limit: toIntOrNull(perUser),
        starts_at: localInputToApi(startsAt),
        ends_at: localInputToApi(endsAt),
        is_active: active,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/shop/discount-codes');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('new') : row?.code ?? t('edit')}
      backHref="/shop/discount-codes"
      backLabel={t('back')}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <div className="form-grid">
        <TextInput label={t('code')} hint={t('codeHint')} value={code} onChange={(e) => setCode(e.target.value.toUpperCase())} dir="ltr" maxLength={32} required error={err('code')} />
        <Select
          label={t('kind')}
          value={kind}
          onChange={(e) => setKind(e.target.value)}
          options={[
            { value: 'percent', label: t('percent') },
            { value: 'fixed', label: t('fixed') },
          ]}
          error={err('kind')}
        />
        <TextInput label={kind === 'percent' ? t('percentField') : t('fixedField')} type="number" value={value} onChange={(e) => setValue(e.target.value)} required error={err('value')} />
        <TextInput label={t('maxDiscount')} hint={t('optional')} type="number" value={maxDiscount} onChange={(e) => setMaxDiscount(e.target.value)} error={err('max_discount')} />
        <TextInput label={t('minTotal')} type="number" value={minTotal} onChange={(e) => setMinTotal(e.target.value)} error={err('min_items_total')} />
        <TextInput label={t('maxUses')} hint={t('optional')} type="number" value={maxUses} onChange={(e) => setMaxUses(e.target.value)} error={err('max_uses')} />
        <TextInput label={t('perUser')} hint={t('optional')} type="number" value={perUser} onChange={(e) => setPerUser(e.target.value)} error={err('per_user_limit')} />
        <TextInput label={t('startsAt')} type="datetime-local" dir="ltr" value={startsAt} onChange={(e) => setStartsAt(e.target.value)} error={err('starts_at')} />
        <TextInput label={t('endsAt')} type="datetime-local" dir="ltr" value={endsAt} onChange={(e) => setEndsAt(e.target.value)} error={err('ends_at')} />
      </div>
      <Switch label={tc('isActive')} checked={active} onChange={setActive} />
    </FormPage>
  );
}

// ─── layette template ──────────────────────────────────────────────────────────────────────────────────────────────

/** /shop/layette — the checklist template, by group (clothing / sleep / feeding / bath / outing). */
export function ShopLayetteScreen() {
  const t = useTranslations('shopAdmin.layette');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const query = layetteApi.useList();
  const commands = useRowCommands({ remove: layetteApi.useRemove(), confirmDelete: t('confirmDelete') });

  const columns: Column<LayetteItem>[] = [
    { key: 'group', header: t('group'), cell: (i) => t(`groups.${i.group_key}` as 'groups.clothing'), className: 'whitespace-nowrap' },
    { key: 'title', header: t('itemTitle'), cell: (i) => localize(i.title) || '—' },
    { key: 'qty', header: t('qty'), cell: (i) => (i.suggested_qty === null ? '—' : formatNumber(i.suggested_qty, locale)), className: 'cell-num' },
    {
      key: 'flags',
      header: t('flags'),
      cell: (i) => (
        <span className="flex flex-wrap gap-1">
          {i.is_essential ? <Badge tone="brand">{t('essential')}</Badge> : null}
          {i.is_first_days ? <Badge tone="data">{t('firstDays')}</Badge> : null}
          {i.is_active ? null : <Badge tone="amber">{tc('inactive')}</Badge>}
        </span>
      ),
    },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (i) => <CrudRowActions id={i.id} editHref={`/shop/layette/${i.id}`} commands={commands} />,
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
          <Link href="/shop/layette/new" className="btn btn-primary btn-sm">
            <Icon name="plus" size={15} />
            {t('new')}
          </Link>
        }
      >
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(i) => i.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          onRowClick={(i) => router.push(`/shop/layette/${i.id}`)}
          emptyText={t('empty')}
          caption={t('title')}
        />
      </Panel>
    </div>
  );
}

/** /shop/layette/new and /:id. */
export function ShopLayetteFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('shopAdmin.layette');
  const detail = layetteApi.useDetail(id);
  const options = useCommerceOptions();
  return (
    <LoadGate queries={[detail, options]} header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/shop/layette" backLabel={t('back')} />}>
      {() => (options.data ? <LayetteForm id={id} row={detail.data?.item ?? null} options={options.data} /> : null)}
    </LoadGate>
  );
}

function LayetteForm({ id, row, options }: { id: number | null; row: LayetteItem | null; options: CommerceOptions }) {
  const t = useTranslations('shopAdmin.layette');
  const tc = useTranslations('crud');
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = layetteApi.useSave(id);
  const [group, setGroup] = useState(row?.group_key ?? options.layette_groups[0] ?? 'clothing');
  const [title, setTitle] = useState<Translations>(row?.title ?? {});
  const [note, setNote] = useState<Translations>(row?.note ?? {});
  const [qty, setQty] = useState(numText(row?.suggested_qty));
  const [categoryId, setCategoryId] = useState(numText(row?.category_id));
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? 0));
  const [essential, setEssential] = useState(row?.is_essential ?? false);
  const [firstDays, setFirstDays] = useState(row?.is_first_days ?? false);
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (f: string) => fieldError(save.error, f);

  const submit = () =>
    save.mutate(
      {
        group_key: group,
        title,
        note,
        suggested_qty: toIntOrNull(qty),
        category_id: toIntOrNull(categoryId),
        sort_order: toIntOrNull(sortOrder) ?? 0,
        is_essential: essential,
        is_first_days: firstDays,
        is_active: active,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/shop/layette');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('new') : localize(row?.title) || t('edit')}
      backHref="/shop/layette"
      backLabel={t('back')}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <div className="form-grid">
        <Select
          label={t('group')}
          value={group}
          onChange={(e) => setGroup(e.target.value)}
          options={options.layette_groups.map((g) => ({ value: g, label: t(`groups.${g}` as 'groups.clothing') }))}
          error={err('group_key')}
        />
        <Select
          label={t('category')}
          hint={t('categoryHint')}
          value={categoryId}
          onChange={(e) => setCategoryId(e.target.value)}
          options={[{ value: '', label: t('noCategory') }, ...options.categories.map((c) => ({ value: String(c.id), label: localize(c.title) || c.slug }))]}
          error={err('category_id')}
        />
        <TextInput label={t('qty')} type="number" value={qty} onChange={(e) => setQty(e.target.value)} error={err('suggested_qty')} />
        <TextInput label={tc('sortOrder')} type="number" value={sortOrder} onChange={(e) => setSortOrder(e.target.value)} error={err('sort_order')} />
      </div>
      <TranslatableField name="title" label={t('itemTitle')} value={title} onChange={setTitle} maxLength={120} required errors={fieldErrorsOf(save.error)} />
      <TranslatableField name="note" label={t('note')} value={note} onChange={setNote} maxLength={300} errors={fieldErrorsOf(save.error)} />
      <Switch label={t('essential')} checked={essential} onChange={setEssential} />
      <Switch label={t('firstDays')} checked={firstDays} onChange={setFirstDays} />
      <Switch label={tc('isActive')} hint={t('activeHint')} checked={active} onChange={setActive} />
    </FormPage>
  );
}
