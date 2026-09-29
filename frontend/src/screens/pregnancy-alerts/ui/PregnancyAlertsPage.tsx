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
import { Link, useDirection } from '@/shared/i18n';
import { openSheet } from '@/shared/sheet';
import { Icon } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { useAlertV2Action } from '../api/actions';
import { alertActionStyle, groupAlertsByDay, resolveAlertAction, showsFacts, visibleActions } from '../model/alerts';

type ActionKey = NonNullable<ReturnType<typeof resolveAlertAction>>['key'];
type T = ReturnType<typeof useTranslations<'pregnancyV2'>>;

function Shell({ children, windowNote }: { children: React.ReactNode; windowNote?: string }) {
  const t = useTranslations('pregnancyV2');
  const isRtl = useDirection() === 'rtl';
  return (
    <div className="view preg-page">
      <div className="scroll flex flex-col gap-3 pb-6">
        <header className="mx-4 mt-4 flex items-center gap-2">
          <Link href="/pregnancy" aria-label={t('common.back')} className="pg2-roundbtn no-underline">
            <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
          </Link>
          <div className="min-w-0 flex-1 text-center">
            <h1 className="text-[18px] font-black text-(--ink)">{t('alerts.title')}</h1>
            {windowNote && <p className="text-[12px] font-bold text-(--muted)">{windowNote}</p>}
          </div>
          <button
            type="button"
            className="pg2-roundbtn"
            aria-label={t('alerts.settings')}
            onClick={() => openSheet('notifications')}
          >
            <Icon name="cog" size={20} />
          </button>
        </header>
        {children}
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
  const worthFollowing = alert.level === 'follow_up' || alert.level === 'urgent';

  return (
    <article
      className={clsx('card pg2-alert pg2-level-' + alert.level, 'p-4', alert.isAcked && 'opacity-70')}
      aria-labelledby={`alert-${alert.id}`}
    >
      <div className="flex items-center gap-2">
        <span className="pg2-chip rounded-full px-2.5 py-0.5 text-[11px]">
          {worthFollowing && <Icon name="warning" size={13} />}
          {t(`alerts.levels.${alert.level}`)}
        </span>
        {dateText && <span className="ms-auto text-[11.5px] font-bold text-(--muted)">{dateText}</span>}
      </div>
      <h3 id={`alert-${alert.id}`} className="mt-2 text-[15px] font-black text-(--ink)">
        {alert.title}
      </h3>
      {facts && (
        <dl className="mt-3 grid grid-cols-2 gap-2 text-[12.5px] leading-6">
          {alert.whatWeSaw && (
            <div className="pg2-fact">
              <dt className="font-black text-(--brand)">{t('alerts.whatWeSaw')}</dt>
              <dd className="m-0 text-(--ink-2)">{alert.whatWeSaw}</dd>
            </div>
          )}
          {alert.howSure && (
            <div className="pg2-fact">
              <dt className="font-black text-(--brand)">{t('alerts.howSure')}</dt>
              <dd className="m-0 text-(--ink-2)">{alert.howSure}</dd>
            </div>
          )}
        </dl>
      )}
      {alert.advice && <p className="mt-2 text-[13px] leading-6 text-(--steel)">{alert.advice}</p>}
      {alert.contact && (
        <p className="pg2-tile mt-3 flex items-center gap-2 rounded-xl p-3 text-[12.5px] font-bold leading-6">
          <Icon name="phone" size={16} />
          <span className="min-w-0 flex-1">{alert.contact.text}</span>
          {phone && (
            <a href={`tel:${phone}`} className="font-black no-underline" dir="ltr">
              {t('alerts.call')}
            </a>
          )}
        </p>
      )}
      {actions.length > 0 && (
        <div className={clsx('mt-3 flex gap-2', style === 'button' ? 'flex-wrap *:flex-1' : 'flex-col items-start')}>
          {actions.map(({ a, r }, i) => {
            const cls =
              style === 'link'
                ? 'pg2-textlink'
                : clsx('btn text-[12.5px] no-underline', i === 0 ? 'pg2-btn-solid' : 'btn-ghost');
            const body = (
              <>
                {label(r.key, a.label)}
                {style === 'link' && <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={16} />}
              </>
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
                  {body}
                </button>
              );
            }
            if (r.kind === 'tel') {
              return (
                <a key={a.key} href={r.href} className={cls}>
                  {body}
                </a>
              );
            }
            return (
              <Link key={a.key} href={r.href} className={cls}>
                {body}
              </Link>
            );
          })}
        </div>
      )}
      <p role="status" className="mt-2 text-[12px] font-bold text-(--muted) empty:hidden">
        {noted ? t('alerts.addedToNote') : mutation.isError ? t('common.saveError') : ''}
      </p>
    </article>
  );
}

// ── Legend + disclaimer (one calm panel) ────────────────────────
function Legend({ rows, t }: { rows: AlertLegendRow[]; t: T }) {
  const byLevel = new Map(rows.map((r) => [r.level, r]));
  const levels: AlertLevelV2[] = rows.length > 0 ? rows.map((r) => r.level) : [...ALERT_LEVELS_V2];
  return (
    <section className="pg2-note mx-4 rounded-2xl p-4" aria-labelledby="alerts-legend">
      <h2 id="alerts-legend" className="text-[14px] font-black text-(--ink)">
        {t('alerts.legendTitle')}
      </h2>
      <ul className="mt-3 flex flex-col gap-2.5">
        {levels.map((level) => {
          const row = byLevel.get(level);
          return (
            <li key={level} className={clsx('pg2-level-' + level, 'flex items-start gap-2 text-[12.5px] leading-6')}>
              <span className="pg2-chip shrink-0 rounded-full px-2.5 py-0.5 text-[11px]">
                {row?.label ?? t(`alerts.levelsShort.${level}`)}
              </span>
              <span className="text-(--steel)">{row?.text ?? t(`alerts.legend.${level}`)}</span>
            </li>
          );
        })}
      </ul>
      <p className="mt-3 text-[12.5px] leading-6 font-bold text-(--ink)">{t('alerts.disclaimer')}</p>
    </section>
  );
}

// ── Page ───────────────────────────────────────────────────────
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
        <span className="sr-only" role="status">
          {t('common.loading')}
        </span>
        {[20, 180, 140, 160].map((h, k) => (
          <div key={k} className="mx-4 animate-pulse rounded-2xl bg-(--surface-2)" style={{ height: h }} aria-hidden />
        ))}
      </Shell>
    );
  }

  if (query.isError) {
    return (
      <Shell>
        <div className="card mx-4 mt-6 p-5 text-center" role="alert">
          <p className="text-[14px] font-bold text-(--ink)">{t('common.loadError')}</p>
          <button type="button" className="btn btn-primary mt-4" onClick={() => void query.refetch()}>
            {t('common.retry')}
          </button>
        </div>
      </Shell>
    );
  }

  const data = query.data;
  if (!data) {
    return (
      <Shell>
        <div className="card mx-4 mt-6 p-5 text-center">
          <p className="text-[14px] font-bold text-(--ink)">{t('common.notActive')}</p>
          <Link href="/pregnancy/setup" className="btn btn-primary mt-4 no-underline">
            {t('today.setupCta')}
          </Link>
        </div>
      </Shell>
    );
  }

  const groups = groupAlertsByDay(data.alerts);

  return (
    <Shell windowNote={t('alerts.window', { days: data.windowDays })}>
      {groups.length === 0 && (
        <div className="card mx-4 p-5 text-center text-[13px] font-bold text-(--steel)">{t('alerts.empty')}</div>
      )}
      {groups.map((g) => {
        const text = dayLabel(g.day, g.label, t);
        return (
          <section key={g.day || 'unknown'} className="mx-4 flex flex-col gap-3" aria-label={text ?? undefined}>
            {g.alerts.map((a) => (
              <AlertCard key={a.id} alert={a} dateText={text} t={t} />
            ))}
          </section>
        );
      })}
      <Legend rows={data.legend} t={t} />
    </Shell>
  );
}
