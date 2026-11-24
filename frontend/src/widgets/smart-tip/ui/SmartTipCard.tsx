'use client';

import clsx from 'clsx';
import { useMemo } from 'react';
import { useLocale, useTranslations } from 'next-intl';

import { localizeFeedText, selectSmartTip, useDailyMessage, type FeedItem } from '@/entities/message';
import type { Locale } from '@/shared/i18n';
import { Icon } from '@/shared/ui';
import '@/shared/ui/styles/home.css';
import '@/shared/ui/styles/skeleton.css';
import '@/shared/ui/styles/sec.css';
import '@/shared/ui/styles/has.css';
import './tip.css';
import '@/shared/ui/styles/ch.css';

interface SmartTipCardProps {
  /** The `smart_tip` winner of GET /messages/feed; null when the engine served none (or it was closed). */
  item?: FeedItem | null;
  /**
   * May the card fall back to the old `/messages/daily` content when there is no `item`? The caller says no while
   * the feed is still loading (so the old copy never flashes before the engine's) and after she closed the
   * engine's tip (so closing it does not just swap in another one).
   */
  fallback?: boolean;
  /** The resolved CTA of `item`; null = the message has no usable button. */
  onCta?: (() => void) | null;
  onDismiss?: () => void;
}

/**
 * Smart tip («نکته هوشمند») — one card for both sources (user request, 2026-11-24).
 *
 * The message engine's `smart_tip` slot wins: the rule's title heads the card, its body is the paragraph, an
 * optional CTA and a dismiss link close it. A day the engine has no matching tip, the card is written by the older
 * `/messages/daily` bundle instead (paragraph, highlighted action, extra insights, phase chip), as it was before the
 * feed. With neither it renders nothing rather than a generic health claim (§11).
 */
export function SmartTipCard({ item = null, fallback = true, onCta = null, onDismiss }: SmartTipCardProps) {
  const t = useTranslations('home');
  const tCommon = useTranslations('common.actions');
  const loc = useLocale() as Locale;
  // Asked only when it is needed: with an engine tip on screen the legacy bundle is never fetched.
  const legacyWanted = !item && fallback;
  const { data: message, isLoading } = useDailyMessage(undefined, legacyWanted);
  const tip = useMemo(() => (legacyWanted ? selectSmartTip(message) : null), [legacyWanted, message]);

  if (item) {
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

  if (!legacyWanted || (!isLoading && !tip)) return null;
  const phaseLabel = message?.phaseLabel ?? null;

  return (
    <div className="sec-tight" data-feed-slot="smart_tip_legacy">
      {/* «هالهٔ هوش»: the only fully engine-written card, marked by an animated
          gradient hairline (gradient scarcity §10.2 holds — it is an edge). */}
      <div className="card pad-card-sm home-tip-card">
        <div className="home-tip-head">
          <span className="home-tip-badge" aria-hidden>
            <Icon name="sparkle" size={15} fill="currentColor" strokeWidth={0} />
          </span>
          <span className="home-tip-title">{t('smartTip.title')}</span>
          {phaseLabel && <span className="home-tip-chip">{phaseLabel}</span>}
        </div>

        {isLoading || !tip ? (
          <SmartTipSkeleton />
        ) : (
          <>
            <p className={clsx('sub', 'home-tip-body', tip.action && 'has-action')}>
              {tip.body}
            </p>
            {tip.action && (
              <div className="tip-action">
                <span className="tip-action-icon">
                  <Icon name="sparkle" size={20} fill="currentColor" strokeWidth={0} />
                </span>
                <span className="tip-action-text">{tip.action}</span>
              </div>
            )}
            {tip.extras.length > 0 && (
              <ul className="home-tip-extras">
                {tip.extras.map(extra => (
                  <li key={extra} className="home-tip-extra">
                    <span className="home-tip-bullet" />
                    <span>{extra}</span>
                  </li>
                ))}
              </ul>
            )}
          </>
        )}
      </div>
    </div>
  );
}

/** Placeholder lines while today's message loads — same rhythm as the real copy. */
function SmartTipSkeleton() {
  return (
    <div aria-hidden className="home-skel">
      {['100%', '92%', '64%'].map(width => (
        <span key={width} className="skeleton-line" style={{ width }} />
      ))}
      <span className="skeleton-line home-skel-block" />
    </div>
  );
}
