'use client';

import { useLocale, useTranslations } from 'next-intl';
import { type ChangeEvent, useRef, useState } from 'react';

import {
  DOCUMENT_SLOTS,
  type DocumentSlot,
  JOIN_BOOKING_MODES,
  type JoinApplication,
  type JoinBookingMode,
  type JoinFile,
  useBusinessTerms,
  useRemoveJoinDocument,
  useSaveJoinStep,
  useSubmitJoin,
  useUploadJoinDocument,
} from '@/entities/city-service';
import { ApiError, type ApiEnvelope, getApiErrorCode } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { Checkbox, Icon, InfoNote, PrimaryButton, ProgressBar, type RadioCardOption, RadioCardGroup, SecondaryButton, StatusPill } from '@/shared/ui';

import { fileProblem, groupErrors, type JoinStep, step4Body, step4From } from '../model/join';
import { useAutosave } from '../model/use-autosave';
import { SaveHint, useErrorText } from './parts';

/** The per-field `errors` of a 422 `application_incomplete`. */
function incompleteErrors(error: unknown): Record<string, string[]> | null {
  if (!(error instanceof ApiError) || getApiErrorCode(error) !== 'application_incomplete') return null;
  const body = error.response?.data as ApiEnvelope<unknown> | undefined;
  return body?.errors ?? {};
}

/** Only an http(s) signed link is ever put in an anchor. */
function safeHref(url: string): string | null {
  try {
    const u = new URL(url);
    return u.protocol === 'https:' || u.protocol === 'http:' ? u.href : null;
  } catch {
    return null;
  }
}

function DocRow({ app, slot, required }: { app: JoinApplication; slot: DocumentSlot; required: boolean }) {
  const t = useTranslations('cityJoin');
  const locale = useLocale() as Locale;
  const errorText = useErrorText();
  const upload = useUploadJoinDocument(app.id);
  const remove = useRemoveJoinDocument(app.id);
  const input = useRef<HTMLInputElement>(null);
  const [progress, setProgress] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const doc: JoinFile | null = app.documents[slot];
  const href = doc ? safeHref(doc.url) : null;
  const title = t(`docs.slots.${slot}.title`);

  const onPick = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    const problem = fileProblem(file, 'doc');
    if (problem) {
      setError(t(`errors.${problem}`));
      return;
    }
    setError(null);
    setProgress(0);
    try {
      await upload.mutateAsync({ slot, file, onProgress: setProgress });
    } catch (e) {
      setError(errorText(e));
    } finally {
      setProgress(null);
    }
  };

  return (
    <li className="cjn-doc">
      <span className="cjn-doc-ic">
        <Icon name={doc ? 'checkCircle' : 'fileDoc'} size={20} />
      </span>
      <span className="cjn-doc-text">
        <span className="cjn-doc-title">
          {title} <span className="cjn-hint">· {required ? t('docs.required') : t('docs.optional')}</span>
        </span>
        <span className="cjn-hint">{doc ? t('docs.uploaded') : t(`docs.slots.${slot}.hint`)}</span>
        {progress !== null ? (
          <ProgressBar value={Math.round(progress * 100)} max={100} label={t('docs.uploading', { percent: formatNumber(Math.round(progress * 100), locale) })} />
        ) : null}
        {error ? (
          <span role="alert" className="cjn-err">
            {error}
          </span>
        ) : null}
      </span>
      <span className="cjn-doc-actions">
        {doc && href ? (
          <a className="cjn-link" href={href} target="_blank" rel="noopener noreferrer nofollow" aria-label={t('docs.viewLabel', { title })}>
            {t('docs.view')}
          </a>
        ) : null}
        <SecondaryButton block={false} loading={progress !== null} onClick={() => input.current?.click()}>
          {doc ? t('docs.replace') : t('docs.upload')}
        </SecondaryButton>
        {doc ? (
          <button
            type="button"
            className="cjn-x is-static"
            aria-label={t('docs.removeLabel', { title })}
            disabled={remove.isPending}
            onClick={() => remove.mutate(slot, { onError: (e) => setError(errorText(e)) })}
          >
            <Icon name="trash" size={16} />
          </button>
        ) : null}
      </span>
      <input ref={input} className="sr-only" type="file" accept="image/jpeg,image/png,image/webp,application/pdf,.pdf" tabIndex={-1} aria-hidden onChange={(e) => void onPick(e)} />
    </li>
  );
}

/** Step 4 «مدارک» (nbl_Dir_JoinDocs): booking mode, private documents, terms and the final submit. */
export function StepDocs({
  app,
  onGoStep,
  onSaveError,
}: {
  app: JoinApplication;
  onGoStep: (step: JoinStep) => void;
  onSaveError: (e: unknown) => void;
}) {
  const t = useTranslations('cityJoin');
  const locale = useLocale() as Locale;
  const errorText = useErrorText();
  const save = useSaveJoinStep(app.id);
  const submit = useSubmitJoin(app.id);
  const [form, setForm] = useState(() => step4From(app));
  const [missing, setMissing] = useState<Record<string, string[]> | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [termsOpen, setTermsOpen] = useState(false);
  const terms = useBusinessTerms(locale).data ?? [];
  const { state, flush } = useAutosave(form, (v) =>
    save.mutateAsync({ step: 4, body: step4Body(v) }).catch((e) => {
      onSaveError(e);
      throw e;
    }),
  );
  const needsHealth = app.category_slugs.includes(app.requirements.health_cert_category);

  const modes: RadioCardOption<JoinBookingMode>[] = JOIN_BOOKING_MODES.map((m) => ({
    value: m,
    title: t(`docs.modes.${m}.title`),
    description: t(`docs.modes.${m}.body`),
    icon: m === 'phone' ? 'phone' : 'calendar',
  }));

  const send = async () => {
    setSubmitError(null);
    setMissing(null);
    if (!(await flush())) return;
    try {
      await submit.mutateAsync();
    } catch (e) {
      const errors = incompleteErrors(e);
      if (errors) setMissing(errors);
      else {
        onSaveError(e);
        setSubmitError(errorText(e));
      }
    }
  };

  const grouped = missing ? groupErrors(missing) : [];

  return (
    <div className="cjn-stack">
      <section className="cjn-stack" aria-label={t('docs.bookingTitle')}>
        <h2 className="cjn-h">{t('docs.bookingTitle')}</h2>
        <RadioCardGroup
          label={t('docs.bookingTitle')}
          options={modes}
          value={form.bookingMode as JoinBookingMode | null}
          onChange={(v) => setForm((f) => ({ ...f, bookingMode: v }))}
        />
      </section>

      <section className="card fld-card cjn-fields" aria-label={t('docs.docsTitle')}>
        <div>
          <h2 className="cjn-h">{t('docs.docsTitle')}</h2>
          <p className="cjn-hint cjn-private">
            <Icon name="lock" size={14} />
            {t('docs.docsHint')}
          </p>
        </div>
        <ul className="cjn-docs">
          {DOCUMENT_SLOTS.map((slot) => (
            <DocRow key={slot} app={app} slot={slot} required={slot === 'licence' || (slot === 'health_cert' && needsHealth)} />
          ))}
        </ul>
      </section>

      <div className="cjn-terms">
        <Checkbox
          checked={form.terms}
          onCheckedChange={(v) => setForm((f) => ({ ...f, terms: v }))}
          label={
            <span>
              {t('docs.termsLead')}
              <button type="button" className="cjn-link" onClick={() => setTermsOpen(true)}>
                {t('docs.termsLink')}
              </button>
              {t('docs.termsTail')}
            </span>
          }
        />
      </div>
      <InfoNote icon="shield">{t('docs.note')}</InfoNote>

      {grouped.length > 0 ? (
        <section className="cjn-missing" role="alert" aria-label={t('docs.missingTitle')}>
          <StatusPill tone="warm" icon="warning">
            {t('docs.missingTitle')}
          </StatusPill>
          <ul>
            {grouped.map((e) => (
              <li key={e.field}>
                <span>{e.message}</span>
                <button type="button" className="cjn-link" onClick={() => onGoStep(e.step)}>
                  {t('docs.goStep', { n: formatNumber(e.step, locale) })}
                </button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      <SaveHint state={state} />
      {submitError ? (
        <p className="cjn-err" role="alert">
          {submitError}
        </p>
      ) : null}
      <PrimaryButton loading={submit.isPending} onClick={() => void send()}>
        {submit.isPending ? t('docs.submitting') : t('docs.submit')}
      </PrimaryButton>

      <AppSheet open={termsOpen} onClose={() => setTermsOpen(false)} size="half" title={t('docs.termsSheet')}>
        {terms.length > 0 ? (
          terms.map((term) => (
            <div key={term.id} className="cjn-term">
              {term.heading ? <strong>{term.heading}</strong> : null}
              <p className="cjn-sheet-copy">{term.body}</p>
            </div>
          ))
        ) : (
          <p className="cjn-sheet-copy">{t('docs.termsSheetBody')}</p>
        )}
      </AppSheet>
    </div>
  );
}
