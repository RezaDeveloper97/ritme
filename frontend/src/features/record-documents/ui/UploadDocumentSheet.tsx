'use client';

import { useLocale, useTranslations } from 'next-intl';
import { type ChangeEvent, useEffect, useId, useRef, useState } from 'react';

import {
  DOCUMENT_KINDS,
  type DocumentKind,
  MAX_DOCUMENT_FILES,
  MAX_DOCUMENT_TEXT,
  type RecordDocument,
  RecordDateField,
  deleteRecordFile,
  useCreateRecordDocument,
  useUploadRecordFile,
} from '@/entities/health-record';
import { ApiError, getApiErrorMessage } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { ChipGroup, Icon, PillChip, PrimaryButton, ProgressBar, TileButton } from '@/shared/ui';

import { addFiles, type DraftError, type DraftFile, overallProgress, type PickError, validateDraft } from '../model/draft';

let seq = 0;
const nextKey = () => `rec-file-${Date.now().toString(36)}-${++seq}`;

/** Object URLs of the photo files, revoked when a file goes or the sheet closes. */
function usePreviews(files: readonly DraftFile[]): Record<string, string> {
  const [urls, setUrls] = useState<Record<string, string>>({});
  useEffect(() => {
    const made: Record<string, string> = {};
    for (const f of files) if (f.kind === 'image') made[f.key] = URL.createObjectURL(f.file);
    setUrls(made);
    return () => Object.values(made).forEach((u) => URL.revokeObjectURL(u));
  }, [files]);
  return urls;
}

interface UploadDocumentSheetProps {
  open: boolean;
  onClose: () => void;
  /** The 201 answer — e.g. route to the document. */
  onCreated: (doc: RecordDocument) => void;
}

/**
 * «افزودن سند» (CB-REC-04, nbl_Rec_Home / _Timeline): camera / gallery / PDF pickers (up to 10 files), the document
 * kind, an optional title and date, then each file goes to `POST /files` (purpose record_document) and the document
 * to `POST /health-record/documents`. Files left behind by a failed create are deleted again. Nothing is logged.
 */
export function UploadDocumentSheet({ open, onClose, onCreated }: UploadDocumentSheetProps) {
  const t = useTranslations('record.upload');
  return (
    <AppSheet open={open} onClose={onClose} size="full" title={t('title')}>
      {open ? <UploadForm onCreated={onCreated} /> : null}
    </AppSheet>
  );
}

function UploadForm({ onCreated }: { onCreated: (doc: RecordDocument) => void }) {
  const t = useTranslations('record.upload');
  const tk = useTranslations('record.kinds');
  const td = useTranslations('record.doc');
  const locale = useLocale() as Locale;
  const titleId = useId();
  const [files, setFiles] = useState<DraftFile[]>([]);
  const [kind, setKind] = useState<DocumentKind | null>(null);
  const [title, setTitle] = useState('');
  const [date, setDate] = useState<string | null>(null);
  const [pickError, setPickError] = useState<PickError | null>(null);
  const [draftError, setDraftError] = useState<DraftError | null>(null);
  const [serverError, setServerError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [progress, setProgress] = useState(0);
  const camera = useRef<HTMLInputElement>(null);
  const gallery = useRef<HTMLInputElement>(null);
  const pdf = useRef<HTMLInputElement>(null);
  const previews = usePreviews(files);
  const upload = useUploadRecordFile();
  const create = useCreateRecordDocument();
  const full = files.length >= MAX_DOCUMENT_FILES;
  const num = (v: number) => formatNumber(v, locale);

  const onPick = (event: ChangeEvent<HTMLInputElement>) => {
    const input = event.target;
    const picked = Array.from(input.files ?? []);
    input.value = '';
    if (!picked.length) return;
    const r = addFiles(files, picked, nextKey);
    setFiles(r.files);
    setPickError(r.error);
    setDraftError(null);
    setServerError(null);
  };

  const submit = async () => {
    const found = validateDraft({ kind, files });
    setDraftError(found);
    setServerError(null);
    if (found || !kind) return;
    setBusy(true);
    setProgress(0);
    const uploaded: number[] = [];
    try {
      for (let i = 0; i < files.length; i++) {
        const stored = await upload.mutateAsync({
          file: files[i]!.file,
          onProgress: (f) => setProgress(overallProgress(i, f, files.length)),
        });
        uploaded.push(stored.id);
        setProgress(overallProgress(i + 1, 0, files.length));
      }
      const doc = await create.mutateAsync({
        kind,
        title: title.trim() || null,
        date,
        fileIds: uploaded,
      });
      onCreated(doc);
    } catch (error) {
      // best effort: an upload that never reached a document would otherwise wait for the 6-hourly sweep
      await Promise.allSettled(uploaded.map((id) => deleteRecordFile(id)));
      setServerError(
        error instanceof ApiError && !error.response ? t('errors.network') : (getApiErrorMessage(error) ?? t('errors.unknown')),
      );
    } finally {
      setBusy(false);
    }
  };

  const fileError = draftError === 'files' ? t('errors.files') : pickError ? t(`errors.${pickError}`, { max: num(MAX_DOCUMENT_FILES) }) : null;

  return (
    <div className="rec-up">
      <div className="lab-up-sources">
        <TileButton layout="card" icon="camera" tone="brand" label={t('camera')} disabled={full || busy} onClick={() => camera.current?.click()} />
        <TileButton layout="card" icon="image" tone="data" label={t('gallery')} disabled={full || busy} onClick={() => gallery.current?.click()} />
        <TileButton layout="card" icon="fileDoc" tone="warm" label={t('pdf')} disabled={full || busy} onClick={() => pdf.current?.click()} />
      </div>
      <input ref={camera} className="sr-only" type="file" accept="image/*" capture="environment" tabIndex={-1} aria-hidden onChange={onPick} />
      <input ref={gallery} className="sr-only" type="file" accept="image/jpeg,image/png,image/webp,image/heic,image/heif" multiple tabIndex={-1} aria-hidden onChange={onPick} />
      <input ref={pdf} className="sr-only" type="file" accept="application/pdf,.pdf" multiple tabIndex={-1} aria-hidden onChange={onPick} />

      <section className="rec-up-block" aria-labelledby={`${titleId}-files`}>
        <div className="rec-up-head">
          <h3 id={`${titleId}-files`} className="rec-up-label">
            {t('files')}
          </h3>
          <span className="rec-up-count">{t('fileCount', { n: num(files.length), max: num(MAX_DOCUMENT_FILES) })}</span>
        </div>
        {files.length === 0 ? (
          <p className="rec-up-hint">{t('noFiles')}</p>
        ) : (
          <ol className="lab-up-pages">
            {files.map((f, i) => (
              <li key={f.key} className="lab-up-page">
                <div className={f.kind === 'pdf' ? 'lab-up-thumb is-pdf' : 'lab-up-thumb'}>
                  {f.kind === 'image' && previews[f.key] ? (
                    // eslint-disable-next-line @next/next/no-img-element -- local object URL, never optimized
                    <img src={previews[f.key]} alt="" className="lab-up-img" />
                  ) : (
                    <Icon name={f.kind === 'pdf' ? 'fileDoc' : 'image'} size={28} strokeWidth={1.6} />
                  )}
                  <span className="lab-up-num">{num(i + 1)}</span>
                </div>
                <button
                  type="button"
                  className="lab-up-remove"
                  aria-label={t('remove', { n: num(i + 1) })}
                  disabled={busy}
                  onClick={() => setFiles((cur) => cur.filter((x) => x.key !== f.key))}
                >
                  <Icon name="x" size={12} strokeWidth={3} />
                </button>
              </li>
            ))}
          </ol>
        )}
        {fileError ? (
          <p className="hrec-error" role="alert">
            {fileError}
          </p>
        ) : null}
      </section>

      <section className="rec-up-block" aria-labelledby={`${titleId}-kind`}>
        <h3 id={`${titleId}-kind`} className="rec-up-label">
          {t('kind')}
        </h3>
        <ChipGroup label={t('kind')}>
          {DOCUMENT_KINDS.map((k) => (
            <PillChip key={k} pressed={kind === k} onPressedChange={() => setKind(k)} disabled={busy}>
              {tk(k)}
            </PillChip>
          ))}
        </ChipGroup>
        {draftError === 'kind' ? (
          <p className="hrec-error" role="alert">
            {t('errors.kind')}
          </p>
        ) : null}
      </section>

      <label className="rec-up-block">
        <span className="rec-up-label">{t('docTitle')}</span>
        <input
          className="hrec-input"
          value={title}
          maxLength={MAX_DOCUMENT_TEXT}
          placeholder={t('titlePlaceholder')}
          disabled={busy}
          onChange={(e) => setTitle(e.target.value)}
        />
      </label>

      <RecordDateField
        label={t('date')}
        value={date}
        unsetLabel={t('dateUnset')}
        clearLabel={td('clearDate')}
        disabled={busy}
        onChange={setDate}
      />

      <p className="rec-up-hint">{t('hint')}</p>

      {busy ? (
        <ProgressBar
          value={Math.round(progress * 100)}
          max={100}
          label={t('uploading', { percent: num(Math.round(progress * 100)) })}
        />
      ) : null}
      {serverError ? (
        <p className="hrec-error" role="alert">
          {serverError}
        </p>
      ) : null}
      <PrimaryButton icon="check" loading={busy} onClick={() => void submit()}>
        {t('submit')}
      </PrimaryButton>

    </div>
  );
}
