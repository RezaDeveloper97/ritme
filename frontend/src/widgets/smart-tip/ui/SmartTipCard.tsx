'use client';

import clsx from 'clsx';
import { useMemo } from 'react';
import { useTranslations } from 'next-intl';

import { selectSmartTip, useDailyMessage } from '@/entities/message';
import { Icon } from '@/shared/ui';

/**
 * Smart tip («نکته هوشمند») — fully driven by `/messages/daily`.
 *
 * Everything in this card — paragraph, highlighted action, extra insights and
 * the phase chip — comes from the server's message engine. While the request is
 * in flight the card shows a skeleton, and a day the engine has nothing to say
 * about renders no card at all rather than a generic health claim (§11).
 *
 * Shown on both the home feed and the analysis screen, so it lives here rather
 * than inside either screen slice.
 */
export function SmartTipCard() {
  const t = useTranslations('home');
  const { data: message, isLoading } = useDailyMessage();

  const tip = useMemo(() => selectSmartTip(message), [message]);
  const phaseLabel = message?.phaseLabel ?? null;

  if (!isLoading && !tip) return null;

  return (
    <div className="sec-tight">
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
