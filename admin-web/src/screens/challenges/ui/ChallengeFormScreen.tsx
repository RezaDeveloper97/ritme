'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { blankToNull, formatNumber, toIntOrNull } from '@/shared/lib';
import {
  FormPage,
  LoadGate,
  PageHeader,
  Switch,
  TextInput,
  toast,
  TranslatableField,
  useNotifyError,
  type Translations,
} from '@/shared/ui';

import { challengesApi, type Challenge } from '../api/challenges';

/** /challenges/new and /challenges/:id (Blade challenges.form + cycle-day-range). */
export function ChallengeFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('challenges');
  const detail = challengesApi.useDetail(id);
  const options = challengesApi.useOptions();
  return (
    <LoadGate
      queries={[detail, options]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/challenges" backLabel={t('backToList')} />}
    >
      {() => <ChallengeForm id={id} row={detail.data?.challenge ?? null} maxDay={options.data?.max_cycle_day ?? 35} />}
    </LoadGate>
  );
}

function ChallengeForm({ id, row, maxDay }: { id: number | null; row: Challenge | null; maxDay: number }) {
  const t = useTranslations('challenges');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = challengesApi.useSave(id);

  const [title, setTitle] = useState<Translations>(row?.title ?? {});
  const [description, setDescription] = useState<Translations>(row?.description ?? {});
  const [dayFrom, setDayFrom] = useState(row?.cycle_day_from?.toString() ?? '');
  const [dayTo, setDayTo] = useState(row?.cycle_day_to?.toString() ?? '');
  const [category, setCategory] = useState(row?.category ?? '');
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? 0));
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (name: string) => fieldError(save.error, name);

  const submit = () =>
    save.mutate(
      {
        title,
        description,
        cycle_day_from: toIntOrNull(dayFrom),
        cycle_day_to: toIntOrNull(dayTo),
        category: blankToNull(category),
        sort_order: toIntOrNull(sortOrder) ?? 0,
        is_active: active,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/challenges');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('new') : localize(row?.title) || t('edit')}
      backHref="/challenges"
      backLabel={t('backToList')}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <TranslatableField name="title" label={t('titleField')} value={title} onChange={setTitle} required maxLength={255} errors={fieldErrorsOf(save.error)} />
      <TranslatableField
        name="description"
        label={t('description')}
        value={description}
        onChange={setDescription}
        kind="textarea"
        errors={fieldErrorsOf(save.error)}
      />
      <fieldset className="field m-0 min-w-0 border-0 p-0">
        <legend className="field-label mb-1.5 p-0">{t('cycleDays')}</legend>
        <div className="form-grid">
          <TextInput
            label={t('dayFrom')}
            type="number"
            min={1}
            max={maxDay}
            placeholder={t('dayFromPlaceholder')}
            value={dayFrom}
            onChange={(e) => setDayFrom(e.target.value)}
            error={err('cycle_day_from')}
          />
          <TextInput
            label={t('dayTo')}
            type="number"
            min={1}
            max={maxDay}
            placeholder={t('dayToPlaceholder')}
            value={dayTo}
            onChange={(e) => setDayTo(e.target.value)}
            error={err('cycle_day_to')}
          />
        </div>
        <span className="field-hint">{t('cycleDaysHint', { max: formatNumber(maxDay, locale) })}</span>
      </fieldset>
      <div className="form-grid">
        <TextInput label={tc('category')} value={category} onChange={(e) => setCategory(e.target.value)} maxLength={255} error={err('category')} />
        <TextInput
          label={tc('sortOrder')}
          hint={tc('sortOrderHint')}
          type="number"
          value={sortOrder}
          onChange={(e) => setSortOrder(e.target.value)}
          error={err('sort_order')}
        />
        <Switch label={tc('isActive')} checked={active} onChange={setActive} />
      </div>
    </FormPage>
  );
}
