'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import {
  type AlertItem,
  type AlertLevel,
  type AlertListState,
  callHref,
  canAck,
  canAddToVisitNote,
  canDismiss,
  evidenceRows,
  groupByLevel,
  useAlertAction,
  useAlerts,
} from '@/entities/alert';
import { feedLinkTarget } from '@/entities/message';
import { Link, useRouter } from '@/shared/i18n';
import {
  Card,
  EmptyState,
  HeaderButton,
  Icon,
  IconCircle,
  ScreenHeader,
  SecondaryButton,
  SegmentedTabs,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  type Tone,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

type T = ReturnType<typeof useTranslations<'alerts'>>;

/** Four levels, four accents: rose (urgent) · amber · violet · turquoise. */
const LEVEL_TONE: Record<AlertLevel, Tone> = { urgent: 'danger', follow_up: 'warm', suggestion: 'brand', info: 'data' };

function Pill({ level, t }: { level: AlertLevel; t: T }) {
  return <span className={clsx('pgn-opill', `nb-tone-${LEVEL_TONE[level]}`)}>{t(`levels.${level}`)}</span>;
}

function AlertCard({ alert, history, t }: { alert: AlertItem; history: boolean; t: T }) {
  const mutation = useAlertAction();
  const tone = LEVEL_TONE[alert.level];
  const titleId = `alert-${alert.id}`;
  const evidence = evidenceRows(alert);
  const tel = callHref(alert);
  const contact = alert.evidence?.contact ?? null;
  const cta = feedLinkTarget(alert.ctaDeeplink);
  const ctaHref = cta?.kind === 'route' && alert.ctaText ? cta.path : null;
  const busy = mutation.isPending;
  const run = (action: 'ack' | 'visit-note' | 'dismiss') => mutation.mutate({ id: alert.id, action });
  const compact = alert.level === 'info' || alert.level === 'suggestion';
  const acked = alert.state === 'acknowledged' || alert.state === 'added_to_visit_note';
  const noted = alert.state === 'added_to_visit_note';
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
      {(canAddToVisitNote(alert) || noted) && (
        <button type="button" className="nb-btn is-block is-outline pgn-btn-outline" disabled={busy || noted} onClick={() => run('visit-note')}>
          {noted ? <Icon name="check" size={18} strokeWidth={2.4} /> : null}
          {noted ? t('actions.visitNoteDone') : t('actions.visitNote')}
        </button>
      )}
    </>
  );

  // Info / suggestion: one compact row; actions are text links.
  if (compact) {
    return (
      <Card as="article" className={clsx('pgn-alert-row', acked && 'is-acked')} aria-labelledby={titleId}>
        <Pill level={alert.level} t={t} />
        <div className="pgn-row-text">
          <h3 id={titleId} className="pgn-row-title">{alert.title}</h3>
          {alert.body && <span className="pgn-row-desc">{alert.body}</span>}
          {!history && (
            <span className="pgn-row-desc">
              {ctaHref && alert.ctaText && <Link href={ctaHref} className="pgn-sect-link">{alert.ctaText}</Link>}{' '}
              {canAck(alert) && <button type="button" className="pgn-sect-link" disabled={busy} onClick={() => run('ack')}>{t('actions.ack')}</button>}{' '}
              {canAddToVisitNote(alert) && <button type="button" className="pgn-sect-link" disabled={busy} onClick={() => run('visit-note')}>{t('actions.visitNote')}</button>}{' '}
              {canDismiss(alert) && <button type="button" className="pgn-sect-link" disabled={busy} onClick={() => run('dismiss')}>{t('actions.dismiss')}</button>}
            </span>
          )}
        </div>
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

function Shell({ children, tab, onTab }: { children: React.ReactNode; tab: AlertListState; onTab: (tab: AlertListState) => void }) {
  const t = useTranslations('alerts');
  const router = useRouter();
  return (
    <div className="view preg-page pgn-screen">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('title')}
          subtitle={t('subtitle')}
          onBack={() => router.push('/')}
          backLabel={t('back')}
          action={<HeaderButton icon="cog" label={t('settings')} onClick={() => router.push('/profile/notifications')} />}
        />
        <div className="pgn-body">
          <SegmentedTabs
            label={t('tabs.label')}
            value={tab}
            onChange={onTab}
            tabs={[
              { value: 'open', label: t('tabs.open') },
              { value: 'history', label: t('tabs.history') },
            ]}
          />
          {children}
          <p className="pgn-disclaimer">{t('disclaimer')}</p>
        </div>
      </div>
      <BottomNav />
    </div>
  );
}

/** «هشدارها و پیام‌ها» — `/alerts`: every mode's inbox, sections by level, open and history (SPEC §5.4.1). */
export function AlertsPage() {
  const t = useTranslations('alerts');
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  const [tab, setTab] = useState<AlertListState>('open');
  const query = useAlerts(tab);

  if (!mounted || query.isLoading) {
    return (
      <Shell tab={tab} onTab={setTab}>
        <SkeletonGroup label={t('loading')} className="pgn-skel">
          <Skeleton shape="card" />
          <Skeleton shape="block" />
          <Skeleton shape="block" />
        </SkeletonGroup>
      </Shell>
    );
  }

  if (query.isError) {
    return (
      <Shell tab={tab} onTab={setTab}>
        <Card className="pgn-state" role="alert">
          <span className="pgn-state-disc" aria-hidden>
            <Icon name="warning" size={24} />
          </span>
          <p className="pgn-state-text">{t('loadError')}</p>
          <SecondaryButton icon="refresh" block={false} onClick={() => void query.refetch()}>
            {t('retry')}
          </SecondaryButton>
        </Card>
      </Shell>
    );
  }

  const items = query.data?.pages.flatMap((p) => p.items) ?? [];
  const sections = groupByLevel(items);

  return (
    <Shell tab={tab} onTab={setTab}>
      {sections.length === 0 && (
        <Card>
          <EmptyState icon="bell" title={t(`empty.${tab}`)} />
        </Card>
      )}
      {sections.map((s) => (
        <section key={s.level} className="pgn-stack" aria-label={t(`levels.${s.level}`)}>
          {s.items.map((a) => (
            <AlertCard key={a.id} alert={a} history={tab === 'history'} t={t} />
          ))}
        </section>
      ))}
      {query.hasNextPage && (
        <SecondaryButton block={false} onClick={() => void query.fetchNextPage()} disabled={query.isFetchingNextPage}>
          {t('loadMore')}
        </SecondaryButton>
      )}
    </Shell>
  );
}
