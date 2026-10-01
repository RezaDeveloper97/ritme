'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import {
  ALERT_LEVELS_V2,
  type AlertLegendRow,
  type AlertLevelV2,
  type PregnancyAlertV2,
  useMarkAllAlertsRead,
  usePregnancyAlertsV2,
} from '@/entities/pregnancy';
import { addDays, today as todayDate, toApiDate } from '@/shared/lib/date';
import { Link, useDirection, useRouter } from '@/shared/i18n';
import {
  Card,
  EmptyState,
  HeaderButton,
  Icon,
  type IconName,
  IconCircle,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  type Tone,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { useAlertV2Action } from '../api/actions';
import { alertActionStyle, groupAlertsByDay, resolveAlertAction, showsFacts, visibleActions } from '../model/alerts';

type ActionKey = NonNullable<ReturnType<typeof resolveAlertAction>>['key'];
type T = ReturnType<typeof useTranslations<'pregnancyV2'>>;

/** Four levels, four accents (PregFull_Alerts): turquoise · violet · amber · rose. */
const LEVEL_TONE: Record<AlertLevelV2, Tone> = {
  info: 'data',
  suggestion: 'brand',
  follow_up: 'warm',
  urgent: 'danger',
};
const LEVEL_ICON: Record<AlertLevelV2, IconName> = {
  info: 'info',
  suggestion: 'sparkle',
  follow_up: 'warning',
  urgent: 'phone',
};

function Shell({ children, windowNote }: { children: React.ReactNode; windowNote?: string }) {
  const t = useTranslations('pregnancyV2');
  const router = useRouter();
  return (
    <div className="view preg-page pgn-screen">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('alerts.title')}
          subtitle={windowNote}
          onBack={() => router.push('/pregnancy')}
          backLabel={t('common.back')}
          action={<HeaderButton icon="cog" label={t('alerts.settings')} onClick={() => router.push('/profile/notifications')} />}
        />
        <div className="pgn-body">{children}</div>
      </div>
      <BottomNav />
    </div>
  );
}

function dayLabel(day: string, fallback: string | null, t: T): string | null {
  const today = todayDate();
  if (day === toApiDate(today)) return t('alerts.today');
  if (day === toApiDate(addDays(today, -1))) return t('alerts.yesterday');
  return fallback;
}

// ── Card ───────────────────────────────────────────────────────
function AlertCard({ alert, dateText, t }: { alert: PregnancyAlertV2; dateText: string | null; t: T }) {
  const mutation = useAlertV2Action();
  const isRtl = useDirection() === 'rtl';
  const [noted, setNoted] = useState(false);
  const style = alertActionStyle(alert.level);
  const actions = visibleActions(
    alert.level,
    alert.actions
      .map((a) => ({ a, r: resolveAlertAction(a, alert) }))
      .filter((x): x is { a: typeof x.a; r: NonNullable<typeof x.r> } => x.r !== null),
  );
  const label = (key: ActionKey, admin: string | null) => admin ?? t(`alerts.actions.${key}`);
  const phone = alert.contact?.phone?.replace(/[^\d+]/g, '');
  const facts = showsFacts(alert.level) && (alert.whatWeSaw || alert.howSure);
  const tone = LEVEL_TONE[alert.level];
  const titleId = `alert-${alert.id}`;
  const pill = <span className={clsx('pgn-opill', `nb-tone-${tone}`)}>{t(`alerts.levelsShort.${alert.level}`)}</span>;

  // Info / suggestion: one compact row (pill · title · link), as drawn.
  if (style !== 'button') {
    return (
      <Card as="article" className={clsx('pgn-alert-row', alert.isAcked && 'is-acked')} aria-labelledby={titleId}>
        {pill}
        <div className="pgn-row-text">
          <h3 id={titleId} className="pgn-row-title">
            {alert.title}
          </h3>
          {(alert.advice || dateText) && <span className="pgn-row-desc">{alert.advice ?? dateText}</span>}
        </div>
        {actions.map(({ a, r }) =>
          r.kind === 'link' ? (
            <Link key={a.key} href={r.href} className="pgn-sect-link">
              {label(r.key, a.label)}
            </Link>
          ) : r.kind === 'tel' ? (
            <a key={a.key} href={r.href} className="pgn-sect-link">
              {label(r.key, a.label)}
            </a>
          ) : (
            <button
              key={a.key}
              type="button"
              className="pgn-sect-link"
              disabled={mutation.isPending}
              onClick={() => mutation.mutate({ id: alert.id, action: r.action })}
            >
              {label(r.key, a.label)}
            </button>
          ),
        )}
      </Card>
    );
  }

  const acks = actions.filter(({ r }) => r.kind === 'server' && r.action === 'ack');
  const main = actions.filter((x) => !acks.includes(x));

  return (
    <Card
      as="article"
      className={clsx('pgn-sect pgn-alert', `nb-tone-${tone}`, alert.isAcked && 'is-acked')}
      aria-labelledby={titleId}
    >
      <div className="pgn-sect-head">
        {pill}
        {dateText && <span className="pgn-meta">{dateText}</span>}
      </div>
      <h3 id={titleId} className="pgn-sect-title">
        {alert.title}
      </h3>
      {facts && (
        <dl className="pgn-facts">
          {alert.whatWeSaw && (
            <div>
              <dt>{t('alerts.whatWeSaw')}:</dt> <dd>{alert.whatWeSaw}</dd>
            </div>
          )}
          {alert.howSure && (
            <div>
              <dt>{t('alerts.howSure')}:</dt> <dd>{alert.howSure}</dd>
            </div>
          )}
        </dl>
      )}
      {alert.advice && <p className="pgn-body-text">{alert.advice}</p>}
      {alert.contact && (
        <p className="pgn-contact">
          <IconCircle icon="phone" tone={tone} size="sm" />
          <span className="pgn-contact-text">{alert.contact.text}</span>
          {phone && (
            <a href={`tel:${phone}`} className="pgn-sect-link" dir="ltr">
              {t('alerts.call')}
            </a>
          )}
        </p>
      )}
      {main.length > 0 && (
        <div className="pgn-alert-actions">
          {main.map(({ a, r }, i) => {
            const cls = clsx(
              'nb-btn is-block',
              alert.level === 'urgent' && i === 0 ? 'is-primary' : 'is-outline pgn-btn-outline',
            );
            if (r.kind === 'server') {
              return (
                <button
                  key={a.key}
                  type="button"
                  className={cls}
                  disabled={mutation.isPending || (r.action === 'add_to_visit_note' && noted)}
                  onClick={() =>
                    mutation.mutate(
                      { id: alert.id, action: r.action },
                      { onSuccess: () => r.action === 'add_to_visit_note' && setNoted(true) },
                    )
                  }
                >
                  {r.action === 'add_to_visit_note' && noted ? <Icon name="check" size={18} strokeWidth={2.4} /> : null}
                  {label(r.key, a.label)}
                </button>
              );
            }
            return r.kind === 'tel' ? (
              <a key={a.key} href={r.href} className={cls}>
                {label(r.key, a.label)}
              </a>
            ) : (
              <Link key={a.key} href={r.href} className={cls}>
                {label(r.key, a.label)}
                <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={16} />
              </Link>
            );
          })}
        </div>
      )}
      {acks.map(({ a, r }) => (
        <button
          key={a.key}
          type="button"
          className="pgn-textbtn"
          disabled={mutation.isPending || alert.isAcked}
          onClick={() => r.kind === 'server' && mutation.mutate({ id: alert.id, action: r.action })}
        >
          {label(r.key, a.label)}
        </button>
      ))}
      <p role="status" className="pgn-status empty:hidden">
        {noted ? t('alerts.addedToNote') : mutation.isError ? t('common.saveError') : ''}
      </p>
    </Card>
  );
}

// ── Legend + disclaimer ────────────────────────────────────────
function Legend({ rows, t }: { rows: AlertLegendRow[]; t: T }) {
  const byLevel = new Map(rows.map((r) => [r.level, r]));
  const levels: AlertLevelV2[] = rows.length > 0 ? rows.map((r) => r.level) : [...ALERT_LEVELS_V2];
  return (
    <Card as="section" className="pgn-sect" aria-labelledby="alerts-legend">
      <h2 id="alerts-legend" className="pgn-sect-title">
        {t('alerts.legendTitle')}
      </h2>
      <ul className="pgn-rows">
        {levels.map((level) => {
          const row = byLevel.get(level);
          return (
            <li key={level} className="pgn-row">
              <IconCircle icon={LEVEL_ICON[level]} tone={LEVEL_TONE[level]} size="sm" />
              <span className="pgn-row-text">
                <b className="pgn-row-title">{row?.label ?? t(`alerts.levelsShort.${level}`)}</b>
                <span className="pgn-row-desc">{row?.text ?? t(`alerts.legend.${level}`)}</span>
              </span>
            </li>
          );
        })}
      </ul>
    </Card>
  );
}

// ── Page ───────────────────────────────────────────────────────
/** «هشدارها و پیام‌ها» — `/pregnancy/alerts` (PregFull_Alerts), four message levels. */
export function PregnancyAlertsPage() {
  const t = useTranslations('pregnancyV2');
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  const query = usePregnancyAlertsV2();
  const markRead = useMarkAllAlertsRead();
  const marked = useRef(false);
  const hasUnread = query.data?.alerts.some((a) => !a.isRead) ?? false;

  // Seeing the list reads it: mark once per visit, only when something is unread.
  useEffect(() => {
    if (!hasUnread || marked.current) return;
    marked.current = true;
    markRead.mutate();
  }, [hasUnread, markRead]);

  if (!mounted || query.isLoading) {
    return (
      <Shell>
        <SkeletonGroup label={t('common.loading')} className="pgn-skel">
          <Skeleton shape="card" className="pgn-skel-hero" />
          <Skeleton shape="block" />
          <Skeleton shape="block" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }

  if (query.isError) {
    return (
      <Shell>
        <Card className="pgn-state" role="alert">
          <span className="pgn-state-disc" aria-hidden>
            <Icon name="warning" size={24} />
          </span>
          <p className="pgn-state-text">{t('common.loadError')}</p>
          <SecondaryButton icon="refresh" block={false} onClick={() => void query.refetch()}>
            {t('common.retry')}
          </SecondaryButton>
        </Card>
      </Shell>
    );
  }

  const data = query.data;
  if (!data) {
    return (
      <Shell>
        <Card>
          <EmptyState
            icon="heart"
            title={t('common.notActive')}
            action={
              <Link href="/pregnancy/setup" className="nb-btn is-primary is-block">
                {t('today.setupCta')}
              </Link>
            }
          />
        </Card>
      </Shell>
    );
  }

  const groups = groupAlertsByDay(data.alerts);

  return (
    <Shell windowNote={t('alerts.window', { days: data.windowDays })}>
      {groups.length === 0 && (
        <Card>
          <EmptyState icon="bell" title={t('alerts.empty')} />
        </Card>
      )}
      {groups.map((g) => {
        const text = dayLabel(g.day, g.label, t);
        return (
          <section key={g.day || 'unknown'} className="pgn-stack" aria-label={text ?? undefined}>
            {g.alerts.map((a) => (
              <AlertCard key={a.id} alert={a} dateText={text} t={t} />
            ))}
          </section>
        );
      })}
      <Legend rows={data.legend} t={t} />
      <p className="pgn-disclaimer">{t('alerts.disclaimer')}</p>
    </Shell>
  );
}
