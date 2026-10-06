'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useLabConsent, useSetLabConsent } from '@/entities/lab';
import { getApiErrorCode, getApiErrorMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { closeSheet, type SheetContentProps } from '@/shared/sheet';
import {
  Checkbox,
  type IconName,
  IconCircle,
  PrimaryButton,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  type Tone,
} from '@/shared/ui';

/** Icons of the consent points, in the order the catalog lists them (sent · stored · control · limits). */
const POINT_LOOKS: readonly { icon: IconName; tone: Tone }[] = [
  { icon: 'export', tone: 'brand' },
  { icon: 'lock', tone: 'data' },
  { icon: 'trash', tone: 'warm' },
  { icon: 'info', tone: 'danger' },
  { icon: 'shield', tone: 'brand' },
];

/** Visible sheet heading («قبل از شروع»). */
export function LabConsentTitle() {
  return <>{useTranslations('labConsent')('title')}</>;
}

/**
 * `?sheet=lab-consent` (nbl_Lab_Consent): the versioned consent text of
 * `ai_lab_analysis` as the server has it in force (`GET /consents/{code}` —
 * body + points, never hard-coded), an explicit checkbox, then
 * `PUT /consents/{code} {granted, version}`. `arg=upload` goes on to the
 * upload screen after accepting. A granted consent can be withdrawn here too.
 * The consent is stored server-side, so it is asked again only when a new
 * version is in force (the artboard's «دفعه بعد نپرس» is therefore implicit).
 */
export function LabConsentSheet({ arg }: SheetContentProps) {
  const t = useTranslations('labConsent');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const consent = useLabConsent();
  const save = useSetLabConsent();
  const [agreed, setAgreed] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const done = () => {
    if (arg === 'upload') router.push('/labs/new');
    else closeSheet();
  };

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
        <p className="lab-error" role="alert">
          {t('loadError')}
        </p>
        <SecondaryButton icon="refresh" onClick={() => void consent.refetch()}>
          {t('retry')}
        </SecondaryButton>
      </div>
    );
  }

  const c = consent.data;
  const submit = (granted: boolean) => {
    setError(null);
    save.mutate(
      { granted, version: c.version },
      {
        onSuccess: () => {
          setAgreed(false);
          if (granted) done();
        },
        onError: (e) => {
          setError(getApiErrorCode(e) === 'consent_version_stale' ? t('stale') : (getApiErrorMessage(e) ?? t('saveError')));
        },
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
        {c.points.map((p, i) => {
          const look = POINT_LOOKS[i % POINT_LOOKS.length]!;
          return (
            <li key={i} className="lab-consent-point">
              <IconCircle icon={look.icon} tone={look.tone} size="sm" />
              <span>{p}</span>
            </li>
          );
        })}
      </ul>
      <p className="lab-consent-version">{t('version', { v: formatNumber(c.version, locale) })}</p>

      {c.granted && !c.needsConsent ? (
        <>
          <p className="lab-consent-granted" role="status">
            {t('granted')}
          </p>
          {error ? (
            <p className="lab-error" role="alert">
              {error}
            </p>
          ) : null}
          <div className="lab-consent-actions">
            <PrimaryButton onClick={done}>{t('continue')}</PrimaryButton>
            <SecondaryButton variant="text" loading={save.isPending} onClick={() => submit(false)}>
              {t('withdraw')}
            </SecondaryButton>
          </div>
        </>
      ) : (
        <>
          <Checkbox className="lab-consent-check" checked={agreed} onCheckedChange={setAgreed} label={t('agree')} />
          {error ? (
            <p className="lab-error" role="alert">
              {error}
            </p>
          ) : null}
          <div className="lab-consent-actions">
            <PrimaryButton disabled={!agreed} loading={save.isPending} onClick={() => submit(true)}>
              {t('accept')}
            </PrimaryButton>
            <SecondaryButton variant="text" onClick={() => closeSheet()}>
              {t('cancel')}
            </SecondaryButton>
          </div>
        </>
      )}
    </div>
  );
}
