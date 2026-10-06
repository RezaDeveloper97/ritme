'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useRef, useState, type ChangeEvent } from 'react';

import { type Lab, LAB_PLUS_FEATURE } from '@/entities/lab';
import { PlusFeatureGate } from '@/entities/plus';
import { type Locale, useDirection } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import {
  Card,
  ChipGroup,
  Icon,
  PillChip,
  PrimaryButton,
  TileButton,
} from '@/shared/ui';

import { useUploadLab } from '../api/upload';
import { uploadErrorOf, type UploadError } from '../model/errors';
import { shrinkImage } from '../model/shrink';
import {
  ACCEPT_IMAGES,
  ACCEPT_PDF,
  addPages,
  type AddError,
  type DraftErrors,
  type DraftPage,
  hasPdfHeader,
  kindOf,
  movePage,
  removePage,
  type UploadLimits,
  validateDraft,
} from '../model/validation';
import { DateSheet } from './DateSheet';

interface LabUploadFormProps {
  limits: UploadLimits;
  /** The 202 answer: the lab to follow (processing / verify / failed). */
  onUploaded: (lab: Lab) => void;
  /** 403 consent_required — open the consent sheet; the draft stays. */
  onConsentRequired: () => void;
}

let seq = 0;
const nextId = () => `page-${Date.now().toString(36)}-${++seq}`;

/** Object URLs of the photo pages, revoked when a page goes or the form unmounts. */
function usePreviews(pages: readonly DraftPage[]): Record<string, string> {
  const [urls, setUrls] = useState<Record<string, string>>({});
  useEffect(() => {
    const made: Record<string, string> = {};
    for (const p of pages) if (p.kind === 'image') made[p.id] = URL.createObjectURL(p.file);
    setUrls(made);
    return () => Object.values(made).forEach((u) => URL.revokeObjectURL(u));
  }, [pages]);
  return urls;
}

/**
 * The lab upload form of nbl_Lab_Upload: camera / gallery / PDF pickers, up to
 * five pages with preview, reorder and remove, the lab type, its date and
 * fasting, then «تحلیل کن» with the upload progress. Checks mirror the server
 * (`model/validation.ts`); photos above the size limit are shrunk on the device
 * first. Files never leave the page except in the upload itself.
 */
export function LabUploadForm({ limits, onUploaded, onConsentRequired }: LabUploadFormProps) {
  const t = useTranslations('labs.upload');
  const locale = useLocale() as Locale;
  const rtl = useDirection() === 'rtl';
  const [pages, setPages] = useState<DraftPage[]>([]);
  const [category, setCategory] = useState<string | null>(null);
  const [takenOn, setTakenOn] = useState<string | null>(() => toApiDate(today()));
  const [fasting, setFasting] = useState<boolean | null>(null);
  const [pickError, setPickError] = useState<AddError | null>(null);
  const [errors, setErrors] = useState<DraftErrors>({});
  const [serverError, setServerError] = useState<UploadError | null>(null);
  const [dateOpen, setDateOpen] = useState(false);
  const [preparing, setPreparing] = useState(false);
  const camera = useRef<HTMLInputElement>(null);
  const gallery = useRef<HTMLInputElement>(null);
  const pdf = useRef<HTMLInputElement>(null);
  const previews = usePreviews(pages);
  const upload = useUploadLab();
  const full = pages.length >= limits.maxFiles;
  const busy = preparing || upload.isPending;

  const onPick = async (event: ChangeEvent<HTMLInputElement>) => {
    const input = event.target;
    const picked = Array.from(input.files ?? []);
    input.value = '';
    if (!picked.length) return;
    setPreparing(true);
    try {
      const ready: File[] = [];
      let early: AddError | null = null;
      for (const f of picked) {
        const kind = kindOf(f);
        if (kind === 'image') ready.push(await shrinkImage(f, limits.maxImageBytes));
        else if (kind === 'pdf') {
          const head = new Uint8Array(await f.slice(0, 16).arrayBuffer());
          if (hasPdfHeader(head)) ready.push(f);
          else early ??= 'file_type';
        } else ready.push(f);
      }
      const r = addPages(pages, ready, limits, nextId);
      setPages(r.pages);
      setPickError(early ?? r.error);
      setErrors((e) => ({ ...e, files: undefined }));
      setServerError(null);
    } finally {
      setPreparing(false);
    }
  };

  const submit = () => {
    const draft = { pages, category, takenOn, fasting };
    const found = validateDraft(draft, limits, toApiDate(today()));
    setErrors(found);
    setServerError(null);
    if (Object.keys(found).length) return;
    upload.mutate(draft, {
      onSuccess: onUploaded,
      onError: (error) => {
        const e = uploadErrorOf(error);
        setServerError(e);
        if (e.kind === 'consent') onConsentRequired();
      },
    });
  };

  const fileMessage = (code: string | undefined): string | null => {
    if (!code) return null;
    return t(`errors.${code}` as 'errors.file_type', {
      max: formatNumber(limits.maxFiles, locale),
      image: formatNumber(Math.round(limits.maxImageBytes / 1024 / 1024), locale),
      pdf: formatNumber(Math.round(limits.maxPdfBytes / 1024 / 1024), locale),
    });
  };
  const filesError = serverError?.fields.files ?? fileMessage(errors.files) ?? fileMessage(pickError ?? undefined);
  const categoryError = serverError?.fields.category ?? (errors.category ? t('errors.category_invalid') : null);
  const dateError = serverError?.fields.taken_on ?? (errors.taken_on ? t('errors.date_future') : null);
  const generalError = useMemo(() => {
    if (!serverError) return null;
    switch (serverError.kind) {
      case 'validation':
      case 'plus':
        return Object.keys(serverError.fields).length ? null : serverError.message;
      case 'consent':
        return t('errors.consent');
      case 'network':
        return t('errors.network');
      case 'busy':
      case 'unavailable':
        return serverError.message ?? t('errors.busy');
      default:
        return serverError.message ?? t('errors.unknown');
    }
  }, [serverError, t]);

  const fastingLabel = fasting === null ? t('fastingUnset') : fasting ? t('yes') : t('no');

  return (
    <>
      <div className="lab-up-body">
        <div className="lab-up-sources">
          <TileButton layout="card" icon="camera" tone="brand" label={t('camera')} disabled={full || busy} onClick={() => camera.current?.click()} />
          <TileButton layout="card" icon="image" tone="data" label={t('gallery')} disabled={full || busy} onClick={() => gallery.current?.click()} />
          <TileButton layout="card" icon="fileDoc" tone="warm" label={t('pdf')} disabled={full || busy} onClick={() => pdf.current?.click()} />
        </div>
        <input ref={camera} className="sr-only" type="file" accept="image/*" capture="environment" tabIndex={-1} aria-hidden onChange={(e) => void onPick(e)} />
        <input ref={gallery} className="sr-only" type="file" accept={ACCEPT_IMAGES} multiple tabIndex={-1} aria-hidden onChange={(e) => void onPick(e)} />
        <input ref={pdf} className="sr-only" type="file" accept={ACCEPT_PDF} tabIndex={-1} aria-hidden onChange={(e) => void onPick(e)} />

        <Card as="section" className="lab-up-card" aria-labelledby="lab-up-pages">
          <div className="lab-up-card-head">
            <h2 id="lab-up-pages" className="lab-card-title">{t('pages')}</h2>
            <span className="lab-up-count">{t('pageCount', { count: pages.length, n: formatNumber(pages.length, locale) })}</span>
          </div>
          <ol className="lab-up-pages">
            {pages.map((p, i) => (
              <li key={p.id} className="lab-up-page">
                <div className={clsx('lab-up-thumb', p.kind === 'pdf' && 'is-pdf')}>
                  {p.kind === 'image' && previews[p.id] ? (
                    // eslint-disable-next-line @next/next/no-img-element -- local object URL, never optimized
                    <img src={previews[p.id]} alt="" className="lab-up-img" />
                  ) : (
                    <Icon name={p.kind === 'pdf' ? 'fileDoc' : 'image'} size={28} strokeWidth={1.6} />
                  )}
                  <span className="lab-up-num">{formatNumber(i + 1, locale)}</span>
                </div>
                <button type="button" className="lab-up-remove" aria-label={t('remove', { n: formatNumber(i + 1, locale) })} onClick={() => setPages((cur) => removePage(cur, p.id))} disabled={busy}>
                  <Icon name="x" size={12} strokeWidth={3} />
                </button>
                {pages.length > 1 ? (
                  <div className="lab-up-move">
                    <button type="button" className="lab-up-move-btn" aria-label={t('moveEarlier', { n: formatNumber(i + 1, locale) })} disabled={i === 0 || busy} onClick={() => setPages((cur) => movePage(cur, p.id, -1))}>
                      <Icon name={rtl ? 'chevronRight' : 'chevronLeft'} size={16} />
                    </button>
                    <button type="button" className="lab-up-move-btn" aria-label={t('moveLater', { n: formatNumber(i + 1, locale) })} disabled={i === pages.length - 1 || busy} onClick={() => setPages((cur) => movePage(cur, p.id, 1))}>
                      <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={16} />
                    </button>
                  </div>
                ) : null}
              </li>
            ))}
            {!full ? (
              <li className="lab-up-page">
                <button type="button" className="lab-up-add" onClick={() => gallery.current?.click()} disabled={busy}>
                  <Icon name="plus" size={22} />
                  <span>{pages.length ? t('nextPage') : t('firstPage')}</span>
                </button>
              </li>
            ) : null}
          </ol>
          <p className="lab-up-hint">{t('limitsHint', { max: formatNumber(limits.maxFiles, locale) })}</p>
          {filesError ? (
            <p className="lab-error" role="alert">
              {filesError}
            </p>
          ) : null}
        </Card>

        <Card as="section" className="lab-up-card" aria-labelledby="lab-up-type">
          <h2 id="lab-up-type" className="lab-card-title">{t('type')}</h2>
          <ChipGroup label={t('type')}>
            {limits.categories.map((c) => (
              <PillChip key={c} pressed={category === c} onPressedChange={() => setCategory(c)} disabled={busy}>
                {t(`categories.${c}` as 'categories.blood')}
              </PillChip>
            ))}
          </ChipGroup>
          {categoryError ? (
            <p className="lab-error" role="alert">
              {categoryError}
            </p>
          ) : null}
        </Card>

        <div className="lab-up-pair">
          <button type="button" className="lab-up-field" onClick={() => setDateOpen(true)} disabled={busy}>
            <span className="lab-up-field-label">{t('date')}</span>
            <span className="lab-up-field-value">{takenOn ? formatLongDate(fromApiDate(takenOn), locale) : t('dateUnset')}</span>
          </button>
          <button
            type="button"
            className="lab-up-field"
            aria-pressed={fasting === true}
            onClick={() => setFasting((f) => (f === null ? true : !f))}
            disabled={busy}
          >
            <span className="lab-up-field-label">{t('fasting')}</span>
            <span className="lab-up-field-value">{fastingLabel}</span>
          </button>
        </div>
        {dateError ? (
          <p className="lab-error" role="alert">
            {dateError}
          </p>
        ) : null}

        <Card as="section" className="lab-up-card lab-up-tips" aria-labelledby="lab-up-tips">
          <h2 id="lab-up-tips" className="lab-card-title is-sm">{t('tipsTitle')}</h2>
          <ul className="lab-checks">
            {(['flat', 'columns', 'onePage', 'privacy'] as const).map((k) => (
              <li key={k} className="lab-check">
                <Icon name="check" size={14} strokeWidth={2.4} className="lab-check-icon" />
                <span>{t(`tips.${k}`)}</span>
              </li>
            ))}
          </ul>
        </Card>

        {serverError?.kind === 'plus' ? (
          <PlusFeatureGate feature={LAB_PLUS_FEATURE} denial={serverError.denial}>
            <p className="lab-up-plus">{t('plusNote')}</p>
          </PlusFeatureGate>
        ) : null}
        {generalError ? (
          <p className="lab-error is-block" role="alert">
            {generalError}
          </p>
        ) : null}
      </div>

      <div className="lab-footer">
        {upload.isPending ? (
          <div className="lab-up-progress" role="status">
            <span className="lab-up-progress-track">
              <span className="lab-up-progress-fill" style={{ inlineSize: `${Math.round(upload.progress * 100)}%` }} />
            </span>
            <span className="lab-up-progress-text">
              {upload.progress < 1
                ? t('uploading', { percent: formatNumber(Math.round(upload.progress * 100), locale) })
                : t('uploaded')}
            </span>
          </div>
        ) : null}
        <PrimaryButton onClick={submit} loading={busy} disabled={pages.length === 0}>
          {t('submit')}
        </PrimaryButton>
      </div>

      {dateOpen ? (
        <DateSheet
          title={t('date')}
          value={takenOn}
          onClose={() => setDateOpen(false)}
          onPick={(d) => {
            setTakenOn(d);
            setDateOpen(false);
            setErrors((e) => ({ ...e, taken_on: undefined }));
          }}
        />
      ) : null}
    </>
  );
}
