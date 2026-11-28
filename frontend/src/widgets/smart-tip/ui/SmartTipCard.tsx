'use client';

import { useLocale, useTranslations } from 'next-intl';

import { localizeFeedText, type FeedItem } from '@/entities/message';
import type { Locale } from '@/shared/i18n';
import { Icon } from '@/shared/ui';
import '@/shared/ui/styles/home.css';
import '@/shared/ui/styles/sec.css';
import '@/shared/ui/styles/ch.css';

interface SmartTipCardProps {
  /** The `smart_tip` winner of GET /messages/feed; null when the engine served none (or it was closed). */
  item?: FeedItem | null;
  /** The resolved CTA of `item`; null = the message has no usable button. */
  onCta?: (() => void) | null;
  onDismiss?: () => void;
}

/**
 * Smart tip («نکته هوشمند») — the message engine's `smart_tip` slot, and nothing else (D-216): the phase message, a
 * symptom logged today and the symptom combinations are engine rules (`tip.*`) an admin edits and approves. The
 * rule's title heads the card, its body is the paragraph, an optional CTA and a dismiss link close it. Without a
 * served tip it renders nothing rather than a generic health claim (§11).
 */
export function SmartTipCard({ item = null, onCta = null, onDismiss }: SmartTipCardProps) {
  const t = useTranslations('home');
  const tCommon = useTranslations('common.actions');
  const loc = useLocale() as Locale;
  if (!item) return null;

  return (
    <div className="sec-tight" data-feed-slot="smart_tip">
      <div className="card pad-card-sm home-tip-card">
        <div className="home-tip-head">
          <span className="home-tip-badge" aria-hidden>
            <Icon name="sparkle" size={15} fill="currentColor" strokeWidth={0} />
          </span>
          <span className="home-tip-title">{item.title ? localizeFeedText(item.title, loc) : t('smartTip.title')}</span>
        </div>
        {item.body && <p className="sub home-tip-body">{localizeFeedText(item.body, loc)}</p>}
        {onCta && item.ctaText ? (
          <button type="button" className="ch-link" data-action="feed_cta" onClick={onCta}>
            {localizeFeedText(item.ctaText, loc)}
          </button>
        ) : null}
        {item.dismissible && onDismiss ? (
          <button type="button" className="ch-link" data-action="feed_dismiss" onClick={onDismiss}>
            {tCommon('close')}
          </button>
        ) : null}
      </div>
    </div>
  );
}
