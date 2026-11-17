'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';

import { type AlertItem, type AlertLevel, callHref, canAck, canDismiss, evidenceRows, useAlertAction } from '@/entities/alert';
import { feedLinkTarget } from '@/entities/message';
import { Link } from '@/shared/i18n';
import { Card, Icon, IconCircle, type Tone } from '@/shared/ui';

// One health alert with its actions (تماس، دیدم، بستن), shown inside the notifications sheet.
// Moved here from the old `/alerts` screen: alerts no longer have a page of their own.

type T = ReturnType<typeof useTranslations<'alerts'>>;

/** Four levels, four accents: rose (urgent) · amber · violet · turquoise. */
const LEVEL_TONE: Record<AlertLevel, Tone> = { urgent: 'danger', follow_up: 'warm', suggestion: 'brand', info: 'data' };

function Pill({ level, t }: { level: AlertLevel; t: T }) {
  return <span className={clsx('pgn-opill', `nb-tone-${LEVEL_TONE[level]}`)}>{t(`levels.${level}`)}</span>;
}

export function AlertCard({ alert, history, t }: { alert: AlertItem; history: boolean; t: T }) {
  const mutation = useAlertAction();
  const tone = LEVEL_TONE[alert.level];
  const titleId = `alert-${alert.id}`;
  const evidence = evidenceRows(alert);
  const tel = callHref(alert);
  const contact = alert.evidence?.contact ?? null;
  const cta = feedLinkTarget(alert.ctaDeeplink);
  const ctaHref = cta?.kind === 'route' && alert.ctaText ? cta.path : null;
  const busy = mutation.isPending;
  const run = (action: 'ack' | 'dismiss') => mutation.mutate({ id: alert.id, action });
  const compact = alert.level === 'info' || alert.level === 'suggestion';
  const acked = alert.state === 'acknowledged' || alert.state === 'added_to_visit_note';
  const urgentOpen = alert.level === 'urgent' && alert.state === 'active' && !history;

  const actionButtons = history ? null : (
    <>
      {tel && contact && (
        <a href={tel} className="nb-btn is-block is-primary" dir="auto">
          <Icon name="phone" size={18} />
          {t('actions.call', { contact })}
        </a>
      )}
      {ctaHref && alert.ctaText && (
        <Link href={ctaHref} className="nb-btn is-block is-outline pgn-btn-outline">
          {alert.ctaText}
        </Link>
      )}
      {canAck(alert) && (
        <button type="button" className={clsx('nb-btn is-block', tel ? 'is-outline pgn-btn-outline' : 'is-primary')} disabled={busy} onClick={() => run('ack')}>
          {t('actions.ack')}
        </button>
      )}
    </>
  );

  // Info / suggestion: one compact row; a single check button marks it as read.
  if (compact) {
    return (
      <Card as="article" className={clsx('pgn-alert-row', acked && 'is-acked')} aria-labelledby={titleId}>
        <Pill level={alert.level} t={t} />
        <div className="pgn-row-text">
          <h3 id={titleId} className="pgn-row-title">{alert.title}</h3>
          {alert.body && <span className="pgn-row-desc">{alert.body}</span>}
          {!history && ctaHref && alert.ctaText && (
            <span className="pgn-row-desc">
              <Link href={ctaHref} className="pgn-sect-link">{alert.ctaText}</Link>
            </span>
          )}
        </div>
        {!history && (
          <button
            type="button"
            className={clsx('pgn-read-btn', acked && 'is-read')}
            disabled={busy || !canAck(alert)}
            aria-pressed={acked}
            aria-label={acked ? t('actions.read') : t('actions.markRead')}
            onClick={() => run('ack')}
          >
            <Icon name="check" size={18} strokeWidth={2.4} />
          </button>
        )}
        <p role="status" className="pgn-status empty:hidden">{mutation.isError ? t('saveError') : ''}</p>
      </Card>
    );
  }

  return (
    <Card as="article" className={clsx('pgn-sect pgn-alert', `nb-tone-${tone}`, acked && 'is-acked')} aria-labelledby={titleId}>
      <div className="pgn-sect-head">
        <Pill level={alert.level} t={t} />
        {alert.hitCount > 1 && <span className="pgn-meta">{t('seenTimes', { count: alert.hitCount })}</span>}
      </div>
      <h3 id={titleId} className="pgn-sect-title">{alert.title}</h3>
      {alert.body && <p className="pgn-body-text">{alert.body}</p>}
      {evidence.length > 0 && (
        <dl className="pgn-facts">
          {evidence.map((r) => (
            <div key={r.key}>
              <dt>{t(r.key === 'seen' ? 'whatWeSaw' : 'howSure')}:</dt> <dd>{r.value}</dd>
            </div>
          ))}
        </dl>
      )}
      {urgentOpen && (
        <p className="pgn-contact">
          <IconCircle icon="phone" tone={tone} size="sm" />
          <span className="pgn-contact-text">{t('urgentHint')}</span>
        </p>
      )}
      {!tel && contact && alert.level === 'urgent' && (
        <p className="pgn-contact">
          <IconCircle icon="phone" tone={tone} size="sm" />
          <span className="pgn-contact-text">{contact}</span>
        </p>
      )}
      {actionButtons && <div className="pgn-alert-actions">{actionButtons}</div>}
      {!history && canDismiss(alert) && (
        <button type="button" className="pgn-textbtn" disabled={busy} onClick={() => run('dismiss')}>
          {t('actions.dismiss')}
        </button>
      )}
      <p role="status" className="pgn-status empty:hidden">{mutation.isError ? t('saveError') : ''}</p>
    </Card>
  );
}
