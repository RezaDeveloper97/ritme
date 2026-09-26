'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { type ChangeEvent, useEffect, useMemo, useRef, useState } from 'react';

import {
  CHECKUP_RESULTS,
  type CheckupRecord,
  type CheckupResult,
  checkupAttachments,
  useCheckup,
  useCheckupAttachment,
  useCheckupNextPreview,
  useCheckupRecords,
} from '@/entities/checkup';
import { useCreateCheckupRecord, useUpdateCheckupRecord } from '@/features/record-checkup';
import type { Locale } from '@/shared/i18n';
import {
  type DateParts,
  diffInDays,
  formatLongDate,
  fromApiDate,
  partsToDate,
  toApiDate,
  toParts,
  today,
} from '@/shared/lib/date';
import { LocalFilesError, type LocalFilesErrorCode } from '@/shared/lib/local-files';
import { AppSheet, type SheetContentProps, closeSheet } from '@/shared/sheet';
import { CalendarPicker, Icon } from '@/shared/ui';

import { parseMarkDoneArg } from '../model/arg';
import { useMarkDoneToast } from '../model/toast';

const NOTE_MAX = 500;
const MAX_MB = Math.round(checkupAttachments.limits.maxFileBytes / (1024 * 1024));

interface PickedFile {
  file: File;
  url: string | null;
}

/** Sheet heading: «X را انجام دادم» / «ویرایش X». */
export function MarkDoneTitle({ arg }: SheetContentProps) {
  const t = useTranslations('checkups.markDone');
  const target = parseMarkDoneArg(arg);
  const detail = useCheckup(target?.typeId ?? null);
  if (!target || !detail.data) return null;
  return <>{target.recordId ? t('editTitle', { title: detail.data.title }) : t('title', { title: detail.data.title })}</>;
}

/**
 * MarkDone (`v14_MarkDone`) — record a visit or edit an existing record.
 * The report file stays on this device (`checkupAttachments`); the API only
 * gets `has_attachment`. Health data (§11): never logged.
 */
export function MarkDoneSheet({ arg }: SheetContentProps) {
  const t = useTranslations('checkups');
  const target = parseMarkDoneArg(arg);
  const detail = useCheckup(target?.typeId ?? null);
  const records = useCheckupRecords({ type: target?.recordId ? target.typeId : null });

  if (!target) return <p className="rmd-state">{t('loadError')}</p>;
  if (detail.isPending) {
    return (
      <p className="rmd-state" role="status">
        {t('loading')}
      </p>
    );
  }
  if (!detail.data) {
    return (
      <div className="rmd-state" role="alert">
        <p>{t('loadError')}</p>
        <button type="button" className="rmd-retry" onClick={() => void detail.refetch()}>
          {t('retry')}
        </button>
      </div>
    );
  }

  let record: CheckupRecord | null = null;
  if (target.recordId) {
    record =
      detail.data.records.find((r) => r.id === target.recordId) ??
      records.data?.pages.flatMap((p) => p.records).find((r) => r.id === target.recordId) ??
      null;
    if (!record) {
      return records.isPending ? (
        <p className="rmd-state" role="status">
          {t('loading')}
        </p>
      ) : (
        <p className="rmd-state" role="alert">
          {t('loadError')}
        </p>
      );
    }
  }

  return <MarkDoneForm key={record?.id ?? 'new'} typeId={target.typeId} record={record} />;
}

function MarkDoneForm({ typeId, record }: { typeId: number; record: CheckupRecord | null }) {
  const t = useTranslations('checkups');
  const tm = useTranslations('checkups.markDone');
  const locale = useLocale() as Locale;
  const showToast = useMarkDoneToast((s) => s.show);

  const [doneOn, setDoneOn] = useState(record?.doneOn ?? toApiDate(today()));
  const [result, setResult] = useState<CheckupResult>(record?.result ?? 'normal');
  const [note, setNote] = useState(record?.note ?? '');
  const [nextOverride, setNextOverride] = useState<string | null>(record?.nextDueOn ?? null);
  const [picked, setPicked] = useState<PickedFile | null>(null);
  const [removedExisting, setRemovedExisting] = useState(false);
  const [fileError, setFileError] = useState<LocalFilesErrorCode | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [picker, setPicker] = useState<'done' | 'next' | null>(null);
  const [draft, setDraft] = useState<DateParts | null>(null);

  const photoRef = useRef<HTMLInputElement>(null);
  const pdfRef = useRef<HTMLInputElement>(null);

  const existing = useCheckupAttachment(record?.hasAttachment ? record.id : null);
  const preview = useCheckupNextPreview(typeId, doneOn);
  const create = useCreateCheckupRecord();
  const update = useUpdateCheckupRecord();
  const saving = create.isPending || update.isPending;

  const existingUrl = useMemo(
    () =>
      existing.data && existing.data.blob.type.startsWith('image/') ? URL.createObjectURL(existing.data.blob) : null,
    [existing.data],
  );
  useEffect(() => () => void (existingUrl && URL.revokeObjectURL(existingUrl)), [existingUrl]);
  useEffect(() => () => void (picked?.url && URL.revokeObjectURL(picked.url)), [picked]);

  const date = (d: string) => formatLongDate(fromApiDate(d), locale);

  const onFile = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    const invalid = checkupAttachments.check(file);
    if (invalid) {
      setFileError(invalid);
      return;
    }
    setFileError(null);
    setPicked({ file, url: file.type.startsWith('image/') ? URL.createObjectURL(file) : null });
  };

  const removeAttachment = () => {
    setPicked(null);
    if (record?.hasAttachment) setRemovedExisting(true);
  };

  const openPicker = (which: 'done' | 'next') => {
    const current = which === 'done' ? doneOn : (nextOverride ?? preview.data?.nextDueOn ?? doneOn);
    setDraft(toParts(fromApiDate(current), locale));
    setPicker(which);
  };
  const draftDate = draft ? partsToDate(draft, locale) : null;
  const draftInvalid =
    !draftDate ||
    (picker === 'done' && diffInDays(draftDate, today()) > 0) ||
    (picker === 'next' && diffInDays(draftDate, fromApiDate(doneOn)) <= 0);

  const confirmPicker = () => {
    if (!draftDate || draftInvalid) return;
    const value = toApiDate(draftDate);
    if (picker === 'done') {
      setDoneOn(value);
      if (nextOverride && diffInDays(fromApiDate(nextOverride), draftDate) <= 0) setNextOverride(null);
    } else {
      setNextOverride(value);
    }
    setPicker(null);
  };

  const onSubmit = () => {
    setFormError(null);
    const input = { doneOn, result, note, nextDueOn: nextOverride };
    const attachment = picked ? { file: picked.file, name: picked.file.name } : undefined;
    const onSuccess = ({ attachmentError }: { attachmentError: LocalFilesErrorCode | null }) => {
      showToast(attachmentError ? tm('attachmentNotSaved') : tm('saved'), attachmentError ? 'warn' : 'ok');
      closeSheet();
    };
    const onError = (error: unknown) => {
      if (error instanceof LocalFilesError) setFileError(error.code);
      else setFormError(t('saveError'));
    };
    if (record) {
      update.mutate(
        {
          recordId: record.id,
          typeId,
          patch: input,
          attachment: attachment ?? (removedExisting ? null : undefined),
        },
        { onSuccess, onError },
      );
    } else {
      create.mutate({ typeId, input, attachment: attachment ?? null }, { onSuccess, onError });
    }
  };

  const showExisting = !picked && !removedExisting && record?.hasAttachment;
  const shownNext = nextOverride ? date(nextOverride) : (preview.data?.nextDueLabel ?? null);
  const nextMeta = [preview.data?.intervalLabel, preview.data?.reminderLabel].filter(Boolean).join(t('separator'));

  return (
    <div className="flex flex-col gap-3.5 pb-2">
      <p className="text-start text-[12.5px] text-(--ink-3)">{tm('subtitle')}</p>

      <div className="fld-row">
        <span className="fld-row-label text-(--ink)">{tm('date')}</span>
        <button type="button" className="chip on" onClick={() => openPicker('done')}>
          <Icon name="calendar" size={14} />
          {date(doneOn)}
        </button>
      </div>

      <div className="flex flex-col gap-2">
        <span className="fld-label-t text-start">{tm('result')}</span>
        <div role="radiogroup" aria-label={tm('result')} className="grid grid-cols-3 gap-2">
          {CHECKUP_RESULTS.map((r) => (
            <button
              key={r}
              type="button"
              role="radio"
              aria-checked={result === r}
              className={clsx(
                'rounded-2xl border-[1.5px] px-2 py-3 text-[12.5px] font-extrabold',
                result === r
                  ? 'border-(--brand) bg-(--pink-bg) text-(--brand)'
                  : 'border-(--line) bg-(--surface) text-(--ink-2)',
              )}
              onClick={() => setResult(r)}
            >
              {t(`result.${r}`)}
            </button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <span className="fld-label-t text-start">
          {tm('attachment')} <span className="text-(--ink-3)">{tm('optional')}</span>
        </span>
        {picked || showExisting ? (
          <div className="flex items-center gap-3 rounded-2xl bg-(--surface-2) p-2.5">
            {(picked?.url ?? (showExisting ? existingUrl : null)) ? (
              // eslint-disable-next-line @next/next/no-img-element -- local object URL, never optimized
              <img
                src={(picked?.url ?? existingUrl) as string}
                alt=""
                className="size-12 shrink-0 rounded-xl object-cover"
              />
            ) : (
              <span className="grid size-12 shrink-0 place-items-center rounded-xl bg-(--surface) text-(--brand)">
                <Icon name="note" size={20} />
              </span>
            )}
            <span className="min-w-0 flex-1 truncate text-start text-[12.5px] font-bold text-(--ink)">
              {picked?.file.name ?? existing.data?.name ?? tm('attachment')}
            </span>
            <button
              type="button"
              className="grid size-8 place-items-center rounded-full bg-(--surface) text-(--ink-3)"
              aria-label={tm('removeAttachment')}
              onClick={removeAttachment}
            >
              <Icon name="x" size={14} />
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-2">
            <button type="button" className="btn btn-ghost" onClick={() => photoRef.current?.click()}>
              <Icon name="camera" size={16} />
              {tm('photo')}
            </button>
            <button type="button" className="btn btn-ghost" onClick={() => pdfRef.current?.click()}>
              <Icon name="note" size={16} />
              {tm('pdf')}
            </button>
          </div>
        )}
        <input ref={photoRef} type="file" accept="image/*" capture="environment" hidden onChange={onFile} />
        <input ref={pdfRef} type="file" accept="application/pdf" hidden onChange={onFile} />
        {fileError && (
          <p role="alert" className="text-start text-[12px] font-bold text-(--danger-deep)">
            {tm(`attachmentError.${fileError}`, { max: MAX_MB })}
          </p>
        )}
        <p className="flex items-start gap-1.5 text-start text-[11.5px] text-(--ink-3)">
          <Icon name="shield" size={14} className="mt-0.5 shrink-0" />
          {tm('privacy')}
        </p>
      </div>

      <label className="fld-label">
        <span className="fld-label-t">
          {tm('note')} <span className="text-(--ink-3)">{tm('optional')}</span>
        </span>
        <textarea
          className="field fld-textarea h-auto min-h-20 py-3"
          rows={2}
          maxLength={NOTE_MAX}
          value={note}
          placeholder={tm('notePlaceholder')}
          onChange={(e) => setNote(e.target.value)}
        />
      </label>

      {shownNext && (
        <div className="flex items-center gap-3 rounded-2xl bg-(--pink-bg) p-3">
          <Icon name="calendar" size={18} className="shrink-0 text-(--brand)" />
          <span className="flex min-w-0 flex-1 flex-col text-start">
            <span className="text-[13px] font-extrabold text-(--ink)">{tm('nextDue', { date: shownNext })}</span>
            {nextMeta && <span className="text-[11.5px] text-(--ink-3)">{nextMeta}</span>}
          </span>
          <button type="button" className="chip" onClick={() => openPicker('next')}>
            {tm('change')}
          </button>
        </div>
      )}

      {formError && (
        <p role="alert" className="text-center text-[12.5px] font-bold text-(--danger-deep)">
          {formError}
        </p>
      )}

      <button type="button" className="btn btn-primary w-full" disabled={saving} onClick={onSubmit}>
        {saving ? t('custom.saving') : tm('submit')}
      </button>

      <AppSheet
        open={picker !== null}
        onClose={() => setPicker(null)}
        size="half"
        title={picker === 'next' ? tm('nextDueTitle') : tm('date')}
        footer={
          <div className="flex gap-2.5">
            <button type="button" className="btn btn-ghost flex-1" onClick={() => setPicker(null)}>
              {t('custom.cancel')}
            </button>
            <button type="button" className="btn btn-primary flex-1" disabled={draftInvalid} onClick={confirmPicker}>
              {t('custom.done')}
            </button>
          </div>
        }
      >
        <CalendarPicker value={draft} onSelect={setDraft} />
        {draft && draftInvalid && (
          <p role="alert" className="mt-2 text-start text-[12px] font-bold text-(--danger-deep)">
            {picker === 'done' ? tm('futureDate') : tm('nextBeforeDone')}
          </p>
        )}
      </AppSheet>
    </div>
  );
}
