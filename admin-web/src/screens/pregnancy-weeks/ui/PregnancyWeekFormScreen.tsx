'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRouter, useSearchParams } from 'next/navigation';
import { useEffect, useState } from 'react';

import { fieldError, fieldErrorsOf, isApiError } from '@/shared/api';
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
import { useWeekDetailsList } from '../api/week-details';
import { detailsFallback } from '../lib/cells';
import { WeekDetailsEditor } from './WeekDetailsForm';

/**
 * /pregnancy-weeks/new?week= and /pregnancy-weeks/:id — the ten text modules of one week, and
 * (`?tab=details`) its structured v2 details, keyed by week number (admin-api.md §13).
 */
export function PregnancyWeekFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('pregnancyWeeks');
  const locale = useLocale();
  const router = useRouter();
  const search = useSearchParams();
  const detail = pregnancyWeeksApi.useDetail(id);
  const map = pregnancyWeeksApi.useList();
  const detailsList = useWeekDetailsList();
  // /pregnancy-weeks/:n with no text row n but structured details for week n → open those (QA L7).
  const fallback =
    id === null ? null : detailsFallback(id, isApiError(detail.error) && detail.error.status === 404, detailsList.data?.items);
  useEffect(() => {
    if (fallback) router.replace(fallback);
  }, [fallback, router]);
  const [tab, setTab] = useState<'texts' | 'details'>(search.get('tab') === 'details' ? 'details' : 'texts');
  const fromQuery = Number(search.get('week'));
  const week =
    detail.data?.pregnancy_week.week_number ?? (Number.isInteger(fromQuery) && fromQuery >= 1 && fromQuery <= 42 ? fromQuery : null);
  // The details tab edits one week in place (it upserts), so it is titled by the week, not «new».
  const header = (
    <PageHeader
      title={week !== null ? t('weekTitle', { week: formatNumber(week, locale) }) : id === null ? t('new') : t('edit')}
      backHref="/pregnancy-weeks"
      backLabel={t('backToList')}
    />
  );
  if (fallback) return null;
  return (
    <div className="flex flex-col gap-4">
      <div role="tablist" aria-label={t('tabs')} className="tabs">
        {(['texts', 'details'] as const).map((key) => (
          <button
            key={key}
            type="button"
            role="tab"
            className="tab"
            aria-selected={tab === key}
            aria-current={tab === key ? 'page' : undefined}
            disabled={key === 'details' && week === null}
            title={key === 'details' && week === null ? t('detailsNeedWeek') : undefined}
            onClick={() => setTab(key)}
          >
            {t(key === 'texts' ? 'tabTexts' : 'tabDetails')}
          </button>
        ))}
      </div>
      {tab === 'details' && week !== null ? (
        <WeekDetailsEditor week={week} header={header} />
      ) : (
        <TextsTab id={id} detail={detail} map={map} initialWeek={search.get('week') ?? ''} />
      )}
    </div>
  );
}

function TextsTab({
  id,
  detail,
  map,
  initialWeek,
}: {
  id: number | null;
  detail: ReturnType<typeof pregnancyWeeksApi.useDetail>;
  map: ReturnType<typeof pregnancyWeeksApi.useList>;
  initialWeek: string;
}) {
  const t = useTranslations('pregnancyWeeks');
  return (
    <LoadGate
      queries={[detail, map]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/pregnancy-weeks" backLabel={t('backToList')} />}
    >
      {() => <WeekForm id={id} row={detail.data?.pregnancy_week ?? null} fields={map.data?.fields ?? []} initialWeek={initialWeek} />}
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
    if (
      !(await confirm({
        message: t('confirmDelete'),
        confirmLabel: t('deleteWeek'),
        tone: 'danger',
      }))
    )
      return;
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
