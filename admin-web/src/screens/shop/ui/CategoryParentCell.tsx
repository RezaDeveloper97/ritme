'use client';

import { useTranslations } from 'next-intl';

import { useLocalized } from '@/shared/i18n';
import { RowActions, Select, toast, useNotifyError } from '@/shared/ui';

import { useCommerceOptions, useSetCategoryParent } from '../api/commerce';

/**
 * CB-SHOP-05: a category's parent (sub-categories under the layette / beauty roots) on bloom's categories list.
 * Only roots are offered; the API refuses a parent for a category that has children of its own.
 */
export function CategoryParentCell({ id }: { id: number }) {
  const t = useTranslations('shopAdmin.categories');
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const options = useCommerceOptions();
  const setParent = useSetCategoryParent();
  const cats = options.data?.categories ?? [];
  const current = cats.find((c) => c.id === id)?.parent_id ?? null;
  const roots = cats.filter((c) => c.parent_id === null && c.id !== id);
  return (
    <RowActions>
      <Select
        aria-label={t('parent')}
        value={current === null ? '' : String(current)}
        disabled={setParent.isPending || !options.data}
        onChange={(e) =>
          setParent.mutate(
            { id, parentId: e.target.value === '' ? null : Number(e.target.value) },
            { onSuccess: () => toast.success(t('parentSaved')), onError: notifyError },
          )
        }
        options={[{ value: '', label: t('root') }, ...roots.map((c) => ({ value: String(c.id), label: localize(c.title) || c.slug }))]}
      />
    </RowActions>
  );
}
