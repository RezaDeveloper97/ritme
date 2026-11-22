'use client';

import { useLocale, useTranslations } from 'next-intl';
import { usePathname } from 'next/navigation';
import { useState } from 'react';

import { type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { Checkbox, IconCircle, PrimaryButton, SecondaryButton } from '@/shared/ui';

import { useAcceptPendingConsents, usePendingConsents } from '../api/consents';
import { shouldShowPending } from '../model/consents';
import '@/shared/ui/styles/lab.css';

/**
 * The one-time acceptance sheet of the mandatory signup consents (T-M10-23,
 * DECISIONS.md H-18) for accounts made before they existed: on the next app
 * open, GET /consents/pending lists the texts not yet accepted (title, body,
 * points as the server has them in force — never hard-coded), one checkbox
 * accepts them all with the versions shown. «Later» hides it until the next
 * app open; nothing is blocked server-side meanwhile. Mount once, inside the
 * app lock (the layout, beside `SheetHost`).
 */
export function PendingConsentsSheet() {
  usePathname(); // re-render on navigation, so the query starts right after sign-in
  const pending = usePendingConsents();
  const [dismissed, setDismissed] = useState(false);
  const open = shouldShowPending(pending.data, dismissed);
  return (
    <AppSheet open={open} onClose={() => setDismissed(true)} size="full">
      {open && pending.data ? <PendingConsentsBody pending={pending.data} onDone={() => setDismissed(true)} /> : null}
    </AppSheet>
  );
}

function PendingConsentsBody({
  pending,
  onDone,
}: {
  pending: NonNullable<ReturnType<typeof usePendingConsents>['data']>;
  onDone: () => void;
}) {
  const t = useTranslations('auth.pending');
  const locale = useLocale() as Locale;
  const accept = useAcceptPendingConsents();
  const [agreed, setAgreed] = useState(false);

  return (
    <div className="lab-consent">
      <div className="lab-consent-head">
        <IconCircle icon="shield" tone="data" size="lg" outlined />
        <p className="lab-consent-lead">{t('lead')}</p>
      </div>
      <h2 className="lab-consent-title">{t('title')}</h2>
      {pending.map((c) => (
        <section key={c.code} aria-label={c.title || undefined}>
          {c.title ? <h3 className="lab-consent-title">{c.title}</h3> : null}
          {c.body ? <p className="lab-consent-body">{c.body}</p> : null}
          {c.points.length ? (
            <ul className="lab-consent-points">
              {c.points.map((p, i) => (
                <li key={i} className="lab-consent-point">
                  <IconCircle icon="check" tone="brand" size="sm" />
                  <span>{p}</span>
                </li>
              ))}
            </ul>
          ) : null}
          <p className="lab-consent-version">{t('version', { v: formatNumber(c.version, locale) })}</p>
        </section>
      ))}
      <Checkbox className="lab-consent-check" checked={agreed} onCheckedChange={setAgreed} label={t('agree')} />
      {accept.isError ? (
        <p className="lab-error" role="alert">
          {t('error')}
        </p>
      ) : null}
      <div className="lab-consent-actions">
        <PrimaryButton
          disabled={!agreed}
          loading={accept.isPending}
          onClick={() => accept.mutate(pending, { onSuccess: onDone })}
        >
          {t('accept')}
        </PrimaryButton>
        <SecondaryButton variant="text" onClick={onDone}>
          {t('later')}
        </SecondaryButton>
      </div>
    </div>
  );
}
