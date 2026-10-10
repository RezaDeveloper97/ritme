'use client';

import { useTranslations } from 'next-intl';
import { useLocale } from 'next-intl';
import { type FormEvent, useEffect, useId, useState } from 'react';

import { useOpenSharedReportByCode } from '@/entities/health-record';
import { ApiError } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { retryAfterSeconds, usePageHide } from '@/shared/lib/page-hide';
import { InfoNote, PrimaryButton, SecondaryButton } from '@/shared/ui';

import { SharedReportBody, SharedReportShell } from './SharedReportPage';

/**
 * `/d` — the doctor types the patient's 24h code (CB-REC-05, D-71): POST /shared-reports/code, then the same
 * read-only paper as the QR link. The code and the report live in the mutation's memory only (no URL, no storage);
 * every refusal is the same «wrong or expired» answer, 429 says to wait.
 */
export function DoctorCodePage() {
  const t = useTranslations('record.doctor');
  const open = useOpenSharedReportByCode();
  const [code, setCode] = useState('');
  const inputId = useId();
  const errorId = useId();
  const loc = useLocale() as Locale;
  // 429: submit stays off until Retry-After passes; typing does not clear it (L4)
  const [blockedUntil, setBlockedUntil] = useState(0);
  const [now, setNow] = useState(() => Date.now());
  const waitSeconds = Math.max(0, Math.ceil((blockedUntil - now) / 1000));
  const blocked = waitSeconds > 0;

  useEffect(() => {
    if (!blockedUntil) return;
    const id = window.setInterval(() => {
      const ms = Date.now();
      setNow(ms);
      if (ms >= blockedUntil) {
        setBlockedUntil(0);
        window.clearInterval(id);
      }
    }, 1000);
    return () => window.clearInterval(id);
  }, [blockedUntil]);

  // a closed tab / bfcache never brings the report or the typed code back (shared clinic PC)
  usePageHide(() => {
    open.reset();
    setCode('');
  });

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const trimmed = code.trim();
    if (!trimmed || open.isPending || blocked) return;
    open.mutate(trimmed, {
      onError: (err) => {
        if (err instanceof ApiError && err.response?.status === 429) {
          const seconds = retryAfterSeconds(err.response.data, err.response.headers);
          const ms = Date.now();
          setNow(ms);
          setBlockedUntil(ms + seconds * 1000);
        }
      },
    });
  };

  if (open.data) {
    return (
      <SharedReportShell title={t('title')}>
        <SharedReportBody report={open.data} />
        <div className="rx-actions is-inline">
          <SecondaryButton
            variant="text"
            onClick={() => {
              open.reset();
              setCode('');
            }}
          >
            {t('another')}
          </SecondaryButton>
        </div>
      </SharedReportShell>
    );
  }

  const status = open.error instanceof ApiError ? open.error.response?.status : undefined;
  let error: string | null = null;
  if (open.isError) {
    if (status === 404 || status === 422) error = t('notFound');
    else if (status === 429) error = blocked ? t('tooManyWait', { s: formatNumber(waitSeconds, loc) }) : null;
    else if (status === 503) error = t('unavailable');
    else error = t('error');
  }

  return (
    <SharedReportShell title={t('title')}>
      <form className="nb-card rx-code-form" onSubmit={submit} noValidate>
        <p className="rx-code-sub">{t('sub')}</p>
        <label htmlFor={inputId} className="hrec-label">
          {t('codeLabel')}
        </label>
        <input
          id={inputId}
          className="hrec-input rx-code-input"
          dir="ltr"
          value={code}
          maxLength={32}
          autoComplete="off"
          autoCapitalize="characters"
          autoCorrect="off"
          spellCheck={false}
          placeholder="XXXX-XXXX"
          aria-invalid={open.isError}
          aria-describedby={error ? errorId : undefined}
          onChange={(e) => {
            setCode(e.target.value);
            // a wrong code's message clears on typing; a 429 stays until Retry-After passes
            if (open.isError && status !== 429) open.reset();
          }}
        />
        {error ? (
          <p id={errorId} className="rx-error" role="alert">
            {error}
          </p>
        ) : null}
        <PrimaryButton type="submit" loading={open.isPending} disabled={!code.trim() || blocked}>
          {t('submit')}
        </PrimaryButton>
      </form>
      <InfoNote icon="lock">{t('privacy')}</InfoNote>
    </SharedReportShell>
  );
}
