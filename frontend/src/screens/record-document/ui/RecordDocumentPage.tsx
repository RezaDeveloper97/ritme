'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useId, useState } from 'react';

import {
  type DatingOffer,
  type DocumentFile,
  type RecordDocument,
  downloadSignedFile,
  isImageMime,
  useAnswerRecordDating,
  useDeleteRecordDocument,
  useExtractRecordDocument,
  useRecordDating,
  useRecordDocument,
} from '@/entities/health-record';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import {
  Card,
  EmptyState,
  Icon,
  IconCircle,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';

import { type ExtractErrorKind, extractErrorOf } from '../model/errors';
import { useSignedBlob } from '../model/use-signed-blob';
import { DETAIL_KEYS, detailValue, sizeOf } from '../model/format';
import { ConfirmSheet, DocConsentSheet, EditSheet } from './DocumentSheets';
import { ReadSection } from './ReadSection';

type SheetId = 'edit' | 'consent' | 'delete' | 'dating' | 'preview';

/** Polling a pending extraction stops after this long; the screen then offers «دوباره بررسی کن» (audit L3). */
const POLL_LIMIT_MS = 3 * 60_000;

const fileName = (docId: number, f: DocumentFile, n: number) =>
  `ritme-document-${docId}-${n}.${f.mime === 'application/pdf' ? 'pdf' : f.mime === 'image/webp' ? 'webp' : 'img'}`;

/**
 * `/record/documents/[id]` (CB-REC-04, nbl_Rec_Doc): preview through the 5-minute signed links, the AI reading
 * (Plus + consent) with its review form or the manual form, the pregnancy dating offer (explicit confirm only),
 * «where used» and delete. Back header, no bottom nav. Health data (§11): only the id is in the URL.
 */
export function RecordDocumentPage({ id }: { id: number }) {
  const t = useTranslations('record.doc');
  const tk = useTranslations('record.kinds');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const [pollStalled, setPollStalled] = useState(false);
  const query = useRecordDocument(id, !pollStalled);
  const [sheet, setSheet] = useState<SheetId | null>(null);
  const [previewFile, setPreviewFile] = useState<DocumentFile | null>(null);
  const pending = query.data?.reviewState === 'pending';
  const [pollRound, setPollRound] = useState(0);
  useEffect(() => {
    if (!pending) {
      setPollStalled(false);
      return;
    }
    const timer = window.setTimeout(() => setPollStalled(true), POLL_LIMIT_MS);
    return () => window.clearTimeout(timer);
  }, [pending, pollRound]);
  const recheck = () => {
    setPollStalled(false);
    setPollRound((r) => r + 1);
    void query.refetch();
  };
  const toTimeline = () => router.push('/record/timeline');
  const extract = useExtractRecordDocument(id);
  const [extractError, setExtractError] = useState<ExtractErrorKind | null>(null);
  const read = () => {
    setExtractError(null);
    extract.mutate(undefined, {
      onError: (e) => {
        const kind = extractErrorOf(e);
        if (kind === 'consent') setSheet('consent');
        else if (kind === 'conflict') void query.refetch();
        else setExtractError(kind);
      },
    });
  };

  const doc = query.data;
  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="rec-doc-skel">
        <Skeleton shape="card" className="rec-doc-skel-preview" />
        <Skeleton shape="card" className="rec-doc-skel-card" />
        <Skeleton shape="card" className="rec-doc-skel-card" />
      </SkeletonGroup>
    );
  } else if (!doc) {
    const missing = id <= 0 || getApiErrorStatus(query.error) === 404;
    body = (
      <EmptyState
        icon={missing ? 'fileDoc' : 'warning'}
        title={missing ? t('notFound') : t('loadError')}
        body={missing ? t('notFoundBody') : undefined}
        action={
          missing ? (
            <PrimaryButton icon="arrowR" block={false} onClick={toTimeline}>
              {t('toTimeline')}
            </PrimaryButton>
          ) : (
            <PrimaryButton icon="refresh" block={false} onClick={() => void query.refetch()}>
              {t('retry')}
            </PrimaryButton>
          )
        }
      />
    );
  } else {
    body = (
      <DocumentBody
        doc={doc}
        extracting={extract.isPending}
        extractError={extractError}
        stalled={pollStalled}
        onExtract={read}
        onRecheck={recheck}
        onSheet={setSheet}
        onPreview={(f) => {
          setPreviewFile(f);
          setSheet('preview');
        }}
      />
    );
  }

  return (
    <div className="view rec-doc-page">
      <SkyLayer />
      <div className="scroll rec-doc-scroll">
        <ScreenHeader
          title={doc ? doc.title || tk(doc.kind) : t('loading')}
          subtitle={doc?.date ? formatLongDate(fromApiDate(doc.date), locale) : doc ? tk(doc.kind) : undefined}
          onBack={toTimeline}
          backLabel={t('back')}
        />
        {body}
      </div>
      {doc ? <DocumentSheets doc={doc} sheet={sheet} previewFile={previewFile} onSheet={setSheet} onConsented={read} /> : null}
    </div>
  );
}

function DocumentBody({
  doc,
  extracting,
  extractError,
  stalled,
  onExtract,
  onRecheck,
  onSheet,
  onPreview,
}: {
  doc: RecordDocument;
  extracting: boolean;
  extractError: ExtractErrorKind | null;
  stalled: boolean;
  onExtract: () => void;
  onRecheck: () => void;
  onSheet: (s: SheetId) => void;
  onPreview: (f: DocumentFile) => void;
}) {
  const t = useTranslations('record.doc');
  const [downloading, setDownloading] = useState(false);
  const [downloadFailed, setDownloadFailed] = useState(false);
  const firstIndex = doc.files.findIndex((f) => f.url);
  const save = async (f: DocumentFile, n: number) => {
    if (!f.url) return;
    setDownloading(true);
    setDownloadFailed(false);
    try {
      await downloadSignedFile(f.url, fileName(doc.id, f, n));
    } catch {
      setDownloadFailed(true);
    } finally {
      setDownloading(false);
    }
  };
  const open = (f: DocumentFile, n: number) => (isImageMime(f.mime) ? onPreview(f) : void save(f, n));

  return (
    <>
      <Preview files={doc.files} onOpen={open} />
      <ReadSection
        doc={doc}
        extracting={extracting}
        extractError={extractError}
        stalled={stalled}
        onExtract={onExtract}
        onRecheck={onRecheck}
        onManual={() => onSheet('edit')}
      />
      <DetailsCard doc={doc} onEdit={() => onSheet('edit')} />
      {doc.kind === 'imaging' && doc.reviewState === 'confirmed' ? <DatingCard id={doc.id} onApply={() => onSheet('dating')} /> : null}
      <WhereUsed doc={doc} />
      <div className="rec-doc-actions">
        {firstIndex >= 0 ? (
          <SecondaryButton icon="download" loading={downloading} onClick={() => void save(doc.files[firstIndex]!, firstIndex + 1)}>
            {t('download')}
          </SecondaryButton>
        ) : null}
        <SecondaryButton icon="trash" className="rec-delete-btn" onClick={() => onSheet('delete')}>
          {t('delete')}
        </SecondaryButton>
      </div>
      {downloadFailed ? (
        <p className="hrec-error" role="alert">
          {t('downloadError')}
        </p>
      ) : null}
    </>
  );
}

function Preview({ files, onOpen }: { files: DocumentFile[]; onOpen: (f: DocumentFile, n: number) => void }) {
  const t = useTranslations('record.doc');
  const locale = useLocale() as Locale;
  const titleId = useId();
  if (files.length === 0) return null;
  const size = (b: number) => {
    const s = sizeOf(b);
    return s.unit === 'kb' ? t('sizeKb', { n: formatNumber(s.value, locale) }) : t('sizeMb', { n: formatNumber(s.value, locale) });
  };
  const [first, ...rest] = files;
  return (
    <section className="rec-doc-preview-wrap" aria-labelledby={titleId}>
      <h2 id={titleId} className="sr-only">
        {t('files', { count: files.length, n: formatNumber(files.length, locale) })}
      </h2>
      <FileTile file={first!} n={1} size={size} big onOpen={onOpen} />
      {rest.length > 0 ? (
        <ul className="rec-doc-thumbs">
          {rest.map((f, i) => (
            <li key={f.id}>
              <FileTile file={f} n={i + 2} size={size} onOpen={onOpen} />
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}

function FileTile({
  file,
  n,
  size,
  big,
  onOpen,
}: {
  file: DocumentFile;
  n: number;
  size: (b: number) => string;
  big?: boolean;
  onOpen: (f: DocumentFile, n: number) => void;
}) {
  const t = useTranslations('record.doc');
  const locale = useLocale() as Locale;
  const image = isImageMime(file.mime);
  // photos are read into a blob: URL (never the signed link in the DOM); a failed read falls back to the icon
  const blob = useSignedBlob(file.id, file.url, image);
  const caption = image ? t('imageFile', { size: size(file.sizeBytes) }) : t('pdfFile', { size: size(file.sizeBytes) });
  const inner = (
    <>
      {blob.src ? (
        // eslint-disable-next-line @next/next/no-img-element -- on-device blob: URL, never optimized or cached
        <img src={blob.src} alt="" className="rec-doc-img" />
      ) : (
        <Icon name={image ? 'image' : 'fileDoc'} size={big ? 40 : 24} strokeWidth={1.6} />
      )}
      {big ? <span className="rec-doc-caption">{file.url ? caption : t('noPreview')}</span> : null}
    </>
  );
  const cls = big ? 'rec-doc-preview' : 'rec-doc-thumb';
  return file.url ? (
    <button
      type="button"
      className={blob.src ? `${cls} has-image` : cls}
      aria-label={image ? t('preview', { n: formatNumber(n, locale) }) : t('open', { n: formatNumber(n, locale) })}
      onClick={() => onOpen(file, n)}
    >
      {inner}
    </button>
  ) : (
    <div className={cls}>{inner}</div>
  );
}

function DetailsCard({ doc, onEdit }: { doc: RecordDocument; onEdit: () => void }) {
  const t = useTranslations('record.doc');
  const tk = useTranslations('record.kinds');
  const locale = useLocale() as Locale;
  const titleId = useId();
  return (
    <section className="rec-doc-section" aria-labelledby={titleId}>
      <div className="rec-doc-section-head">
        <h2 id={titleId} className="rec-doc-h2">
          {t('details')}
        </h2>
        <button type="button" className="hrec-edit" onClick={onEdit}>
          {t('edit')}
        </button>
      </div>
      <Card className="rec-doc-card">
        <div className="hrec-rows">
          <div className="hrec-row">
            <span className="hrec-row-label">{t('fields.kind')}</span>
            <span className="rec-row-value">{tk(doc.kind)}</span>
          </div>
          {DETAIL_KEYS.map((key) => {
            // once values were read from the document, date / centre / doctor show in that section instead
            if (doc.extracted && (doc.reviewState === 'confirmed' || doc.reviewState === 'needs_review') && key !== 'title' && key !== 'note') return null;
            const v = detailValue(doc, key);
            if (!v) return null;
            const shown = key === 'date' || key === 'ended_on' ? formatLongDate(fromApiDate(v), locale) : v;
            return (
              <div key={key} className="hrec-row">
                <span className="hrec-row-label">{t(`fields.${key}`)}</span>
                <span className="rec-row-value">{shown}</span>
              </div>
            );
          })}
        </div>
      </Card>
    </section>
  );
}

function gaText(weeks: number, days: number, t: ReturnType<typeof useTranslations<'record.doc'>>, locale: Locale) {
  return t('gaValue', { weeks: formatNumber(weeks, locale), days: formatNumber(days, locale) });
}

function DatingCard({ id, onApply }: { id: number; onApply: () => void }) {
  const t = useTranslations('record.doc');
  const locale = useLocale() as Locale;
  const dating = useRecordDating(id, true);
  const answer = useAnswerRecordDating(id);
  const titleId = useId();
  const o = dating.data;
  if (!o || o.state === 'unavailable' || o.state === 'applied') return null;
  if (o.state === 'dismissed') {
    return <p className="rec-doc-note">{t('dating.dismissed')}</p>;
  }
  const p = o.proposed;
  const date = (v: string) => formatLongDate(fromApiDate(v), locale);
  return (
    <section className="rec-doc-section" aria-labelledby={titleId}>
      <h2 id={titleId} className="rec-doc-h2">
        {t('dating.title')}
      </h2>
      <Card className="rec-doc-card rec-dating">
        <p className="rec-dating-offer">
          {p && o.scanDate ? t('dating.offer', { scan: date(o.scanDate), ga: gaText(p.gaWeeks, p.gaDays, t, locale) }) : o.message}
        </p>
        {p ? (
          <div className="rec-dating-compare">
            <div className="rec-dating-col is-new">
              <span className="rec-dating-label">{t('dating.proposed')}</span>
              <span className="rec-dating-value">{t('dating.due', { date: date(p.dueDate) })}</span>
              <span className="rec-dating-sub">{t('dating.today', { ga: gaText(p.weeksToday, p.daysToday, t, locale) })}</span>
            </div>
            {o.current ? (
              <div className="rec-dating-col">
                <span className="rec-dating-label">{t('dating.current')}</span>
                <span className="rec-dating-value">{t('dating.due', { date: date(o.current.dueDate) })}</span>
              </div>
            ) : null}
          </div>
        ) : null}
        {answer.isError ? (
          <p className="hrec-error" role="alert">
            {t('dating.error')}
          </p>
        ) : null}
        <div className="rec-read-actions">
          <PrimaryButton icon="check" onClick={onApply}>
            {t('dating.apply')}
          </PrimaryButton>
          <SecondaryButton variant="text" loading={answer.isPending} onClick={() => answer.mutate('dismiss')}>
            {t('dating.dismiss')}
          </SecondaryButton>
        </div>
      </Card>
    </section>
  );
}

function WhereUsed({ doc }: { doc: RecordDocument }) {
  const t = useTranslations('record.doc.used');
  const titleId = useId();
  return (
    <section className="rec-doc-section" aria-labelledby={titleId}>
      <h2 id={titleId} className="rec-doc-h2">
        {t('title')}
      </h2>
      <Card className="rec-doc-card">
        {doc.whereUsed.length === 0 ? (
          <p className="hrec-empty">{t('none')}</p>
        ) : (
          <ul className="rec-used">
            {doc.whereUsed.map((w) => (
              <li key={`${w.type}-${w.targetId}`} className="rec-used-row">
                <IconCircle icon={w.type === 'claim' ? 'shield' : 'heart'} tone={w.type === 'claim' ? 'period' : 'bloom'} size="md" />
                <span className="rec-used-text">
                  <span className="rec-used-title">{w.type === 'claim' ? t('claim') : t('pregnancy')}</span>
                  <span className="rec-used-sub">
                    {w.type === 'pregnancy' ? t('pregnancyBody') : w.state === 'waiting' ? t('waiting') : t('attached')}
                  </span>
                </span>
                <StatusPill tone={w.state === 'waiting' ? 'warm' : 'success'}>
                  {w.state === 'waiting' ? t('waiting') : w.state === 'applied' ? t('applied') : t('attached')}
                </StatusPill>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </section>
  );
}

function DocumentSheets({
  doc,
  sheet,
  previewFile,
  onSheet,
  onConsented,
}: {
  doc: RecordDocument;
  sheet: SheetId | null;
  previewFile: DocumentFile | null;
  onSheet: (s: SheetId | null) => void;
  /** Consent accepted → read the document again. */
  onConsented: () => void;
}) {
  const t = useTranslations('record.doc');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const remove = useDeleteRecordDocument(doc.id);
  const answer = useAnswerRecordDating(doc.id);
  const dating = useRecordDating(doc.id, doc.kind === 'imaging' && doc.reviewState === 'confirmed');
  const close = () => onSheet(null);
  const due = dueOf(dating.data);
  return (
    <>
      <EditSheet open={sheet === 'edit'} doc={doc} onClose={close} />
      <PreviewSheet open={sheet === 'preview'} file={previewFile} onClose={close} />
      <DocConsentSheet
        open={sheet === 'consent'}
        onClose={close}
        onGranted={() => {
          close();
          onConsented();
        }}
      />
      <ConfirmSheet
        open={sheet === 'delete'}
        title={t('deleteTitle')}
        body={t('deleteBody')}
        yes={t('deleteYes')}
        no={t('deleteNo')}
        danger
        busy={remove.isPending}
        error={remove.isError ? t('deleteError') : null}
        onYes={() => remove.mutate(undefined, { onSuccess: () => router.replace('/record/timeline') })}
        onClose={close}
      />
      <ConfirmSheet
        open={sheet === 'dating'}
        title={t('dating.confirmTitle')}
        body={due ? t('dating.confirmBody', { date: formatLongDate(fromApiDate(due), locale) }) : t('dating.confirmTitle')}
        yes={t('dating.confirmYes')}
        no={t('dating.confirmNo')}
        busy={answer.isPending}
        error={answer.isError ? t('dating.error') : null}
        onYes={() => answer.mutate('apply', { onSuccess: close })}
        onClose={close}
      />
    </>
  );
}

const dueOf = (o: DatingOffer | undefined): string | null => o?.proposed?.dueDate ?? null;

/**
 * The in-app photo viewer (audit L2): a sheet inside the React tree — so the app lock unmounts it — showing the
 * photo from a blob: URL that is revoked when it closes. Signed links never open in a new tab.
 */
function PreviewSheet({ open, file, onClose }: { open: boolean; file: DocumentFile | null; onClose: () => void }) {
  const t = useTranslations('record.doc');
  return (
    <AppSheet open={open && !!file} onClose={onClose} size="full" title={t('previewTitle')}>
      {open && file ? <PreviewBody file={file} /> : null}
    </AppSheet>
  );
}

function PreviewBody({ file }: { file: DocumentFile }) {
  const t = useTranslations('record.doc');
  const blob = useSignedBlob(file.id, file.url, true);
  if (blob.src) {
    // eslint-disable-next-line @next/next/no-img-element -- on-device blob: URL
    return <img src={blob.src} alt="" className="rec-lightbox-img" />;
  }
  return <p className={blob.failed ? 'hrec-error' : 'hrec-empty'}>{blob.failed ? t('noPreview') : t('loading')}</p>;
}
