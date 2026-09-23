'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf, type Option } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { blankToNull, excerpt, toIntOrNull } from '@/shared/lib';
import {
  FormPage,
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

import { affirmationsApi, type Affirmation } from '../api/affirmations';

/** /affirmations/new and /affirmations/:id (Blade affirmations.form). */
export function AffirmationFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('affirmations');
  const detail = affirmationsApi.useDetail(id);
  const options = affirmationsApi.useOptions();
  return (
    <LoadGate
      queries={[detail, options]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/affirmations" backLabel={t('backToList')} />}
    >
      {() => <AffirmationForm id={id} row={detail.data?.affirmation ?? null} phases={options.data?.phases ?? []} />}
    </LoadGate>
  );
}

function AffirmationForm({ id, row, phases }: { id: number | null; row: Affirmation | null; phases: Option[] }) {
  const t = useTranslations('affirmations');
  const tc = useTranslations('crud');
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = affirmationsApi.useSave(id);

  const [text, setText] = useState<Translations>(row?.text ?? {});
  const [phase, setPhase] = useState(row?.cycle_phase ?? '');
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? 0));
  const [active, setActive] = useState(row?.is_active ?? true);

  const submit = () =>
    save.mutate(
      { text, cycle_phase: blankToNull(phase), sort_order: toIntOrNull(sortOrder) ?? 0, is_active: active },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/affirmations');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('new') : excerpt(localize(row?.text), 60) || t('edit')}
      backHref="/affirmations"
      backLabel={t('backToList')}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <TranslatableField
        name="text"
        label={t('text')}
        value={text}
        onChange={setText}
        kind="textarea"
        required
        errors={fieldErrorsOf(save.error)}
      />
      <div className="form-grid">
        <Select
          label={tc('phase')}
          value={phase}
          onChange={(e) => setPhase(e.target.value)}
          options={[{ value: '', label: tc('allPhasesOption') }, ...phases]}
          error={fieldError(save.error, 'cycle_phase')}
        />
        <TextInput
          label={tc('sortOrder')}
          hint={tc('sortOrderHint')}
          type="number"
          value={sortOrder}
          onChange={(e) => setSortOrder(e.target.value)}
          error={fieldError(save.error, 'sort_order')}
        />
        <Switch label={tc('isActive')} checked={active} onChange={setActive} />
      </div>
    </FormPage>
  );
}
