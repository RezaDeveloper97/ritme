'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { toIntOrNull } from '@/shared/lib';
import {
  FormPage,
  LoadGate,
  PageHeader,
  Select,
  Switch,
  TextInput,
  TranslatableField,
  toast,
  useNotifyError,
  type Translations,
} from '@/shared/ui';

import { careItemsApi, type CareItem, type CareOptions } from '../api/care-items';
import { cleanTranslations } from '../lib/order';
import { useKindLabel } from './labels';

/** /pregnancy-care-plan/new and /pregnancy-care-plan/:id. */
export function CareItemFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('pregnancyCarePlan');
  const detail = careItemsApi.useDetail(id);
  const options = careItemsApi.useOptions();
  return (
    <LoadGate
      queries={id === null ? [options] : [detail, options]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/pregnancy-care-plan" backLabel={t('backToList')} />}
    >
      {() => (options.data ? <CareItemForm id={id} row={detail.data?.care_item ?? null} options={options.data} /> : null)}
    </LoadGate>
  );
}

const num = (v: number | null | undefined) => (v === null || v === undefined ? '' : String(v));

function CareItemForm({ id, row, options }: { id: number | null; row: CareItem | null; options: CareOptions }) {
  const t = useTranslations('pregnancyCarePlan');
  const tc = useTranslations('crud');
  const router = useRouter();
  const localize = useLocalized();
  const kindLabel = useKindLabel();
  const notifyError = useNotifyError();
  const save = careItemsApi.useSave(id);

  const [key, setKey] = useState(row?.key ?? '');
  const [title, setTitle] = useState<Translations>(row?.title ?? {});
  const [prep, setPrep] = useState<Translations>(row?.prep ?? {});
  const [kind, setKind] = useState(row?.kind ?? options.kinds[0] ?? 'visit');
  const [weekFrom, setWeekFrom] = useState(num(row?.week_from));
  const [weekTo, setWeekTo] = useState(num(row?.week_to));
  const [remind, setRemind] = useState(num(row?.remind_before ?? options.default_remind_before));
  const [sortOrder, setSortOrder] = useState(num(row?.sort_order ?? options.next_sort_order));
  const [active, setActive] = useState(row?.is_active ?? true);
  const errors = fieldErrorsOf(save.error);
  const err = (name: string) => fieldError(save.error, name);

  const submit = () =>
    save.mutate(
      {
        ...(id === null ? { key: key.trim() } : {}),
        title,
        prep: cleanTranslations(prep),
        kind,
        week_from: toIntOrNull(weekFrom),
        week_to: toIntOrNull(weekTo),
        remind_before: toIntOrNull(remind),
        sort_order: toIntOrNull(sortOrder) ?? undefined,
        is_active: active,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/pregnancy-care-plan');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('new') : localize(row?.title) || t('edit')}
      backHref="/pregnancy-care-plan"
      backLabel={t('backToList')}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <TextInput
        label={t('key')}
        hint={id === null ? t('keyHint') : t('keyFixed')}
        value={key}
        onChange={(e) => setKey(e.target.value)}
        dir="ltr"
        maxLength={64}
        pattern="[a-z][a-z0-9_]*"
        required={id === null}
        disabled={id !== null}
        error={err('key')}
      />
      <TranslatableField name="title" label={t('titleField')} value={title} onChange={setTitle} required maxLength={255} errors={errors} />
      <TranslatableField name="prep" label={t('prep')} value={prep} onChange={setPrep} kind="textarea" maxLength={2000} errors={errors} />
      <div className="form-grid">
        <Select
          label={t('kind')}
          value={kind}
          onChange={(e) => setKind(e.target.value)}
          options={options.kinds.map((k) => ({
            value: k,
            label: kindLabel(k),
          }))}
          required
          error={err('kind')}
        />
        <TextInput
          label={t('remindBefore')}
          hint={t('remindBeforeHint')}
          type="number"
          min={0}
          max={options.max_remind_before}
          value={remind}
          onChange={(e) => setRemind(e.target.value)}
          error={err('remind_before')}
        />
        <TextInput
          label={t('weekFrom')}
          type="number"
          min={options.min_week}
          max={options.max_week}
          required
          value={weekFrom}
          onChange={(e) => setWeekFrom(e.target.value)}
          error={err('week_from')}
        />
        <TextInput
          label={t('weekTo')}
          type="number"
          min={options.min_week}
          max={options.max_week}
          required
          value={weekTo}
          onChange={(e) => setWeekTo(e.target.value)}
          error={err('week_to')}
        />
        <TextInput
          label={tc('sortOrder')}
          hint={tc('sortOrderHint')}
          type="number"
          value={sortOrder}
          onChange={(e) => setSortOrder(e.target.value)}
          error={err('sort_order')}
        />
      </div>
      <Switch label={tc('isActive')} checked={active} onChange={setActive} />
      {row && row.appointments_count > 0 ? <p className="field-hint m-0">{t('inUseHint', { count: row.appointments_count })}</p> : null}
    </FormPage>
  );
}
