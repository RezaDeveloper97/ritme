'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { blankToNull, toIntOrNull } from '@/shared/lib';
import { FormPage, LoadGate, PageHeader, Switch, TextInput, toast, TranslatableField, useNotifyError, type Translations } from '@/shared/ui';

import { shopCategoriesApi, type ShopCategory } from '../api/shop';

/** /shop/categories/new and /shop/categories/:id. */
export function ShopCategoryFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('shop');
  const detail = shopCategoriesApi.useDetail(id);
  return (
    <LoadGate
      queries={[detail]}
      header={<PageHeader title={id === null ? t('newCategory') : t('editCategory')} backHref="/shop/categories" backLabel={t('backToCategories')} />}
    >
      {() => <CategoryForm id={id} row={detail.data?.category ?? null} />}
    </LoadGate>
  );
}

function CategoryForm({ id, row }: { id: number | null; row: ShopCategory | null }) {
  const t = useTranslations('shop');
  const tc = useTranslations('crud');
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = shopCategoriesApi.useSave(id);
  const [slug, setSlug] = useState(row?.slug ?? '');
  const [title, setTitle] = useState<Translations>(row?.title ?? {});
  const [icon, setIcon] = useState(row?.icon ?? '');
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? 0));
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (name: string) => fieldError(save.error, name);

  const submit = () =>
    save.mutate(
      { slug: slug.trim(), title, icon: blankToNull(icon), sort_order: toIntOrNull(sortOrder) ?? 0, is_active: active },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/shop/categories');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('newCategory') : localize(row?.title) || t('editCategory')}
      backHref="/shop/categories"
      backLabel={t('backToCategories')}
      onSubmit={submit}
      submitLabel={id === null ? t('createCategory') : tc('saveChanges')}
      saving={save.isPending}
    >
      <TranslatableField name="title" label={t('titleField')} value={title} onChange={setTitle} maxLength={120} errors={fieldErrorsOf(save.error)} />
      <div className="form-grid">
        <TextInput label={t('slug')} hint={t('slugHint')} value={slug} onChange={(e) => setSlug(e.target.value)} dir="ltr" maxLength={64} required error={err('slug')} />
        <TextInput label={t('icon')} hint={t('iconHint')} value={icon} onChange={(e) => setIcon(e.target.value)} dir="ltr" maxLength={32} error={err('icon')} />
        <TextInput label={tc('sortOrder')} hint={tc('sortOrderHint')} type="number" value={sortOrder} onChange={(e) => setSortOrder(e.target.value)} error={err('sort_order')} />
        <Switch label={tc('isActive')} checked={active} onChange={setActive} />
      </div>
    </FormPage>
  );
}
