'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf, type Option } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { blankToNull, toIntOrNull } from '@/shared/lib';
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

import { taskTemplatesApi, type TaskTemplate } from '../api/task-templates';

/** /task-templates/new and /task-templates/:id (Blade task-templates.form). */
export function TaskTemplateFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('taskTemplates');
  const detail = taskTemplatesApi.useDetail(id);
  const options = taskTemplatesApi.useOptions();
  return (
    <LoadGate
      queries={[detail, options]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/task-templates" backLabel={t('backToList')} />}
    >
      {() => (
        <TaskTemplateForm
          id={id}
          row={detail.data?.task_template ?? null}
          phases={options.data?.phases ?? []}
          categories={options.data?.categories ?? []}
        />
      )}
    </LoadGate>
  );
}

function TaskTemplateForm({
  id,
  row,
  phases,
  categories,
}: {
  id: number | null;
  row: TaskTemplate | null;
  phases: Option[];
  categories: Option[];
}) {
  const t = useTranslations('taskTemplates');
  const tc = useTranslations('crud');
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = taskTemplatesApi.useSave(id);

  const [key, setKey] = useState(row?.key ?? '');
  const [category, setCategory] = useState(row?.category ?? categories[0]?.value ?? '');
  const [title, setTitle] = useState<Translations>(row?.title ?? {});
  const [description, setDescription] = useState<Translations>(row?.description ?? {});
  const [phase, setPhase] = useState(row?.cycle_phase ?? '');
  const [icon, setIcon] = useState(row?.icon ?? '');
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? 0));
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (name: string) => fieldError(save.error, name);

  const submit = () =>
    save.mutate(
      {
        key: key.trim(),
        category,
        title,
        description,
        cycle_phase: blankToNull(phase),
        icon: blankToNull(icon),
        sort_order: toIntOrNull(sortOrder) ?? 0,
        is_active: active,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/task-templates');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('new') : localize(row?.title) || t('edit')}
      backHref="/task-templates"
      backLabel={t('backToList')}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <div className="form-grid">
        <TextInput
          label={t('key')}
          hint={t('keyHint')}
          value={key}
          onChange={(e) => setKey(e.target.value)}
          required
          dir="ltr"
          error={err('key')}
        />
        <Select label={tc('category')} value={category} onChange={(e) => setCategory(e.target.value)} options={categories} required error={err('category')} />
      </div>
      <TranslatableField name="title" label={t('titleField')} value={title} onChange={setTitle} required errors={fieldErrorsOf(save.error)} />
      <TranslatableField
        name="description"
        label={t('description')}
        value={description}
        onChange={setDescription}
        kind="textarea"
        errors={fieldErrorsOf(save.error)}
      />
      <div className="form-grid">
        <Select
          label={tc('phase')}
          value={phase}
          onChange={(e) => setPhase(e.target.value)}
          options={[{ value: '', label: tc('allPhasesOption') }, ...phases]}
          error={err('cycle_phase')}
        />
        <TextInput label={t('icon')} hint={t('iconHint')} value={icon} onChange={(e) => setIcon(e.target.value)} dir="ltr" error={err('icon')} />
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
