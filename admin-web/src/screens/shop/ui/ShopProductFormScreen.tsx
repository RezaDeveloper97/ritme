'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { blankToNull, formatNumber, toFormData, toIntOrNull } from '@/shared/lib';
import {
  FormPage,
  ImageUpload,
  LoadGate,
  PageHeader,
  Select,
  Switch,
  TextInput,
  toast,
  TranslatableField,
  useNotifyError,
  type Translations,
} from '@/shared/ui';

import { shopProductsApi, type ShopOptions, type ShopProduct } from '../api/shop';

/** /shop/new and /shop/:id — a link-out product (B-N10-03). */
export function ShopProductFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('shop');
  const detail = shopProductsApi.useDetail(id);
  const options = shopProductsApi.useOptions();
  return (
    <LoadGate
      queries={[detail, options]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/shop" backLabel={t('backToList')} />}
    >
      {() => (options.data ? <ProductForm id={id} row={detail.data?.product ?? null} options={options.data} /> : null)}
    </LoadGate>
  );
}

function ProductForm({ id, row, options }: { id: number | null; row: ShopProduct | null; options: ShopOptions }) {
  const t = useTranslations('shop');
  const tc = useTranslations('crud');
  const ta = useTranslations('shopAdmin.commerce'); // CB-SHOP-05
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const localizeCat = (c: ShopOptions['categories'][number]) => localize(c.title) || c.slug;
  const save = shopProductsApi.useSave(id);

  const [image, setImage] = useState<File | null>(null);
  const [removeImage, setRemoveImage] = useState(false);
  const [categoryId, setCategoryId] = useState(String(row?.category_id ?? options.categories[0]?.id ?? ''));
  const [title, setTitle] = useState<Translations>(row?.title ?? {});
  const [priceText, setPriceText] = useState(row?.price_text ?? '');
  const [badge, setBadge] = useState(row?.badge ?? '');
  const [externalUrl, setExternalUrl] = useState(row?.external_url ?? '');
  const [partner, setPartner] = useState(row?.partner_label ?? '');
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? 0));
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (name: string) => fieldError(save.error, name);
  const img = options.image;

  const submit = () =>
    save.mutate(
      toFormData({
        image: image ?? undefined,
        remove_image: removeImage && !image,
        category_id: toIntOrNull(categoryId),
        title,
        price_text: blankToNull(priceText),
        badge: blankToNull(badge),
        external_url: blankToNull(externalUrl),
        partner_label: blankToNull(partner),
        sort_order: toIntOrNull(sortOrder) ?? 0,
        is_active: active,
      }),
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/shop');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('new') : localize(row?.title) || t('edit')}
      backHref="/shop"
      backLabel={t('backToList')}
      onSubmit={submit}
      submitLabel={id === null ? t('create') : tc('saveChanges')}
      saving={save.isPending}
      headerActions={
        // CB-SHOP-05: seller, gallery, variants and stock of this product.
        id !== null ? (
          <Link href={`/shop/${id}/commerce`} className="btn btn-sm">
            {ta('commerceLink')}
          </Link>
        ) : null
      }
    >
      <ImageUpload
        label={t('image')}
        value={image}
        currentUrl={removeImage ? null : row?.image_url}
        onChange={(f) => {
          setImage(f);
          if (f) setRemoveImage(false);
        }}
        maxMb={Math.round(img.max_kb / 1024)}
        accept={img.types.map((type) => `image/${type}`)}
        hint={t('imageHint', {
          minWidth: formatNumber(img.min_width, locale),
          minHeight: formatNumber(img.min_height, locale),
          size: formatNumber(Math.round(img.max_kb / 1024), locale),
        })}
        error={err('image')}
      />
      {row?.image_url && !image ? <Switch label={t('removeImage')} checked={removeImage} onChange={setRemoveImage} /> : null}
      <div className="form-grid">
        <Select
          label={t('category')}
          value={categoryId}
          onChange={(e) => setCategoryId(e.target.value)}
          options={options.categories.map((c) => ({ value: String(c.id), label: localizeCat(c) }))}
          required
          error={err('category_id')}
        />
        <TextInput label={tc('sortOrder')} hint={tc('sortOrderHint')} type="number" value={sortOrder} onChange={(e) => setSortOrder(e.target.value)} error={err('sort_order')} />
      </div>
      <TranslatableField name="title" label={t('titleField')} value={title} onChange={setTitle} maxLength={160} errors={fieldErrorsOf(save.error)} />
      <div className="form-grid">
        <TextInput label={t('priceText')} hint={t('priceTextHint')} value={priceText} onChange={(e) => setPriceText(e.target.value)} maxLength={80} error={err('price_text')} />
        <TextInput label={t('badge')} value={badge} onChange={(e) => setBadge(e.target.value)} maxLength={60} error={err('badge')} />
        <TextInput
          label={t('externalUrl')}
          hint={t('externalUrlHint')}
          value={externalUrl}
          onChange={(e) => setExternalUrl(e.target.value)}
          dir="ltr"
          maxLength={1000}
          error={err('external_url')}
        />
        <TextInput label={t('partnerLabel')} value={partner} onChange={(e) => setPartner(e.target.value)} maxLength={120} error={err('partner_label')} />
        <Switch label={tc('isActive')} checked={active} onChange={setActive} />
      </div>
    </FormPage>
  );
}
