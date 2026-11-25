'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { useModuleEnabled } from '@/entities/app-module';
import {
  type DueCard,
  type NextVisit,
  type PregnancyToday,
  usePregnancyToday,
  V2_TERM_WEEKS,
  type WeekTip,
} from '@/entities/pregnancy';
import { isLossCareOpen, LossCareReturn, useLossState } from '@/entities/loss';
import { useDailyMessage } from '@/entities/message';
import { Link, type Locale, useDirection } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import { openSheet } from '@/shared/sheet';
import {
  formatDayMonth,
  formatLongDate,
  formatNumber,
  formatWeekday,
  formatWeekdayDayMonth,
  fromApiDate,
  toApiDate,
  today as todayDate,
} from '@/shared/lib/date';
import {
  Card,
  EmptyState,
  HubHeader,
  Icon,
  type IconName,
  IconCircle,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  type Tone,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';
import { ForToday } from '@/widgets/for-today';
import { HeaderActions } from '@/widgets/notification-bell';
import { PregnancyCareChecklist } from '@/widgets/pregnancy-care-checklist';
import { TodayRemindersCard } from '@/widgets/today-reminders';

import { selectPregnancySmartTip } from '../model/smart-tip';
import { WeekRing } from './WeekRing';
import '@/shared/ui/styles/preg.css';
import '@/shared/ui/styles/pgn.css';

type T = ReturnType<typeof useTranslations<'pregnancyV2'>>;

function Shell({ header, children }: { header?: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="view preg-page pgn-screen">
      <SkyLayer />
      <div className="scroll">
        {header}
        <div className="pgn-body is-hub">{children}</div>
      </div>
      <BottomNav />
    </div>
  );
}

function Header({ t }: { t: T }) {
  const locale = useLocale() as Locale;
  return (
    <HubHeader
      className="pgn-hub"
      date={formatWeekdayDayMonth(todayDate(), locale)}
      greeting={t('common.title')}
      actions={<HeaderActions className="nb-hbtn is-soft" />}
    />
  );
}

// ── Ring + status pills ────────────────────────────────────────
function RingHero({ data, t }: { data: PregnancyToday; t: T }) {
  const locale = useLocale() as Locale;
  const week = data.progress.week;
  const days = data.due?.daysLeft ?? null;
  const trimester = data.trimester as 1 | 2 | 3 | null;
  const level = data.confidence.level;
  const levelText = level ? t(`common.confidence.levels.${level}`) : null;
  const weekText = t('common.weekOf', { week, total: V2_TERM_WEEKS });
  return (
    <>
      <WeekRing
        week={week}
        label={days != null ? t('today.ringLabel', { week, total: V2_TERM_WEEKS, days }) : weekText}
      >
        <span className="pgn-ring-week">{weekText}</span>
        {days != null ? (
          <>
            <span className="pgn-ring-num">{formatNumber(days, locale)}</span>
            <span className="pgn-ring-cap">{t('today.daysToBirth')}</span>
          </>
        ) : (
          <span className="pgn-ring-num">{formatNumber(week, locale)}</span>
        )}
        <span className="pgn-ring-sub">{t('today.percentShort', { percent: data.progress.percent })}</span>
      </WeekRing>
      {(trimester || levelText) && (
        <div className="pgn-pills">
          {trimester && <span className="pgn-opill nb-tone-bloom">{t(`common.trimester.${trimester}`)}</span>}
          {levelText && (
            <span className="pgn-opill nb-tone-warm">
              {data.uncertaintyDays != null
                ? t('common.confidence.withRange', { level: levelText, days: data.uncertaintyDays })
                : t('common.confidence.label', { level: levelText })}
            </span>
          )}
        </div>
      )}
    </>
  );
}

// ── Due date ───────────────────────────────────────────────────
function DueDateCard({ due, t }: { due: DueCard; t: T }) {
  const locale = useLocale() as Locale;
  const range = due.range
    ? t('today.usualRange', {
        from: formatDayMonth(fromApiDate(due.range.from), locale),
        to: formatDayMonth(fromApiDate(due.range.to), locale),
      })
    : null;
  return (
    <Card as="section" className="pgn-sect" aria-labelledby="pgn-due">
      <div className="pgn-sect-head">
        <h2 id="pgn-due" className="pgn-sect-title">
          {t('common.dueDate')}
        </h2>
        <Link href="/pregnancy/setup" className="pgn-icon-link" aria-label={t('setup.editBasis')} {...pilot('preg.setup-edit')}>
          <Icon name="pen" size={18} strokeWidth={1.8} />
        </Link>
      </div>
      <span className="pgn-display">{due.dateLabel ?? formatLongDate(fromApiDate(due.date), locale)}</span>
      <p className="pgn-caption">
        {range ? `${range}${t('common.separator')}${t('common.estimateNote')}` : t('common.estimateNote')}
      </p>
    </Card>
  );
}

// ── Quick tiles (4 across) ─────────────────────────────────────
function Tile({
  href,
  icon,
  tone,
  label,
  badge,
  ariaLabel,
  onClick,
  pilotId,
}: {
  /** Autopilot name of the tile. */
  pilotId: string;
  /** A tile either links somewhere or (with `onClick`) opens a sheet in place. */
  href?: string;
  onClick?: () => void;
  icon: IconName;
  tone: Tone;
  label: string;
  badge?: string;
  ariaLabel?: string;
}) {
  const inner = (
    <>
      <IconCircle icon={icon} tone={tone} size="md" />
      <b className="pgn-tile-label">{label}</b>
      {badge && (
        <span className="pgn-tile-badge" aria-hidden>
          {badge}
        </span>
      )}
    </>
  );
  if (onClick || !href) {
    return (
      <button type="button" onClick={onClick} aria-label={ariaLabel} className="pgn-tile" {...pilot(pilotId)}>
        {inner}
      </button>
    );
  }
  return (
    <Link href={href} aria-label={ariaLabel} className="pgn-tile" {...pilot(pilotId)}>
      {inner}
    </Link>
  );
}

function QuickTiles({ data, t }: { data: PregnancyToday; t: T }) {
  const locale = useLocale() as Locale;
  return (
    <nav className="pgn-tiles" aria-label={t('common.title')}>
      <Tile pilotId="preg.open:log" href="/pregnancy/log" icon="plus" tone="bloom" label={t('today.actions.logShort')} />
      <Tile pilotId="preg.open:weekly" href="/pregnancy/log?tab=weekly" icon="check" tone="data" label={t('today.actions.checkup')} />
      <Tile pilotId="preg.open:weeks" href={`/pregnancy/weeks/${data.progress.week}`} icon="calendar" tone="brand" label={t('today.actions.weeks')} />
      <Tile
        pilotId="preg.open:alerts"
        onClick={() => openSheet('notifications')}
        icon="bell"
        tone="danger"
        label={t('today.actions.alerts')}
        badge={data.unreadAlerts > 0 ? formatNumber(data.unreadAlerts, locale) : undefined}
        ariaLabel={t('today.newAlerts', { count: data.unreadAlerts })}
      />
    </nav>
  );
}

// ── Next visit ─────────────────────────────────────────────────
function NextVisitCard({ visit, t }: { visit: NextVisit; t: T }) {
  const locale = useLocale() as Locale;
  const date = visit.date ? fromApiDate(visit.date) : null;
  const weekday = date ? formatWeekday(date, locale) : null;
  const time = visit.time ? formatNumber(visit.time, locale) : null;
  const meta =
    weekday && time && visit.week != null
      ? t('today.visitMeta', { weekday, time, week: visit.week })
      : [weekday ?? visit.dateLabel, time].filter(Boolean).join(t('common.separator'));
  return (
    <Card as="section" className="pgn-sect" aria-labelledby="pgn-visit">
      <div className="pgn-sect-head">
        <h2 id="pgn-visit" className="pgn-sect-title">
          {visit.daysUntil != null ? t('today.nextVisit', { days: visit.daysUntil }) : visit.title}
        </h2>
        <Link href="/pregnancy/calendar" className="pgn-sect-link" {...pilot('preg.open:calendar')}>
          {t('today.calendarLink')}
        </Link>
      </div>
      <Link href="/pregnancy/calendar" className="pgn-row is-link" {...pilot('preg.open:next-visit')}>
        <IconCircle icon="calendar" tone="brand" size="sm" />
        <span className="pgn-row-text">
          <b className="pgn-row-title">{visit.title}</b>
          {meta && <span className="pgn-row-desc">{meta}</span>}
        </span>
      </Link>
    </Card>
  );
}

// ── Smart tip ──────────────────────────────────────────────────
/** From week 20 the home offers «زایمان کردم» → postpartum activation (B-N5-04; closes the pregnancy as delivered). */
const DELIVERED_FROM_WEEK = 20;

function DeliveredCard({ t }: { t: T }) {
  return (
    <Card as="section" className="pgn-delivered" aria-labelledby="pgn-delivered-title">
      <IconCircle icon="heart" tone="bloom" size="lg" />
      <div className="pgn-delivered-text">
        <h2 id="pgn-delivered-title" className="pgn-delivered-title">
          {t('today.delivered.title')}
        </h2>
        <p className="pgn-delivered-body">{t('today.delivered.body')}</p>
      </div>
      <Link href="/postpartum/setup" className="nb-btn is-outline is-block" {...pilot('preg.delivered')}>
        {t('today.delivered.cta')}
      </Link>
    </Card>
  );
}

/**
 * «نکتهٔ هوشمند» of the pregnancy home (the cycle home's smart tip, for pregnancy): the week's tip, replaced for
 * the day by a logged symptom, plus the engine's general lines. See `model/smart-tip.ts`.
 */
function SmartTipCard({ week: weekTip, weekNumber, t }: { week: WeekTip | null; weekNumber: number; t: T }) {
  const isRtl = useDirection() === 'rtl';
  const { data: daily } = useDailyMessage();
  const tip = selectPregnancySmartTip(weekTip, daily);
  if (!tip) return null;
  const tipWeek = weekTip?.week ?? weekNumber;
  const href = weekTip?.link ?? `/pregnancy/weeks/${tipWeek}`;
  const more = (
    <>
      {t('today.readMore', { week: tipWeek })}
      <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={15} strokeWidth={2.2} />
    </>
  );
  return (
    <Card as="section" className="pgn-sect pgn-tip" aria-labelledby="pgn-tip" data-tip-kind={tip.kind}>
      <div className="pgn-sect-head">
        <span className="pgn-eyebrow">{tip.kind === 'symptom' ? t('today.tipEyebrowToday') : t('today.tipEyebrow')}</span>
        {tip.kind === 'week' && weekTip?.readMinutes != null && (
          <span className="pgn-meta">{t('today.readMinutes', { minutes: weekTip.readMinutes })}</span>
        )}
      </div>
      <h2 id="pgn-tip" className={tip.title ? 'pgn-sect-title' : 'sr-only'}>
        {tip.title || t('today.tipTitle')}
      </h2>
      <p className="pgn-body-text">{tip.body}</p>
      {tip.action && (
        <p className="pgn-tip-action">
          <Icon name="sparkle" size={16} fill="currentColor" strokeWidth={0} aria-hidden />
          <span>{tip.action}</span>
        </p>
      )}
      {tip.extras.length > 0 && (
        <ul className="pgn-tip-extras">
          {tip.extras.map((line) => (
            <li key={line}>{line}</li>
          ))}
        </ul>
      )}
      {/^https?:\/\//.test(href) ? (
        <a href={href} target="_blank" rel="noopener noreferrer" className="pgn-link">
          {more}
        </a>
      ) : (
        <Link href={href} className="pgn-link">
          {more}
        </Link>
      )}
    </Card>
  );
}

// ── States ─────────────────────────────────────────────────────
function Loading({ t }: { t: T }) {
  return (
    <Shell>
      <SkeletonGroup label={t('common.loading')} className="pgn-skel">
        <Skeleton width="medium" />
        <Skeleton shape="circle" className="pgn-skel-ring" />
        <Skeleton shape="card" />
        <Skeleton shape="block" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    </Shell>
  );
}

// ── Main export ────────────────────────────────────────────────
/** «بارداری» today — `/pregnancy` (PregFull_Main + v13_Preg_Home reminders block). */
export function PregnancyPage() {
  const t = useTranslations('pregnancyV2');
  // «زایمان کردم» leads into postpartum: no card while the admin switched that module off.
  const postpartumOn = useModuleEnabled('postpartum');
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  const query = usePregnancyToday();
  // B-N5-10 (canvas QA CB-LOSS-03): right after a loss, no «راه‌اندازی حالت بارداری» CTA here.
  const lossState = useLossState();
  const recentLoss = isLossCareOpen(lossState.data?.loss, toApiDate(todayDate()));

  if (!mounted || query.isLoading || (!query.data && lossState.isLoading)) return <Loading t={t} />;

  if (query.isError) {
    return (
      <Shell header={<Header t={t} />}>
        <Card className="pgn-state" role="alert">
          <span className="pgn-state-disc" aria-hidden>
            <Icon name="warning" size={24} />
          </span>
          <p className="pgn-state-text">{t('common.loadError')}</p>
          <SecondaryButton icon="refresh" block={false} onClick={() => void query.refetch()} {...pilot('preg.retry')}>
            {t('common.retry')}
          </SecondaryButton>
        </Card>
      </Shell>
    );
  }

  const data = query.data;
  if (!data && recentLoss) {
    return (
      <Shell header={<Header t={t} />}>
        <Card className="pgn-state">
          <span className="pgn-state-disc is-calm" aria-hidden>
            <Icon name="heart" size={24} />
          </span>
          <p className="pgn-state-text">{t('common.notActiveAfterLoss')}</p>
        </Card>
        <LossCareReturn variant="row" />
      </Shell>
    );
  }
  if (!data) {
    return (
      <Shell header={<Header t={t} />}>
        <Card>
          <EmptyState
            icon="heart"
            title={t('common.notActive')}
            action={
              <Link href="/pregnancy/setup" className="nb-btn is-primary is-block" {...pilot('preg.setup')}>
                {t('today.setupCta')}
              </Link>
            }
          />
        </Card>
      </Shell>
    );
  }

  return (
    <Shell header={<Header t={t} />}>
      <RingHero data={data} t={t} />
      {data.due && <DueDateCard due={data.due} t={t} />}
      <QuickTiles data={data} t={t} />
      {/* The «ویزیت بعدی» card below already shows this appointment. */}
      {/* CB-NAV-03b: «برای امروز» = reminders + to-do row, like the other homes. */}
      <ForToday reminders={<TodayRemindersCard hideAppointmentId={data.nextVisit?.appointmentId ?? null} />} />
      {data.nextVisit && <NextVisitCard visit={data.nextVisit} t={t} />}
      <SmartTipCard week={data.tip} weekNumber={data.progress.week} t={t} />
      <PregnancyCareChecklist week={data.progress.week} tasks={data.tasks} />
      {postpartumOn && data.progress.week >= DELIVERED_FROM_WEEK && <DeliveredCard t={t} />}
    </Shell>
  );
}
