'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef } from 'react';

import { localizeFeedText, telHref, type FeedItem } from '@/entities/message';
import type { Locale } from '@/shared/i18n';
import { PrimaryButton, SecondaryButton, UrgentCard } from '@/shared/ui';
import '@/shared/ui/styles/del.css';

interface UrgentModalProps {
  item: FeedItem;
  /** Posts `POST /alerts/{id}/ack`; the modal closes when the feed no longer carries the item. */
  onAck: () => void;
  pending: boolean;
  failed: boolean;
}

/**
 * The urgent health message (SPEC §5.4.1). It cannot be closed: no backdrop tap, no Escape, no close button; the
 * only way out is «دیدم», which acknowledges the alert on the server. The call button (115 or the doctor) is a
 * plain `tel:` link, always available.
 */
export function UrgentModal({ item, onAck, pending, failed }: UrgentModalProps) {
  const t = useTranslations('home.feed');
  const loc = useLocale() as Locale;
  const dialogRef = useRef<HTMLDivElement>(null);
  const contact = item.evidence?.contact ?? '';
  const tel = telHref(contact);

  // Focus lands inside the dialog; Escape is swallowed so it cannot dismiss anything underneath.
  useEffect(() => {
    dialogRef.current?.querySelector<HTMLElement>('.nb-urgent-action button')?.focus({ preventScroll: true });
    const block = (e: KeyboardEvent) => {
      if (e.key === 'Escape') e.preventDefault();
    };
    window.addEventListener('keydown', block);
    return () => window.removeEventListener('keydown', block);
  }, []);

  return (
    <div ref={dialogRef} role="alertdialog" aria-modal="true" aria-label={item.title} className="del-backdrop is-urgent" data-feed-slot="urgent_modal">
      <div className="del-card">
        <UrgentCard
          title={localizeFeedText(item.title, loc)}
          hotlines={
            tel
              ? [{ number: tel.slice(4), label: t('call'), display: localizeFeedText(contact, loc) }]
              : undefined
          }
          hotlinesLabel={t('call')}
          action={
            <PrimaryButton loading={pending} onClick={onAck}>
              {t('ack')}
            </PrimaryButton>
          }
        >
          <p>{localizeFeedText(item.body, loc)}</p>
          {!tel && contact ? <p>{localizeFeedText(contact, loc)}</p> : null}
          {item.evidence?.seen ? (
            <p>
              <b>{t('seen')}: </b>
              {localizeFeedText(item.evidence.seen, loc)}
            </p>
          ) : null}
          {item.evidence?.confidence ? (
            <p>
              <b>{t('confidence')}: </b>
              {localizeFeedText(item.evidence.confidence, loc)}
            </p>
          ) : null}
        </UrgentCard>
        {failed ? (
          <p className="del-error" role="alert">
            {t('ackError')}
          </p>
        ) : null}
      </div>
    </div>
  );
}

interface InAppModalProps {
  item: FeedItem;
  /** The CTA, already resolved to an action (null = the message has no usable button). */
  onCta: (() => void) | null;
  onClose: () => void;
}

/** The in-app `modal` slot: a dismissible message with an optional CTA. */
export function InAppModal({ item, onCta, onClose }: InAppModalProps) {
  const t = useTranslations('common.actions');
  const loc = useLocale() as Locale;
  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={item.title}
      className="del-backdrop"
      data-feed-slot="modal"
      onClick={onClose}
    >
      <div className="card del-card" onClick={(e) => e.stopPropagation()}>
        <div className="del-title">{localizeFeedText(item.title, loc)}</div>
        <p className="del-body">{localizeFeedText(item.body, loc)}</p>
        <div className="del-btns">
          {onCta && item.ctaText ? (
            <PrimaryButton onClick={onCta}>{localizeFeedText(item.ctaText, loc)}</PrimaryButton>
          ) : null}
          <SecondaryButton onClick={onClose}>{t('close')}</SecondaryButton>
        </div>
      </div>
    </div>
  );
}
