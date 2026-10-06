'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { type CreatedShareLink, type ReportSelection, useCreateShareLink } from '@/entities/health-record';
import { usePlusLocked } from '@/entities/plus';
import { ApiError } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { EmptyState, InfoNote, PrimaryButton, SecondaryButton, Skeleton } from '@/shared/ui';
import { plusDenialOf } from '@/shared/ui/plus-gate';

import { sharedReportUrl } from '../model/selection';

interface ShareLinkSheetProps {
  open: boolean;
  onClose: () => void;
  selection: ReportSelection;
}

const PDF_SHARE = 'plus.pdf_share';

/**
 * «اشتراک لینک»: creates the 7-day link once per opening (Plus — free users get the Plus pitch, as does a 402), then
 * shows the URL with copy / native share. The token lives only in this component's memory and the URL it shows.
 */
export function ShareLinkSheet({ open, onClose, selection }: ShareLinkSheetProps) {
  const t = useTranslations('recordExport.shareSheet');
  const tp = useTranslations('recordExport.preview');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const locked = usePlusLocked(PDF_SHARE);
  const create = useCreateShareLink();
  const [link, setLink] = useState<CreatedShareLink | null>(null);
  const [copied, setCopied] = useState(false);
  const started = useRef(false);

  useEffect(() => {
    if (!open) {
      started.current = false;
      setLink(null);
      setCopied(false);
      create.reset();
      return;
    }
    if (locked || started.current) return;
    started.current = true;
    create.mutate(selection, { onSuccess: setLink });
    // Only (re)run when the sheet opens; the selection is fixed while it is open.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, locked]);

  const denied = locked || plusDenialOf(create.error) !== null;
  const limit = create.error instanceof ApiError && create.error.response?.status === 409;
  const url = link && typeof window !== 'undefined' ? sharedReportUrl(window.location.origin, loc, link.token) : '';

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
    } catch {
      setCopied(false);
    }
  };
  const shareNative = async () => {
    try {
      await navigator.share({ title: t('shareText'), url });
    } catch {
      // dismissed
    }
  };

  let body;
  if (denied) {
    body = (
      <EmptyState
        icon="crown"
        title={t('plusTitle')}
        body={t('plusBody')}
        action={
          <PrimaryButton block={false} onClick={() => router.push('/plus')}>
            {t('plusCta')}
          </PrimaryButton>
        }
      />
    );
  } else if (create.isError) {
    body = (
      <EmptyState
        icon="warning"
        title={limit ? t('limit') : t('error')}
        action={
          limit ? (
            <SecondaryButton block={false} onClick={() => router.push('/profile/privacy')}>
              {t('manage')}
            </SecondaryButton>
          ) : (
            <PrimaryButton block={false} icon="refresh" onClick={() => create.mutate(selection, { onSuccess: setLink })}>
              {tp('retry')}
            </PrimaryButton>
          )
        }
      />
    );
  } else if (!link) {
    body = (
      <div className="rx-share-loading" role="status" aria-label={t('creating')}>
        <Skeleton shape="block" />
        <Skeleton width="medium" />
      </div>
    );
  } else {
    body = (
      <div className="rx-share">
        <p className="rx-share-body">{t('body', { date: formatLongDate(new Date(link.expiresAt), loc) })}</p>
        <output className="rx-share-url" dir="ltr">
          {url}
        </output>
        <div className="rx-share-actions">
          <PrimaryButton icon={copied ? 'check' : 'copy'} onClick={() => void copy()}>
            {copied ? t('copied') : t('copy')}
          </PrimaryButton>
          {typeof navigator !== 'undefined' && 'share' in navigator ? (
            <SecondaryButton icon="share" onClick={() => void shareNative()}>
              {t('shareNative')}
            </SecondaryButton>
          ) : null}
        </div>
        <InfoNote icon="lock">
          <button type="button" className="rx-link" onClick={() => router.push('/profile/privacy')}>
            {t('manage')}
          </button>
        </InfoNote>
      </div>
    );
  }

  return (
    <AppSheet open={open} onClose={onClose} size="half" title={t('title')}>
      {body}
    </AppSheet>
  );
}
