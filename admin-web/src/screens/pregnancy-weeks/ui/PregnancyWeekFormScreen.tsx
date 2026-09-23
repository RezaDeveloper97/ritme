'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRouter, useSearchParams } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { formatNumber, sectionsOf, toIntOrNull } from '@/shared/lib';
import {
  Button,
  confirm,
  FormPage,
  LoadGate,
  PageHeader,
  TextInput,
  toast,
  TranslatableField,
  useNotifyError,
  type Translations,
} from '@/shared/ui';

import { pregnancyWeeksApi, type PregnancyWeek } from '../api/pregnancy-weeks';

/** /pregnancy-weeks/new?week= and /pregnancy-weeks/:id — the ten modules of one week. */
export function PregnancyWeekFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('pregnancyWeeks');
  const search = useSearchParams();
  const detail = pregnancyWeeksApi.useDetail(id);
  const map = pregnancyWeeksApi.useList();
  return (
    <LoadGate
      queries={[detail, map]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/pregnancy-weeks" backLabel={t('backToList')} />}
    >
      {() => (
        <WeekForm
          id={id}
          row={detail.data?.pregnancy_week ?? null}
          fields={map.data?.fields ?? []}
          initialWeek={search.get('week') ?? ''}
        />
      )}
    </LoadGate>
  );
}

function WeekForm({
  id,
  row,
  fields,
  initialWeek,
}: {
  id: number | null;
  row: PregnancyWeek | null;
  fields: string[];
  initialWeek: string;
}) {
  const t = useTranslations('pregnancyWeeks');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const notifyError = useNotifyError();
  const save = pregnancyWeeksApi.useSave(id);
  const remove = pregnancyWeeksApi.useRemove();

  const [week, setWeek] = useState(row ? String(row.week_number) : initialWeek);
  const [sections, setSections] = useState<Record<string, Translations>>(() => sectionsOf(row, fields));
  const errors = fieldErrorsOf(save.error);
  const label = (field: string) => (t.has(`fields.${field}` as 'fields.faq') ? t(`fields.${field}` as 'fields.faq') : field);

  const submit = () =>
    save.mutate(
      { week_number: toIntOrNull(week), ...sections },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/pregnancy-weeks');
        },
        onError: notifyError,
      },
    );

  const onDelete = async () => {
    if (id === null) return;
    if (!(await confirm({ message: t('confirmDelete'), confirmLabel: t('deleteWeek'), tone: 'danger' }))) return;
    remove.mutate(id, {
      onSuccess: () => {
        toast.success(tc('deleted'));
        router.replace('/pregnancy-weeks');
      },
      onError: notifyError,
    });
  };

  return (
    <FormPage
      title={row ? t('weekTitle', { week: formatNumber(row.week_number, locale) }) : t('new')}
      backHref="/pregnancy-weeks"
      backLabel={t('backToList')}
      headerActions={
        id !== null ? (
          <Button variant="danger" onClick={onDelete} loading={remove.isPending}>
            {t('deleteWeek')}
          </Button>
        ) : null
      }
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <div className="form-grid">
        <TextInput
          label={t('weekNumber')}
          hint={id === null ? t('weekNumberHint') : undefined}
          type="number"
          min={1}
          max={42}
          required
          readOnly={id !== null}
          value={week}
          onChange={(e) => setWeek(e.target.value)}
          error={fieldError(save.error, 'week_number')}
        />
      </div>
      {fields.map((field) => (
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
