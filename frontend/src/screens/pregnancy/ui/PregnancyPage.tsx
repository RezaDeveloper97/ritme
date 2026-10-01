'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import {
  type DueCard,
  type NextVisit,
  type PregnancyToday,
  usePregnancyToday,
  V2_TERM_WEEKS,
  type WeekTip,
} from '@/entities/pregnancy';
import { Link, type Locale, useDirection, useRouter } from '@/shared/i18n';
import {
  formatDayMonth,
  formatLongDate,
  formatNumber,
  formatWeekday,
  formatWeekdayDayMonth,
  fromApiDate,
  today as todayDate,
} from '@/shared/lib/date';
import {
  Card,
  EmptyState,
  HeaderButton,
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
import { PregnancyCareChecklist } from '@/widgets/pregnancy-care-checklist';
import { TodayRemindersCard } from '@/widgets/today-reminders';

import { WeekRing } from './WeekRing';

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

function Header({ unread, t }: { unread: number; t: T }) {
  const locale = useLocale() as Locale;
  const router = useRouter();
  const tSearch = useTranslations('search');
  return (
    <HubHeader
      className="pgn-hub"
      date={formatWeekdayDayMonth(todayDate(), locale)}
      greeting={t('common.title')}
      actions={
        <>
          {/* CB-NAV-02: global search (nbd_Nav_Today header). */}
          <HeaderButton variant="soft" icon="search" label={tSearch('open')} onClick={() => router.push('/search')} />
          <HeaderButton
            variant="soft"
            icon="bell"
            badge={unread > 0}
            label={t('today.newAlerts', { count: unread })}
            onClick={() => router.push('/pregnancy/alerts')}
          />
        </>
      }
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
        <Link href="/pregnancy/setup" className="pgn-icon-link" aria-label={t('setup.editBasis')}>
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
}: {
  href: string;
  icon: IconName;
  tone: Tone;
  label: string;
  badge?: string;
  ariaLabel?: string;
}) {
  return (
    <Link href={href} aria-label={ariaLabel} className="pgn-tile">
      <IconCircle icon={icon} tone={tone} size="md" />
      <b className="pgn-tile-label">{label}</b>
      {badge && (
        <span className="pgn-tile-badge" aria-hidden>
          {badge}
        </span>
      )}
    </Link>
  );
}

function QuickTiles({ data, t }: { data: PregnancyToday; t: T }) {
  const locale = useLocale() as Locale;
  return (
    <nav className="pgn-tiles" aria-label={t('common.title')}>
      <Tile href="/pregnancy/log" icon="plus" tone="bloom" label={t('today.actions.logShort')} />
      <Tile href="/pregnancy/log?tab=weekly" icon="check" tone="data" label={t('today.actions.checkup')} />
      <Tile href={`/pregnancy/weeks/${data.progress.week}`} icon="calendar" tone="brand" label={t('today.actions.weeks')} />
      <Tile
        href="/pregnancy/alerts"
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
        <Link href="/pregnancy/calendar" className="pgn-sect-link">
          {t('today.calendarLink')}
        </Link>
      </div>
      <Link href="/pregnancy/calendar" className="pgn-row is-link">
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
function TipCard({ tip, week, t }: { tip: WeekTip; week: number; t: T }) {
  const isRtl = useDirection() === 'rtl';
  const tipWeek = tip.week ?? week;
  const href = tip.link ?? `/pregnancy/weeks/${tipWeek}`;
  const more = (
    <>
      {t('today.readMore', { week: tipWeek })}
      <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={15} strokeWidth={2.2} />
    </>
  );
  return (
    <Card as="section" className="pgn-sect pgn-tip" aria-labelledby="pgn-tip">
      <div className="pgn-sect-head">
        <span className="pgn-eyebrow">{t('today.tipEyebrow')}</span>
        {tip.readMinutes != null && (
          <span className="pgn-meta">{t('today.readMinutes', { minutes: tip.readMinutes })}</span>
        )}
      </div>
      <h2 id="pgn-tip" className="pgn-sect-title">
        {tip.title}
      </h2>
      <p className="pgn-body-text">{tip.body}</p>
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
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  const query = usePregnancyToday();

  if (!mounted || query.isLoading) return <Loading t={t} />;

  if (query.isError) {
    return (
      <Shell header={<Header unread={0} t={t} />}>
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
      <Shell header={<Header unread={0} t={t} />}>
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

  return (
    <Shell header={<Header unread={data.unreadAlerts} t={t} />}>
      <RingHero data={data} t={t} />
      {data.due && <DueDateCard due={data.due} t={t} />}
      <QuickTiles data={data} t={t} />
      {/* The «ویزیت بعدی» card below already shows this appointment. */}
      <TodayRemindersCard hideAppointmentId={data.nextVisit?.appointmentId ?? null} />
      {data.nextVisit && <NextVisitCard visit={data.nextVisit} t={t} />}
      {data.tip && <TipCard tip={data.tip} week={data.progress.week} t={t} />}
      <PregnancyCareChecklist week={data.progress.week} tasks={data.tasks} />
    </Shell>
  );
}
