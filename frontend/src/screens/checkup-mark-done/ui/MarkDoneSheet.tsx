'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { type ChangeEvent, useEffect, useMemo, useRef, useState } from 'react';

import {
  CHECKUP_ATTACHMENT_IMAGE_TYPES,
  CHECKUP_RESULTS,
  type CheckupRecord,
  type CheckupResult,
  checkupAttachments,
  checkupResultIcon,
  isCheckupAttachmentImage,
  useCheckup,
  useCheckupAttachment,
  useCheckupNextPreview,
  useCheckupRecords,
} from '@/entities/checkup';
import { useCreateCheckupRecord, useUpdateCheckupRecord } from '@/features/record-checkup';
import { getApiSaveErrorMessage } from '@/shared/api';
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
import { CalendarPicker, Icon, PrimaryButton, SecondaryButton, Skeleton, SkeletonGroup } from '@/shared/ui';

import { parseMarkDoneArg } from '../model/arg';
import { useMarkDoneToast } from '../model/toast';

const NOTE_MAX = 500;
const MAX_MB = Math.round(checkupAttachments.limits.maxFileBytes / (1024 * 1024));
/** The photo picker offers only the allow-listed photo types — never `image/*`, which admits SVG (audit M3-M7 #2). */
const PHOTO_ACCEPT = CHECKUP_ATTACHMENT_IMAGE_TYPES.join(',');

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
      <SkeletonGroup label={t('loading')} className="rmd-form-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
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
        <SkeletonGroup label={t('loading')} className="rmd-form-skel">
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
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
      existing.data && isCheckupAttachmentImage(existing.data.blob.type) ? URL.createObjectURL(existing.data.blob) : null,
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
    setPicked({ file, url: isCheckupAttachmentImage(file.type) ? URL.createObjectURL(file) : null });
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
      // The write limit (429) shows the server's localized «wait a moment».
      else setFormError(getApiSaveErrorMessage(error, t('saveError')));
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

      <div className="cfm-group">
        <span className="cfm-label" id="ckm-date">
          {tm('date')}
        </span>
        <button type="button" className="cfm-pick" aria-labelledby="ckm-date" onClick={() => openPicker('done')}>
          <span className="cfm-pick-v">{date(doneOn)}</span>
          <Icon name="calendar" size={18} />
        </button>
      </div>

      <div className="flex flex-col gap-2">
        <span className="cfm-label">{tm('result')}</span>
        <div role="radiogroup" aria-label={tm('result')} className="grid grid-cols-3 gap-2">
          {CHECKUP_RESULTS.map((r) => (
            <button
              key={r}
              type="button"
              role="radio"
              aria-checked={result === r}
              className={clsx(
                `ck-result-${r}`,
                'flex flex-col items-center gap-1.5 rounded-2xl border-[1.5px] px-2 py-3 text-[12.5px] font-extrabold',
                // Coloured by result (normal → success, follow-up → amber, pending → neutral).
                result === r
                  ? 'border-(--ck-ink) bg-(--ck-soft) text-(--ck-ink)'
                  : 'border-(--line) bg-(--surface) text-(--ink-2)',
              )}
              onClick={() => setResult(r)}
            >
              <Icon name={checkupResultIcon(r)} size={18} strokeWidth={2} />
              {t(`result.${r}`)}
            </button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <span className="cfm-label">
          {tm('attachment')} <span className="text-[12px] font-semibold text-(--ink-3)">{tm('optional')}</span>
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
            {(
              [
                ['camera', 'photo', photoRef],
                ['note', 'pdf', pdfRef],
              ] as const
            ).map(([icon, label, ref]) => (
              <button
                key={label}
                type="button"
                className="flex flex-col items-center gap-1 rounded-2xl border-[1.5px] border-dashed border-(--brand-line-soft) px-2 py-3.5 text-[13px] font-extrabold text-(--brand-strong)"
                onClick={() => ref.current?.click()}
              >
                <Icon name={icon} size={18} />
                {tm(label)}
              </button>
            ))}
          </div>
        )}
        <input ref={photoRef} type="file" accept={PHOTO_ACCEPT} capture="environment" hidden onChange={onFile} />
        <input ref={pdfRef} type="file" accept="application/pdf" hidden onChange={onFile} />
        {fileError && (
          <p role="alert" className="text-start text-[12px] font-bold text-(--danger-deep)">
            {tm(`attachmentError.${fileError}`, { max: MAX_MB })}
          </p>
        )}
        <p className="text-start text-[11.5px] text-(--ink-3)">{tm('privacy')}</p>
      </div>

      <label className="cfm-group">
        <span className="cfm-label">{tm('note')}</span>
        <span className="ck-note">
          <textarea
            rows={1}
            maxLength={NOTE_MAX}
            value={note}
            placeholder={tm('notePlaceholder')}
            onChange={(e) => setNote(e.target.value)}
          />
          <Icon name="note" size={18} className="mt-0.5 shrink-0" />
        </span>
      </label>

      {shownNext && (
        <div className="flex items-center gap-3 rounded-2xl bg-(--success-soft) p-3.5">
          <Icon name="bellPlain" size={18} className="shrink-0 text-(--success)" />
          <span className="flex min-w-0 flex-1 flex-col text-start">
            <span className="text-[13.5px] font-extrabold text-(--success)">{tm('nextDue', { date: shownNext })}</span>
            {nextMeta && <span className="text-[11.5px] font-semibold text-(--ink-2)">{nextMeta}</span>}
          </span>
          <button
            type="button"
            className="min-h-11 shrink-0 px-1 text-[12.5px] font-extrabold text-(--brand-strong)"
            onClick={() => openPicker('next')}
          >
            {tm('change')}
          </button>
        </div>
      )}

      {formError && (
        <p role="alert" className="text-center text-[12.5px] font-bold text-(--danger-deep)">
          {formError}
        </p>
      )}

      <PrimaryButton loading={saving} onClick={onSubmit}>
        {saving ? t('custom.saving') : tm('submit')}
      </PrimaryButton>

      <AppSheet
        open={picker !== null}
        onClose={() => setPicker(null)}
        size="half"
        title={picker === 'next' ? tm('nextDueTitle') : tm('date')}
        footer={
          <div className="flex gap-2.5">
            <SecondaryButton block={false} className="flex-1" onClick={() => setPicker(null)}>
              {t('custom.cancel')}
            </SecondaryButton>
            <PrimaryButton block={false} className="flex-1" disabled={draftInvalid} onClick={confirmPicker}>
              {t('custom.done')}
            </PrimaryButton>
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
