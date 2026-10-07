'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useId, useState } from 'react';

import {
  EXTRACT_FIELDS,
  IMAGING_STUDIES,
  ITEM_FIELDS,
  LOW_CONFIDENCE,
  type RecordDocument,
  RecordDateField,
  type ReviewDraft,
  type ReviewItemDraft,
  isDateField,
  isNumberField,
  reviewDraftOf,
  reviewFieldsBody,
  reviewItemsBody,
  reviewItemsOf,
  useReviewRecordDocument,
} from '@/entities/health-record';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import { Card, ChipGroup, Icon, IconCircle, PillChip, PrimaryButton, SecondaryButton, StatusPill } from '@/shared/ui';

import type { ExtractErrorKind } from '../model/errors';
import { confirmedRows } from '../model/format';
import { TextField } from './DocumentSheets';

type T = ReturnType<typeof useTranslations<'record.doc'>>;

/** A read / confirmed value as the user reads it (dates in the locale's calendar, study kinds by name). */
export function fieldText(key: string, value: string | number, t: T, locale: Locale): string {
  if (isDateField(key) && typeof value === 'string') return formatLongDate(fromApiDate(value), locale);
  if (key === 'kind' && typeof value === 'string') {
    return (IMAGING_STUDIES as readonly string[]).includes(value) ? t(`studies.${value as (typeof IMAGING_STUDIES)[number]}`) : value;
  }
  return typeof value === 'number' ? formatNumber(value, locale) : value;
}

const fieldLabel = (key: string, t: T): string =>
  key === 'kind' ? t('fields.study') : key === 'title' ? t('fields.docTitle') : t(`fields.${key}` as 'fields.date');

/**
 * «اطلاعات خوانده‌شده از سند — اگر اشتباه است، ویرایش کن» (nbl_Rec_Doc, CB-REC-02 API): start the AI reading,
 * wait for it, review and confirm the read values, or show the confirmed ones. Free users (402) and failures fall
 * back to the manual form.
 */
export function ReadSection({
  doc,
  extracting,
  extractError,
  stalled,
  onExtract,
  onRecheck,
  onManual,
}: {
  doc: RecordDocument;
  extracting: boolean;
  extractError: ExtractErrorKind | null;
  /** Polling gave up (audit L3): show «دوباره بررسی کن» instead of the spinner. */
  stalled: boolean;
  onExtract: () => void;
  onRecheck: () => void;
  onManual: () => void;
}) {
  const t = useTranslations('record.doc');
  const router = useRouter();
  const titleId = useId();
  const state = doc.reviewState;

  let body;
  if (state === 'pending' && stalled) {
    body = (
      <div className="rec-read-start">
        <p className="rec-read-strong">{t('read.stalled')}</p>
        <p className="rec-read-sub">{t('read.stalledBody')}</p>
        <div className="rec-read-actions">
          <PrimaryButton icon="refresh" onClick={onRecheck}>
            {t('read.recheck')}
          </PrimaryButton>
        </div>
      </div>
    );
  } else if (state === 'pending') {
    body = (
      <div className="rec-read-state" role="status">
        <Icon name="loader" size={22} className="rec-spin" />
        <div>
          <p className="rec-read-strong">{t('read.pending')}</p>
          <p className="rec-read-sub">{t('read.pendingBody')}</p>
        </div>
      </div>
    );
  } else if (state === 'needs_review') {
    body = <ReviewForm doc={doc} />;
  } else if (state === 'confirmed') {
    body = <ConfirmedRows doc={doc} />;
  } else if (extractError === 'plus') {
    body = (
      <div className="rec-read-start">
        <p className="rec-read-strong">{t('read.plusTitle')}</p>
        <p className="rec-read-sub">{t('read.plusBody')}</p>
        <div className="rec-read-actions">
          <PrimaryButton icon="pencil" onClick={onManual}>
            {t('read.manual')}
          </PrimaryButton>
          <SecondaryButton variant="text" icon="crown" onClick={() => router.push('/plus')}>
            {t('read.plusCta')}
          </SecondaryButton>
        </div>
      </div>
    );
  } else {
    const failed = state === 'failed';
    const extra =
      doc.kind === 'imaging' ? t('read.extraImaging') : doc.kind === 'prescription' ? t('read.extraPrescription') : t('read.extraOther');
    const note =
      extractError === 'busy' ? t('read.busy') : extractError === 'noFiles' ? t('read.noFiles') : extractError === 'unknown' ? t('read.error') : null;
    body = (
      <div className="rec-read-start">
        <p className="rec-read-strong">{failed ? t('read.failed') : t('read.start')}</p>
        <p className="rec-read-sub">{failed ? t('read.failedBody') : t('read.startBody', { extra })}</p>
        {note ? (
          <p className="hrec-error" role="alert">
            {note}
          </p>
        ) : null}
        <div className="rec-read-actions">
          <PrimaryButton icon={failed ? 'refresh' : 'sparkle'} loading={extracting} disabled={doc.files.length === 0} onClick={onExtract}>
            {failed ? t('read.retry') : t('read.start')}
          </PrimaryButton>
          <SecondaryButton variant="text" icon="pencil" onClick={onManual}>
            {t('read.manual')}
          </SecondaryButton>
        </div>
      </div>
    );
  }

  return (
    <section className="rec-doc-section" aria-labelledby={titleId}>
      <div className="rec-doc-section-head">
        <div>
          <h2 id={titleId} className="rec-doc-h2">
            {t('read.title')}
          </h2>
          {state === 'needs_review' || state === 'confirmed' ? <p className="rec-doc-h2-sub">{t('read.subtitle')}</p> : null}
        </div>
        {state === 'confirmed' ? (
          <StatusPill tone="success" icon="check">
            {t('read.confirmed')}
          </StatusPill>
        ) : null}
      </div>
      <Card className="rec-doc-card">{body}</Card>
    </section>
  );
}

function ConfirmedRows({ doc }: { doc: RecordDocument }) {
  const t = useTranslations('record.doc');
  const locale = useLocale() as Locale;
  const rows = confirmedRows(doc.kind, doc.extracted);
  const items = doc.extracted?.reviewedItems ?? [];
  if (rows.length === 0 && items.length === 0) return <p className="hrec-empty">{t('notSet')}</p>;
  return (
    <div className="hrec-rows">
      {rows.map((r) =>
        'weeks' in r ? (
          <div key="ga" className="hrec-row">
            <span className="hrec-row-label">{t('fields.ga')}</span>
            <span className="rec-row-value">
              {r.days !== null
                ? t('gaValue', { weeks: formatNumber(r.weeks, locale), days: formatNumber(r.days, locale) })
                : t('gaWeeks', { weeks: formatNumber(r.weeks, locale) })}
            </span>
          </div>
        ) : (
          <div key={r.key} className="hrec-row">
            <span className="hrec-row-label">{fieldLabel(r.key, t)}</span>
            <span className="rec-row-value">{fieldText(r.key, r.value, t, locale)}</span>
          </div>
        ),
      )}
      {items.map((it, i) => (
        <div key={`item-${i}`} className="hrec-row">
          <span className="hrec-row-label">{it.medicine ?? t('read.item.medicine')}</span>
          <span className="rec-row-value">{[it.dose, it.frequency, it.duration].filter(Boolean).join(' · ')}</span>
        </div>
      ))}
    </div>
  );
}

const emptyItem = (): ReviewItemDraft => ({ medicine: '', dose: '', frequency: '', duration: '' });

function ReviewForm({ doc }: { doc: RecordDocument }) {
  const t = useTranslations('record.doc');
  const tu = useTranslations('record.upload');
  const locale = useLocale() as Locale;
  const review = useReviewRecordDocument(doc.id);
  const [draft, setDraft] = useState<ReviewDraft>(() => reviewDraftOf(doc.kind, doc.extracted));
  const [items, setItems] = useState<ReviewItemDraft[]>(() => reviewItemsOf(doc.extracted));
  useEffect(() => {
    setDraft(reviewDraftOf(doc.kind, doc.extracted));
    setItems(reviewItemsOf(doc.extracted));
  }, [doc.kind, doc.extracted]);

  const fields = doc.extracted?.fields ?? {};
  const low = (key: string) => {
    const c = fields[key]?.confidence;
    return c !== undefined && c < LOW_CONFIDENCE ? t('read.lowConfidence') : null;
  };
  const set = (key: string, v: string) => setDraft((d) => ({ ...d, [key]: v }));
  const submit = () =>
    review.mutate({
      fields: reviewFieldsBody(doc.kind, draft),
      ...(doc.kind === 'prescription' ? { items: reviewItemsBody(items) } : {}),
    });
  const error = review.isError ? (getApiErrorStatus(review.error) === 422 ? t('read.invalid') : t('read.error')) : null;

  return (
    <div className="rec-review-form">
      {EXTRACT_FIELDS[doc.kind].map((key) => {
        if (key === 'kind') {
          return (
            <div key={key} className="rec-up-block">
              <span className="rec-up-label">{t('fields.study')}</span>
              <ChipGroup label={t('fields.study')}>
                {IMAGING_STUDIES.map((s) => (
                  <PillChip key={s} pressed={draft.kind === s} onPressedChange={(on) => set('kind', on ? s : '')}>
                    {t(`studies.${s}`)}
                  </PillChip>
                ))}
              </ChipGroup>
              {low(key) ? <span className="rec-low">{low(key)}</span> : null}
            </div>
          );
        }
        if (isDateField(key)) {
          return (
            <div key={key} className="rec-up-block">
              <RecordDateField
                label={fieldLabel(key, t)}
                value={draft[key] || null}
                unsetLabel={tu('dateUnset')}
                clearLabel={t('clearDate')}
                onChange={(v) => set(key, v ?? '')}
              />
              {low(key) ? <span className="rec-low">{low(key)}</span> : null}
            </div>
          );
        }
        if (isNumberField(key)) {
          return (
            <label key={key} className="rec-up-block">
              <span className="rec-up-label">{fieldLabel(key, t)}</span>
              <input
                className="hrec-input rec-num"
                inputMode="numeric"
                dir="ltr"
                value={draft[key] ? formatNumber(draft[key]!, locale) : ''}
                maxLength={2}
                onChange={(e) => set(key, e.target.value.replace(/[^\d۰-۹٠-٩]/g, ''))}
              />
              {low(key) ? <span className="rec-low">{low(key)}</span> : null}
            </label>
          );
        }
        return (
          <TextField
            key={key}
            label={fieldLabel(key, t)}
            value={draft[key] ?? ''}
            multiline={key === 'findings'}
            max={500}
            hint={low(key)}
            onChange={(v) => set(key, v)}
          />
        );
      })}

      {doc.kind === 'prescription' ? (
        <div className="rec-items">
          <span className="rec-up-label">{t('read.items')}</span>
          {items.map((it, i) => (
            <div key={i} className="rec-item">
              <div className="rec-item-head">
                <IconCircle icon="pill" tone="data" size="sm" />
                <span className="rec-item-n">{formatNumber(i + 1, locale)}</span>
                <button
                  type="button"
                  className="hrec-entry-del"
                  aria-label={t('read.removeItem', { n: formatNumber(i + 1, locale) })}
                  onClick={() => setItems((cur) => cur.filter((_, j) => j !== i))}
                >
                  <Icon name="trash" size={18} />
                </button>
              </div>
              {ITEM_FIELDS.map((f) => (
                <TextField
                  key={f}
                  label={t(`read.item.${f}`)}
                  value={it[f]}
                  max={500}
                  onChange={(v) => setItems((cur) => cur.map((row, j) => (j === i ? { ...row, [f]: v } : row)))}
                />
              ))}
            </div>
          ))}
          {items.length < 30 ? (
            <SecondaryButton icon="plus" onClick={() => setItems((cur) => [...cur, emptyItem()])}>
              {t('read.addItem')}
            </SecondaryButton>
          ) : null}
        </div>
      ) : null}

      {error ? (
        <p className="hrec-error" role="alert">
          {error}
        </p>
      ) : null}
      <PrimaryButton icon="check" loading={review.isPending} onClick={submit}>
        {t('read.confirm')}
      </PrimaryButton>
    </div>
  );
}
