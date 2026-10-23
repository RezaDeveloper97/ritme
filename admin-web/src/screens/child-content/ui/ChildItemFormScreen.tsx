'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { useNumber } from '@/shared/lib';
import {
  Badge,
  FormPage,
  LoadGate,
  PageHeader,
  Select,
  Switch,
  TextArea,
  TextInput,
  TranslatableField,
  toast,
  useNotifyError,
  type Translations,
} from '@/shared/ui';

import Link from 'next/link';

import { useChildItem, useSaveChildItem, useVaccineSchedule, type ChildItem } from '../api/child-content';
import {
  DOMAINS,
  KINDS,
  MAX_MONTHS,
  TOPICS,
  approxDays,
  buildMeta,
  draftOf,
  type ChildKind,
  type MetaDraft,
  type MetaErrors,
} from '../lib/meta';
import { useAgeLabel, useCodeLabels } from './labels';
import { ReadOnlyNotice, useCanWrite } from './parts';

const CODE_RE = /^[a-z][a-z0-9_]*$/;

/** Prefill of a new item from the list page (`?month=` / `?visit=&age=`). */
export interface ChildItemPrefill {
  month?: number;
  visit?: string;
  age?: number;
}

function backHref(kind: ChildKind, month: number | null): string {
  const page = KINDS[kind].page;
  return page === 'milestones' && month !== null ? `/children-content/milestones?month=${month}` : `/children-content/${page}`;
}

/** /children-content/:kind/new and /children-content/:kind/:id. */
export function ChildItemFormScreen({ kind, id, prefill = {} }: { kind: ChildKind; id: number | null; prefill?: ChildItemPrefill }) {
  const t = useTranslations('childContent');
  const detail = useChildItem(KINDS[kind].group, id);
  return (
    <LoadGate
      queries={[detail]}
      header={<PageHeader title={t(`kinds.${kind}`)} backHref={backHref(kind, prefill.month ?? null)} backLabel={t('back')} />}
    >
      {() => <ChildItemForm kind={kind} id={id} row={detail.data?.catalog_item ?? null} prefill={prefill} />}
    </LoadGate>
  );
}

function ChildItemForm({ kind, id, row, prefill }: { kind: ChildKind; id: number | null; row: ChildItem | null; prefill: ChildItemPrefill }) {
  const t = useTranslations('childContent');
  const tc = useTranslations('crud');
  const n = useNumber();
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const labels = useCodeLabels();
  const canWrite = useCanWrite();
  const group = KINDS[kind].group;
  const save = useSaveChildItem(group, id);

  const initial = (): MetaDraft => {
    const d = draftOf(row?.meta);
    if (row) return d;
    const age = prefill.age ?? prefill.month;
    return { ...d, visit: prefill.visit ?? '', ageMonths: age === undefined ? '' : String(age), minutes: kind === 'learn' ? '3' : '' };
  };
  const [code, setCode] = useState(row?.code ?? '');
  const [title, setTitle] = useState<Translations>(row?.title ?? {});
  const [body, setBody] = useState<Translations>(row?.body ?? {});
  const [draft, setDraft] = useState<MetaDraft>(initial);
  const [active, setActive] = useState(row?.is_active ?? true);
  const [reviewed, setReviewed] = useState(row ? !row.needs_review : false);
  const [codeError, setCodeError] = useState<string | undefined>();
  const [metaErrors, setMetaErrors] = useState<MetaErrors>({});

  const set = <K extends keyof MetaDraft>(key: K, value: MetaDraft[K]) => setDraft((d) => ({ ...d, [key]: value }));
  const apiErrors = fieldErrorsOf(save.error);
  const metaMsg = (key: keyof MetaDraft) => (metaErrors[key] ? t(`metaErrors.${metaErrors[key]}`) : undefined);
  const ageMonths = /^\d+$/.test(draft.ageMonths) ? Number(draft.ageMonths) : null;
  const back = backHref(kind, kind === 'learn' || kind === 'vaccines' ? null : ageMonths);

  const submit = () => {
    const c = code.trim();
    const badCode = id === null && (!CODE_RE.test(c) || c.length > 64);
    setCodeError(badCode ? t('codeInvalid') : undefined);
    const built = buildMeta(kind, draft, row?.meta ?? null);
    setMetaErrors(built.ok ? {} : built.errors);
    if (badCode || !built.ok) return;
    const clean = (v: Translations) => Object.fromEntries(Object.entries(v).filter(([, s]) => s.trim()).map(([k, s]) => [k, s.trim()]));
    const cleanBody = clean(body);
    save.mutate(
      {
        ...(id === null ? { code: c } : {}),
        title: clean(title),
        body: Object.keys(cleanBody).length ? cleanBody : null,
        meta: built.meta,
        is_active: active,
        needs_review: !reviewed,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push(back);
        },
        onError: notifyError,
      },
    );
  };

  const monthsInput = (key: 'ageMonths' | 'fromMonths' | 'toMonths', label: string, hint?: string) => (
    <TextInput
      type="number"
      label={label}
      hint={hint}
      min={0}
      max={MAX_MONTHS}
      required
      value={draft[key]}
      onChange={(e) => set(key, e.target.value)}
      error={metaMsg(key)}
      disabled={!canWrite}
    />
  );

  return (
    <FormPage
      title={id === null ? t(`newKind.${kind}`) : localize(row?.title) || row?.code || t(`kinds.${kind}`)}
      meta={
        <>
          <span>{t(`kinds.${kind}`)}</span>
          <span dir="ltr" className="cell-mono">
            {group}
          </span>
          {row ? row.needs_review ? <Badge tone="amber">{t('needsReview')}</Badge> : <Badge tone="green">{t('reviewed')}</Badge> : null}
        </>
      }
      backHref={back}
      backLabel={t('back')}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
      readOnly={!canWrite}
    >
      {canWrite ? null : <ReadOnlyNotice />}
      <p className="field-hint m-0">{t('clinicalHint')}</p>
      <TextInput
        label={t('fields.code')}
        hint={id === null ? t('codeHint') : t('codeFixed')}
        value={code}
        onChange={(e) => setCode(e.target.value)}
        dir="ltr"
        maxLength={64}
        required={id === null}
        disabled={id !== null || !canWrite}
        error={codeError ?? fieldError(save.error, 'code')}
      />
      <TranslatableField
        name="title"
        label={kind === 'vaccines' ? t('fields.doseName') : t('fields.title')}
        value={title}
        onChange={setTitle}
        required
        maxLength={255}
        errors={apiErrors}
      />
      <TranslatableField
        name="body"
        label={t(`bodyLabel.${kind}`)}
        kind="textarea"
        value={body}
        onChange={setBody}
        maxLength={5000}
        errors={apiErrors}
      />

      {kind === 'vaccines' ? <ScheduleAge code={row?.code ?? code.trim()} /> : null}

      {kind === 'vaccines' ? null : kind !== 'learn' ? (
        <div className="form-grid">
          {monthsInput(
            'ageMonths',
            t('fields.ageMonths'),
            ageMonths !== null ? t('ageDaysHint', { days: n(approxDays(ageMonths)) }) : t('ageMonthsHint'),
          )}
          {kind === 'milestones' ? (
            <Select
              label={t('fields.domain')}
              required
              value={draft.domain}
              disabled={!canWrite}
              onChange={(e) => set('domain', e.target.value)}
              error={metaMsg('domain')}
              options={[{ value: '', label: t('chooseDomain') }, ...DOMAINS.map((d) => ({ value: d, label: labels.domain(d) }))]}
            />
          ) : null}
        </div>
      ) : (
        <>
          <div className="form-grid">
            <Select
              label={t('fields.topic')}
              required
              value={draft.topic}
              disabled={!canWrite}
              onChange={(e) => set('topic', e.target.value)}
              error={metaMsg('topic')}
              options={[{ value: '', label: t('chooseTopic') }, ...TOPICS.map((k) => ({ value: k, label: labels.topic(k) }))]}
            />
            {monthsInput('fromMonths', t('fields.fromMonths'))}
            {monthsInput('toMonths', t('fields.toMonths'))}
            <TextInput
              type="number"
              label={t('fields.minutes')}
              min={0}
              max={240}
              value={draft.minutes}
              onChange={(e) => set('minutes', e.target.value)}
              error={metaMsg('minutes')}
              disabled={!canWrite}
            />
          </div>
          <TextInput
            label={t('fields.articleSlug')}
            hint={t('articleSlugHint')}
            dir="ltr"
            maxLength={191}
            value={draft.articleSlug}
            onChange={(e) => set('articleSlug', e.target.value)}
            error={metaMsg('articleSlug')}
            disabled={!canWrite}
          />
          <Switch label={t('fields.featured')} hint={t('featuredHint')} checked={draft.featured} onChange={(v) => set('featured', v)} disabled={!canWrite} />
        </>
      )}

      {kind === 'vaccines' ? (
        <TextArea
          label={t('fields.adminNote')}
          hint={t('adminNoteHint')}
          rows={3}
          maxLength={1000}
          value={draft.adminNote}
          onChange={(e) => set('adminNote', e.target.value)}
          disabled={!canWrite}
        />
      ) : null}

      <Switch label={tc('isActive')} hint={t('activeHint')} checked={active} onChange={setActive} disabled={!canWrite} />
      <Switch label={t('fields.reviewed')} hint={t('reviewedHint')} checked={reviewed} onChange={setReviewed} disabled={!canWrite} />
    </FormPage>
  );
}

/**
 * A dose's due age, read-only: the clinical registry key child.vaccine_schedule in force today owns it (T-M10-22,
 * D-173, H-17), so it is changed through a new registry version with two medical approvals, not here.
 */
function ScheduleAge({ code }: { code: string }) {
  const t = useTranslations('childContent');
  const tcc = useTranslations('clinicalConfig');
  const n = useNumber();
  const ageLabel = useAgeLabel();
  const schedule = useVaccineSchedule();
  const age = code ? schedule.data?.ages.get(code) : undefined;
  return (
    <div className="flex flex-wrap items-center gap-2 text-sm" role="group" aria-label={t('fields.ageMonths')}>
      <strong>{t('fields.ageMonths')}</strong>
      {schedule.data ? (
        age !== undefined ? (
          <Badge tone="data">
            {ageLabel(age)}
            {age > 0 ? ` · ${t('approxDays', { days: n(approxDays(age)) })}` : null}
          </Badge>
        ) : (
          <Badge tone="amber">{t('noVisit')}</Badge>
        )
      ) : (
        <span className="text-muted">—</span>
      )}
      <Badge>{t('readOnlyBadge')}</Badge>
      <Link href="/clinical-config" className="btn btn-sm">
        {tcc('title')}
      </Link>
    </div>
  );
}
