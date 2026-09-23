'use client';

import { useTranslations } from 'next-intl';
import { useRouter, useSearchParams } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { optionLabel, sectionsOf } from '@/shared/lib';
import {
  Button,
  confirm,
  FormPage,
  LoadGate,
  PageHeader,
  Select,
  TextInput,
  toast,
  TranslatableField,
  useNotifyError,
  type Translations,
} from '@/shared/ui';

import { phaseContentsApi, type PhaseContent, type PhaseMap } from '../api/phase-contents';

/** /phase-contents/new?phase= and /phase-contents/:id — the nine sections of one sub-phase. */
export function PhaseContentFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('phaseContents');
  const search = useSearchParams();
  const detail = phaseContentsApi.useDetail(id);
  const map = phaseContentsApi.useList();
  return (
    <LoadGate
      queries={[detail, map]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/phase-contents" backLabel={t('backToList')} />}
    >
      {() =>
        map.data ? (
          <PhaseForm id={id} row={detail.data?.phase_content ?? null} map={map.data} initialPhase={search.get('phase') ?? ''} />
        ) : null
      }
    </LoadGate>
  );
}

function PhaseForm({
  id,
  row,
  map,
  initialPhase,
}: {
  id: number | null;
  row: PhaseContent | null;
  map: PhaseMap;
  initialPhase: string;
}) {
  const t = useTranslations('phaseContents');
  const tc = useTranslations('crud');
  const router = useRouter();
  const notifyError = useNotifyError();
  const save = phaseContentsApi.useSave(id);
  const remove = phaseContentsApi.useRemove();

  const free = map.phases.filter((p) => !map.items.some((i) => i.value === p.value && i.id !== null));
  const [phase, setPhase] = useState(row?.phase ?? (initialPhase || free[0]?.value || map.phases[0]?.value || ''));
  const [sections, setSections] = useState<Record<string, Translations>>(() => sectionsOf(row, map.fields));
  const errors = fieldErrorsOf(save.error);
  const phaseName = optionLabel(map.items, row?.phase ?? phase);
  const label = (field: string) => (t.has(`fields.${field}` as 'fields.sleep') ? t(`fields.${field}` as 'fields.sleep') : field);

  const submit = () =>
    save.mutate(
      { phase, ...sections },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/phase-contents');
        },
        onError: notifyError,
      },
    );

  const onDelete = async () => {
    if (id === null) return;
    if (!(await confirm({ message: t('confirmDelete'), confirmLabel: t('deletePhase'), tone: 'danger' }))) return;
    remove.mutate(id, {
      onSuccess: () => {
        toast.success(tc('deleted'));
        router.replace('/phase-contents');
      },
      onError: notifyError,
    });
  };

  return (
    <FormPage
      title={row ? phaseName : t('new')}
      backHref="/phase-contents"
      backLabel={t('backToList')}
      headerActions={
        id !== null ? (
          <Button variant="danger" onClick={onDelete} loading={remove.isPending}>
            {t('deletePhase')}
          </Button>
        ) : null
      }
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <div className="form-grid">
        {id !== null ? (
          <TextInput label={t('phase')} value={phaseName} readOnly />
        ) : (
          <Select
            label={t('phase')}
            value={phase}
            onChange={(e) => setPhase(e.target.value)}
            options={map.phases}
            required
            error={fieldError(save.error, 'phase')}
          />
        )}
      </div>
      {map.fields.map((field) => (
        <div key={field} className="form-section">
          <TranslatableField
            name={field}
            label={label(field)}
            value={sections[field] ?? {}}
            onChange={(value) => setSections((s) => ({ ...s, [field]: value }))}
            kind="textarea"
            errors={errors}
          />
        </div>
      ))}
    </FormPage>
  );
}
