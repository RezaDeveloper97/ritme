'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { blankToNull, formatNumber, toIntOrNull } from '@/shared/lib';
import {
  Badge,
  Button,
  CheckboxGrid,
  DataTable,
  FormPage,
  ImageUpload,
  LoadGate,
  PageHeader,
  Panel,
  Select,
  Switch,
  TextArea,
  TextInput,
  toast,
  TranslatableField,
  useNotifyError,
  useRowCommands,
  RowActions,
  type Column,
  type Translations,
} from '@/shared/ui';

import {
  useCommerce,
  useCommerceOptions,
  useCreateVariant,
  useSaveCommerce,
  useUploadImage,
  useVariants,
  variantsApi,
  type Commerce,
  type CommerceOptions,
  type Variant,
} from '../api/commerce';

import { Money } from './ShopTabs';

/**
 * /shop/:id/commerce — the internal-shop side of a bloom product (CB-SHOP-05): seller, subtitle / description,
 * attributes (material, wash, size guide, ingredients… as JSON), IRC code, expiry, skin types, badges, plain
 * packaging, the gallery and the variants with price and stock. A product with no active variant stays link-out.
 */
export function ShopCommerceScreen({ id }: { id: number }) {
  const t = useTranslations('shopAdmin.commerce');
  const commerce = useCommerce(id);
  const options = useCommerceOptions();
  return (
    <LoadGate queries={[commerce, options]} header={<PageHeader title={t('title')} backHref={`/shop/${id}`} backLabel={t('back')} />}>
      {() =>
        commerce.data && options.data ? (
          <div className="flex flex-col gap-4">
            <CommerceForm row={commerce.data.product} options={options.data} />
            <Gallery row={commerce.data.product} options={options.data} />
            <Variants productId={id} />
          </div>
        ) : null
      }
    </LoadGate>
  );
}

const jsonText = (v: unknown) => (v && (Array.isArray(v) ? v.length > 0 : Object.keys(v).length > 0) ? JSON.stringify(v, null, 2) : '');

function parseJson(text: string): { ok: true; value: unknown } | { ok: false } {
  if (text.trim() === '') return { ok: true, value: null };
  try {
    return { ok: true, value: JSON.parse(text) as unknown };
  } catch {
    return { ok: false };
  }
}

function CommerceForm({ row, options }: { row: Commerce; options: CommerceOptions }) {
  const t = useTranslations('shopAdmin.commerce');
  const tc = useTranslations('crud');
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = useSaveCommerce(row.id);
  const [sellerId, setSellerId] = useState(row.seller_id === null ? '' : String(row.seller_id));
  const [subtitle, setSubtitle] = useState<Translations>(row.subtitle);
  const [description, setDescription] = useState<Translations>(row.description);
  const [attributes, setAttributes] = useState(jsonText(row.attributes));
  const [badges, setBadges] = useState(jsonText(row.badges));
  const [irc, setIrc] = useState(row.irc_code ?? '');
  const [expires, setExpires] = useState(row.expires_on ?? '');
  const [skins, setSkins] = useState<string[]>(row.skin_types);
  const [discreet, setDiscreet] = useState(row.discreet_packaging);
  const [jsonError, setJsonError] = useState<string | null>(null);
  const err = (f: string) => fieldError(save.error, f);

  const submit = () => {
    const a = parseJson(attributes);
    const b = parseJson(badges);
    if (!a.ok || !b.ok) {
      setJsonError(!a.ok ? 'attributes' : 'badges');
      return;
    }
    setJsonError(null);
    save.mutate(
      {
        seller_id: toIntOrNull(sellerId),
        subtitle,
        description,
        attributes: a.value,
        badges: b.value,
        irc_code: blankToNull(irc),
        expires_on: blankToNull(expires),
        skin_types: skins,
        discreet_packaging: discreet,
      },
      { onSuccess: () => toast.success(tc('saved')), onError: notifyError },
    );
  };

  return (
    <FormPage
      title={localize(row.title) || t('title')}
      backHref={`/shop/${row.id}`}
      backLabel={t('back')}
      meta={
        <span className="flex gap-1">
          {row.is_link_out ? <Badge tone="amber">{t('linkOut')}</Badge> : <Badge tone="green">{t('internal')}</Badge>}
          <Badge tone="neutral">{t('rating', { avg: row.rating_avg ?? '—', count: row.rating_count })}</Badge>
        </span>
      }
      onSubmit={submit}
      submitLabel={tc('saveChanges')}
      saving={save.isPending}
    >
      <div className="form-grid">
        <Select
          label={t('seller')}
          value={sellerId}
          onChange={(e) => setSellerId(e.target.value)}
          options={[{ value: '', label: t('noSeller') }, ...options.sellers.map((s) => ({ value: String(s.id), label: localize(s.name) || s.slug }))]}
          error={err('seller_id')}
        />
        <TextInput label={t('irc')} value={irc} onChange={(e) => setIrc(e.target.value)} dir="ltr" maxLength={40} error={err('irc_code')} />
        <TextInput label={t('expires')} type="date" value={expires} onChange={(e) => setExpires(e.target.value)} dir="ltr" error={err('expires_on')} />
      </div>
      <TranslatableField name="subtitle" label={t('subtitle')} value={subtitle} onChange={setSubtitle} maxLength={200} errors={fieldErrorsOf(save.error)} />
      <TranslatableField name="description" kind="textarea" label={t('description')} value={description} onChange={setDescription} maxLength={5000} errors={fieldErrorsOf(save.error)} />
      <TextArea
        label={t('attributes')}
        hint={t('attributesHint')}
        value={attributes}
        onChange={(e) => setAttributes(e.target.value)}
        dir="ltr"
        rows={6}
        error={jsonError === 'attributes' ? t('jsonInvalid') : err('attributes')}
      />
      <TextArea
        label={t('badges')}
        hint={t('badgesHint')}
        value={badges}
        onChange={(e) => setBadges(e.target.value)}
        dir="ltr"
        rows={3}
        error={jsonError === 'badges' ? t('jsonInvalid') : err('badges')}
      />
      <CheckboxGrid
        label={t('skinTypes')}
        options={options.skin_types.map((s) => ({ value: s, label: t(`skin.${s}` as 'skin.oily') }))}
        value={skins}
        onChange={setSkins}
      />
      <Switch label={t('discreet')} hint={t('discreetHint')} checked={discreet} onChange={setDiscreet} />
    </FormPage>
  );
}

function Gallery({ row, options }: { row: Commerce; options: CommerceOptions }) {
  const t = useTranslations('shopAdmin.commerce');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const upload = useUploadImage(row.id);
  const save = useSaveCommerce(row.id);
  const [file, setFile] = useState<File | null>(null);
  const img = options.image;
  const paths = row.images.map((i) => i.path);
  // Reorder / remove re-send the full commerce body (PUT is a full replace of these fields).
  const resend = (images: string[]) =>
    save.mutate(
      {
        seller_id: row.seller_id,
        subtitle: row.subtitle,
        description: row.description,
        attributes: row.attributes,
        badges: row.badges,
        irc_code: row.irc_code,
        expires_on: row.expires_on,
        skin_types: row.skin_types,
        discreet_packaging: row.discreet_packaging,
        images,
      },
      { onError: notifyError },
    );

  return (
    <Panel title={t('gallery')}>
      <div className="flex flex-wrap gap-3">
        {row.images.map((image, index) => (
          <figure key={image.path} className="flex flex-col items-center gap-1">
            {/* eslint-disable-next-line @next/next/no-img-element -- API-hosted upload */}
            <img src={image.url} alt="" className="thumb" loading="lazy" />
            <span className="row-actions">
              <Button size="sm" variant="ghost" disabled={index === 0 || save.isPending} onClick={() => resend([image.path, ...paths.filter((p) => p !== image.path)])}>
                {t('makeFirst')}
              </Button>
              <Button size="sm" variant="danger" disabled={save.isPending} onClick={() => resend(paths.filter((p) => p !== image.path))}>
                {t('remove')}
              </Button>
            </span>
          </figure>
        ))}
        {row.images.length === 0 ? <p className="text-muted">{t('noImages')}</p> : null}
      </div>
      {row.images.length < img.max_count ? (
        <div className="mt-4 flex flex-col gap-2">
          <ImageUpload
            label={t('addImage')}
            value={file}
            onChange={setFile}
            maxMb={Math.round(img.max_kb / 1024)}
            accept={img.types.map((type) => `image/${type}`)}
            hint={t('imageHint', {
              minWidth: formatNumber(img.min_width, locale),
              minHeight: formatNumber(img.min_height, locale),
              count: formatNumber(img.max_count, locale),
            })}
            error={fieldError(upload.error, 'image')}
          />
          <div>
            <Button
              variant="primary"
              size="sm"
              disabled={!file}
              loading={upload.isPending}
              onClick={() => file && upload.mutate(file, { onSuccess: () => setFile(null), onError: notifyError })}
            >
              {t('upload')}
            </Button>
          </div>
        </div>
      ) : null}
    </Panel>
  );
}

function Variants({ productId }: { productId: number }) {
  const t = useTranslations('shopAdmin.variants');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const localize = useLocalized();
  const query = useVariants(productId);
  const commands = useRowCommands({ remove: variantsApi.useRemove() });
  const [editing, setEditing] = useState<Variant | 'new' | null>(null);

  const columns: Column<Variant>[] = [
    { key: 'sku', header: t('sku'), cell: (v) => <span dir="ltr">{v.sku ?? '—'}</span> },
    {
      key: 'label',
      header: t('label'),
      cell: (v) => [localize(v.size), localize(v.color), localize(v.volume)].filter(Boolean).join(' · ') || '—',
    },
    { key: 'price', header: t('price'), cell: (v) => <Money value={v.price} /> },
    { key: 'stock', header: t('stock'), cell: (v) => formatNumber(v.stock, locale), className: 'cell-num' },
    {
      key: 'status',
      header: tc('status'),
      cell: (v) => (v.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge tone="amber">{tc('inactive')}</Badge>),
    },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (v) => (
        <RowActions>
          <Button size="sm" variant="ghost" onClick={() => setEditing(v)}>
            {tc('edit')}
          </Button>
          <Button size="sm" variant="danger" onClick={() => void commands.remove(v.id)}>
            {tc('delete')}
          </Button>
        </RowActions>
      ),
      className: 'cell-actions',
    },
  ];

  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        <Button size="sm" variant="primary" onClick={() => setEditing('new')}>
          {t('new')}
        </Button>
      }
    >
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(v) => v.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        emptyText={t('empty')}
        caption={t('title')}
      />
      {editing ? (
        <VariantForm key={editing === 'new' ? 'new' : editing.id} productId={productId} row={editing === 'new' ? null : editing} onDone={() => setEditing(null)} />
      ) : null}
    </Panel>
  );
}

function VariantForm({ productId, row, onDone }: { productId: number; row: Variant | null; onDone: () => void }) {
  const t = useTranslations('shopAdmin.variants');
  const tc = useTranslations('crud');
  const notifyError = useNotifyError();
  const create = useCreateVariant(productId);
  const update = variantsApi.useSave(row?.id ?? 0);
  const save = row ? update : create;
  const [sku, setSku] = useState(row?.sku ?? '');
  const [size, setSize] = useState<Translations>(row?.size ?? {});
  const [color, setColor] = useState<Translations>(row?.color ?? {});
  const [volume, setVolume] = useState<Translations>(row?.volume ?? {});
  const [price, setPrice] = useState(row ? String(row.price) : '');
  const [compare, setCompare] = useState(row?.compare_at_price === null || !row ? '' : String(row.compare_at_price));
  const [stock, setStock] = useState(String(row?.stock ?? 0));
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? 0));
  const [active, setActive] = useState(row?.is_active ?? true);
  const errors = fieldErrorsOf(save.error);
  const err = (f: string) => fieldError(save.error, f);

  const submit = () =>
    save.mutate(
      {
        sku: blankToNull(sku),
        size,
        color,
        volume,
        price: toIntOrNull(price),
        compare_at_price: toIntOrNull(compare),
        stock: toIntOrNull(stock),
        sort_order: toIntOrNull(sortOrder) ?? 0,
        is_active: active,
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
    <div className="flex flex-col gap-3 border-t border-line p-4">
      <h3 className="font-semibold">{row ? t('edit') : t('new')}</h3>
      <div className="form-grid">
        <TextInput label={t('sku')} value={sku} onChange={(e) => setSku(e.target.value)} dir="ltr" maxLength={64} error={err('sku')} />
        <TextInput label={t('price')} hint={t('tomanHint')} type="number" value={price} onChange={(e) => setPrice(e.target.value)} required error={err('price')} />
        <TextInput label={t('compareAt')} type="number" value={compare} onChange={(e) => setCompare(e.target.value)} error={err('compare_at_price')} />
        <TextInput label={t('stock')} type="number" value={stock} onChange={(e) => setStock(e.target.value)} required error={err('stock')} />
        <TextInput label={tc('sortOrder')} type="number" value={sortOrder} onChange={(e) => setSortOrder(e.target.value)} error={err('sort_order')} />
      </div>
      <TranslatableField name="size" label={t('size')} value={size} onChange={setSize} maxLength={60} errors={errors} />
      <TranslatableField name="color" label={t('color')} value={color} onChange={setColor} maxLength={60} errors={errors} />
      <TranslatableField name="volume" label={t('volume')} value={volume} onChange={setVolume} maxLength={60} errors={errors} />
      <Switch label={tc('isActive')} checked={active} onChange={setActive} />
      <div className="row-actions">
        <Button variant="primary" loading={save.isPending} onClick={submit}>
          {row ? tc('saveChanges') : tc('create')}
        </Button>
        <Button variant="ghost" onClick={onDone}>
          {tc('cancel')}
        </Button>
      </div>
    </div>
  );
}
