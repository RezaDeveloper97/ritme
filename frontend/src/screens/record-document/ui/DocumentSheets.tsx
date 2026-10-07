'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import {
  DOCUMENT_KINDS,
  type DocumentKind,
  MAX_DOCUMENT_NOTE,
  MAX_DOCUMENT_TEXT,
  type RecordDocument,
  RecordDateField,
  useDocAiConsent,
  useSetDocAiConsent,
  useUpdateRecordDocument,
} from '@/entities/health-record';
import { getApiErrorCode, getApiErrorMessage } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import {
  Checkbox,
  ChipGroup,
  IconCircle,
  PillChip,
  PrimaryButton,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
} from '@/shared/ui';

/* The document screen's sheets (CB-REC-04). They mount outside the screen's scroller, next to `.view`. */

/** A yes / no decision: delete, re-date the pregnancy. */
export function ConfirmSheet({
  open,
  title,
  body,
  yes,
  no,
  danger,
  busy,
  error,
  onYes,
  onClose,
}: {
  open: boolean;
  title: string;
  body: string;
  yes: string;
  no: string;
  danger?: boolean;
  busy: boolean;
  error: string | null;
  onYes: () => void;
  onClose: () => void;
}) {
  return (
    <AppSheet open={open} onClose={onClose} size="half" title={title}>
      <div className="hrec-sheet">
        <p className="rec-confirm-body">{body}</p>
        {error ? (
          <p className="hrec-error" role="alert">
            {error}
          </p>
        ) : null}
        <div className="hrec-sheet-actions">
          <PrimaryButton icon={danger ? 'trash' : 'check'} className={danger ? 'rec-danger-btn' : undefined} loading={busy} onClick={onYes}>
            {yes}
          </PrimaryButton>
          <SecondaryButton variant="text" onClick={onClose}>
            {no}
          </SecondaryButton>
        </div>
      </div>
    </AppSheet>
  );
}

interface EditDraft {
  kind: DocumentKind;
  title: string;
  date: string | null;
  endedOn: string | null;
  centre: string;
  doctor: string;
  note: string;
}

const draftOf = (d: RecordDocument): EditDraft => ({
  kind: d.kind,
  title: d.title ?? '',
  date: d.date,
  endedOn: d.endedOn,
  centre: d.centre ?? '',
  doctor: d.doctor ?? '',
  note: d.note ?? '',
});

/** «ویرایش سند» — the manual form (free users fill everything in here; PUT is partial). */
export function EditSheet({ open, doc, onClose }: { open: boolean; doc: RecordDocument; onClose: () => void }) {
  const t = useTranslations('record.doc');
  const tk = useTranslations('record.kinds');
  const tu = useTranslations('record.upload');
  const save = useUpdateRecordDocument(doc.id);
  const [d, setD] = useState<EditDraft>(() => draftOf(doc));
  useEffect(() => {
    if (open) {
      setD(draftOf(doc));
      save.reset();
    }
    // Re-seed only when the sheet opens.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);
  const pending = doc.reviewState === 'pending';
  const set = <K extends keyof EditDraft>(k: K, v: EditDraft[K]) => setD((cur) => ({ ...cur, [k]: v }));
  const submit = () => {
    const text = (v: string) => v.trim() || null;
    save.mutate(
      {
        ...(pending ? {} : { kind: d.kind }),
        title: text(d.title),
        date: d.date,
        endedOn: d.kind === 'hospital' ? d.endedOn : null,
        centre: text(d.centre),
        doctor: text(d.doctor),
        note: text(d.note),
      },
      { onSuccess: onClose },
    );
  };
  const error = save.isError
    ? getApiErrorCode(save.error) === 'extraction_running'
      ? t('running')
      : (getApiErrorMessage(save.error) ?? t('saveError'))
    : null;

  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="full"
      title={t('editTitle')}
      footer={
        <div className="rec-sheet-btns">
          <SecondaryButton block={false} onClick={onClose}>
            {t('cancel')}
          </SecondaryButton>
          <PrimaryButton block={false} loading={save.isPending} onClick={submit}>
            {t('save')}
          </PrimaryButton>
        </div>
      }
    >
      <div className="hrec-sheet">
        <div className="rec-up-block">
          <span className="rec-up-label">{t('fields.kind')}</span>
          <ChipGroup label={t('fields.kind')}>
            {DOCUMENT_KINDS.map((k) => (
              <PillChip key={k} pressed={d.kind === k} disabled={pending} onPressedChange={() => set('kind', k)}>
                {tk(k)}
              </PillChip>
            ))}
          </ChipGroup>
          {pending ? <p className="hrec-hint">{t('running')}</p> : null}
        </div>
        <TextField label={t('fields.title')} value={d.title} onChange={(v) => set('title', v)} />
        <RecordDateField
          label={t('fields.date')}
          value={d.date}
          unsetLabel={tu('dateUnset')}
          clearLabel={t('clearDate')}
          onChange={(v) => set('date', v)}
        />
        {d.kind === 'hospital' ? (
          <RecordDateField
            label={t('fields.ended_on')}
            value={d.endedOn}
            unsetLabel={tu('dateUnset')}
            clearLabel={t('clearDate')}
            onChange={(v) => set('endedOn', v)}
          />
        ) : null}
        <TextField label={t('fields.centre')} value={d.centre} onChange={(v) => set('centre', v)} />
        <TextField label={t('fields.doctor')} value={d.doctor} onChange={(v) => set('doctor', v)} />
        <TextField label={t('fields.note')} value={d.note} multiline max={MAX_DOCUMENT_NOTE} onChange={(v) => set('note', v)} />
        {error ? (
          <p className="hrec-error" role="alert">
            {error}
          </p>
        ) : null}
      </div>
    </AppSheet>
  );
}

export function TextField({
  label,
  value,
  onChange,
  multiline,
  max = MAX_DOCUMENT_TEXT,
  hint,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  multiline?: boolean;
  max?: number;
  hint?: string | null;
}) {
  return (
    <label className="rec-up-block">
      <span className="rec-up-label">{label}</span>
      {multiline ? (
        <textarea className="hrec-input rec-textarea" value={value} maxLength={max} rows={3} onChange={(e) => onChange(e.target.value)} />
      ) : (
        <input className="hrec-input" value={value} maxLength={max} onChange={(e) => onChange(e.target.value)} />
      )}
      {hint ? <span className="rec-low">{hint}</span> : null}
    </label>
  );
}

/**
 * The `ai_documents` consent (B-N6-05 platform; the same flow as bloom's lab consent sheet): the versioned text from
 * `GET /consents/ai_documents` (never hard-coded), an explicit checkbox, `PUT {granted, version}`, then `onGranted`.
 */
export function DocConsentSheet({ open, onClose, onGranted }: { open: boolean; onClose: () => void; onGranted: () => void }) {
  const t = useTranslations('record.doc.consent');
  return (
    <AppSheet open={open} onClose={onClose} size="full" title={t('title')}>
      {open ? <ConsentBody onClose={onClose} onGranted={onGranted} /> : null}
    </AppSheet>
  );
}

function ConsentBody({ onClose, onGranted }: { onClose: () => void; onGranted: () => void }) {
  const t = useTranslations('record.doc.consent');
  const locale = useLocale() as Locale;
  const consent = useDocAiConsent();
  const save = useSetDocAiConsent();
  const [agreed, setAgreed] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (consent.isPending) {
    return (
      <SkeletonGroup label={t('loading')} className="lab-consent">
        <Skeleton shape="line" width="medium" />
        <Skeleton shape="block" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  }
  if (consent.isError) {
    return (
      <div className="lab-consent">
        <p className="hrec-error" role="alert">
          {t('loadError')}
        </p>
        <SecondaryButton icon="refresh" onClick={() => void consent.refetch()}>
          {t('retry')}
        </SecondaryButton>
      </div>
    );
  }
  const c = consent.data;
  const accept = () => {
    setError(null);
    save.mutate(
      { granted: true, version: c.version },
      {
        onSuccess: onGranted,
        onError: (e) => setError(getApiErrorCode(e) === 'consent_version_stale' ? t('stale') : (getApiErrorMessage(e) ?? t('saveError'))),
      },
    );
  };
  return (
    <div className="lab-consent">
      <div className="lab-consent-head">
        <IconCircle icon="shield" tone="data" size="lg" outlined />
        <p className="lab-consent-lead">{t('lead')}</p>
      </div>
      {c.title ? <h3 className="lab-consent-title">{c.title}</h3> : null}
      {c.body ? <p className="lab-consent-body">{c.body}</p> : null}
      <ul className="lab-consent-points">
        {c.points.map((p, i) => (
          <li key={i} className="lab-consent-point">
            <IconCircle icon={i % 2 ? 'lock' : 'info'} tone={i % 2 ? 'data' : 'brand'} size="sm" />
            <span>{p}</span>
          </li>
        ))}
      </ul>
      <p className="lab-consent-version">{t('version', { v: formatNumber(c.version, locale) })}</p>
      <Checkbox className="lab-consent-check" checked={agreed} onCheckedChange={setAgreed} label={t('agree')} />
      {error ? (
        <p className="hrec-error" role="alert">
          {error}
        </p>
      ) : null}
      <div className="lab-consent-actions">
        <PrimaryButton disabled={!agreed} loading={save.isPending} onClick={accept}>
          {t('accept')}
        </PrimaryButton>
        <SecondaryButton variant="text" onClick={onClose}>
          {t('cancel')}
        </SecondaryButton>
      </div>
    </div>
  );
}
