'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { blankToNull, excerpt, toIntOrNull } from '@/shared/lib';
import {
  CheckboxGrid,
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

import { recommendationsApi, type Recommendation, type RecommendationOptions } from '../api/recommendations';
import { keepReachable, subphasesFor } from '../lib/subphases';

/** /recommendations/new and /recommendations/:id (Blade recommendations.form + subphase-picker). */
export function RecommendationFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('recommendations');
  const detail = recommendationsApi.useDetail(id);
  const options = recommendationsApi.useOptions();
  return (
    <LoadGate
      queries={[detail, options]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/recommendations" backLabel={t('backToList')} />}
    >
      {() =>
        options.data ? <RecommendationForm id={id} row={detail.data?.recommendation ?? null} options={options.data} /> : null
      }
    </LoadGate>
  );
}

function RecommendationForm({
  id,
  row,
  options,
}: {
  id: number | null;
  row: Recommendation | null;
  options: RecommendationOptions;
}) {
  const t = useTranslations('recommendations');
  const tc = useTranslations('crud');
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = recommendationsApi.useSave(id);

  const [type, setType] = useState(row?.type ?? options.types[0]?.value ?? '');
  const [phase, setPhase] = useState(row?.cycle_phase ?? '');
  const [text, setText] = useState<Translations>(row?.text ?? {});
  const [title, setTitle] = useState<Translations>(row?.title ?? {});
  const [subphases, setSubphases] = useState<string[]>(row?.cycle_subphases ?? []);
  const [trigger, setTrigger] = useState(row?.symptom_trigger ?? '');
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? 0));
  const [active, setActive] = useState(row?.is_active ?? true);
  const errors = fieldErrorsOf(save.error);
  const err = (name: string) => fieldError(save.error, name);

  const picker = subphasesFor(options.subphases, options.subphase_phases, phase);
  const onPhase = (next: string) => {
    setPhase(next);
    const p = subphasesFor(options.subphases, options.subphase_phases, next);
    setSubphases((current) => keepReachable(current, p.visible, p.narrowable));
  };

  const submit = () =>
    save.mutate(
      {
        type,
        cycle_phase: blankToNull(phase),
        text,
        title,
        cycle_subphases: picker.narrowable ? subphases : [],
        symptom_trigger: blankToNull(trigger),
        sort_order: toIntOrNull(sortOrder) ?? 0,
        is_active: active,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/recommendations');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('new') : excerpt(localize(row?.text), 60) || t('edit')}
      backHref="/recommendations"
      backLabel={t('backToList')}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <div className="form-grid">
        <Select
          label={t('type')}
          hint={t('typeHint')}
          value={type}
          onChange={(e) => setType(e.target.value)}
          options={options.types}
          required
          error={err('type')}
        />
        <Select
          label={tc('phase')}
          value={phase}
          onChange={(e) => onPhase(e.target.value)}
          options={[{ value: '', label: tc('allPhasesOption') }, ...options.phases]}
          error={err('cycle_phase')}
        />
      </div>
      <TranslatableField name="text" label={t('text')} value={text} onChange={setText} kind="textarea" required maxLength={2000} errors={errors} />
      <div className="flex flex-col gap-1">
        <TranslatableField name="title" label={t('titleOptional')} value={title} onChange={setTitle} maxLength={255} errors={errors} />
        <span className="field-hint">{t('titleHint')}</span>
      </div>
      {picker.narrowable ? (
        <CheckboxGrid
          label={t('subphasesOptional')}
          hint={t('subphasesHint')}
          options={picker.visible}
          value={subphases}
          onChange={setSubphases}
          error={err('cycle_subphases') ?? Object.entries(errors ?? {}).find(([k]) => k.startsWith('cycle_subphases.'))?.[1][0]}
        />
      ) : null}
      <div className="form-grid">
        <Select
          label={t('triggerOptional')}
          hint={t('triggerHint')}
          value={trigger}
          onChange={(e) => setTrigger(e.target.value)}
          options={[{ value: '', label: t('noTrigger') }, ...options.triggers]}
          error={err('symptom_trigger')}
        />
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
