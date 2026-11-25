'use client';

import { useLocale, useTranslations } from 'next-intl';

import { localizeFeedText, type FeedItem } from '@/entities/message';
import type { Locale } from '@/shared/i18n';
import { PrimaryButton, SecondaryButton } from '@/shared/ui';
import '@/shared/ui/styles/del.css';

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
