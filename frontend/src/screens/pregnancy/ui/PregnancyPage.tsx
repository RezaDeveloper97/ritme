'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import {
  type DueCard,
  type NextVisit,
  type PregnancyProgressV2,
  type PregnancyToday,
  trimesterFills,
  usePregnancyToday,
  V2_TERM_WEEKS,
  type WeekTip,
} from '@/entities/pregnancy';
import { Link, type Locale, useDirection } from '@/shared/i18n';
import {
  formatDayMonth,
  formatLongDate,
  formatNumber,
  formatWeekdayDayMonth,
  fromApiDate,
  monthName,
  today as todayDate,
  toApiDate,
  toParts,
  weekdayLabels,
  weekOf,
} from '@/shared/lib/date';
import { Icon, type IconName } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';
import { PregnancyCareChecklist } from '@/widgets/pregnancy-care-checklist';
import { PregnancyWeekCarousel } from '@/widgets/pregnancy-week-carousel';
import { TodayRemindersCard } from '@/widgets/today-reminders';

type T = ReturnType<typeof useTranslations<'pregnancyV2'>>;

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <div className="view preg-page">
      {/* Children never shrink: an overflow-hidden card would otherwise collapse to 0 in this column. */}
      <div className="scroll flex flex-col gap-3 pb-6 *:shrink-0">{children}</div>
      <BottomNav />
    </div>
  );
}

const range = (value: { from: string; to: string }, locale: Locale) => ({
  from: formatDayMonth(fromApiDate(value.from), locale),
  to: formatDayMonth(fromApiDate(value.to), locale),
});

// ── Date strip ─────────────────────────────────────────────────
function DateStrip({ locale }: { locale: Locale }) {
  const now = todayDate();
  const days = weekOf(now, locale);
  const labels = weekdayLabels(locale);
  const todayKey = toApiDate(now);
  return (
    <section className="px-4 pt-3">
      <h1 className="text-[15px] font-black text-(--ink)">{formatWeekdayDayMonth(now, locale)}</h1>
      <div className="mt-2 grid grid-cols-7 gap-1.5">
        {days.map((d, k) => {
          const key = toApiDate(d);
          const isToday = key === todayKey;
          const isFuture = key > todayKey;
          const body = (
            <>
              <span className="text-[10.5px] font-bold">{labels[k]}</span>
              <span className="text-[14px] font-black">{formatNumber(toParts(d, locale).day, locale)}</span>
            </>
          );
          const cls = clsx(
            'flex flex-col items-center gap-0.5 rounded-xl py-1.5 no-underline',
            isToday ? 'bg-(--brand-fill) text-(--on-accent)' : 'bg-(--surface) text-(--ink)',
            isFuture && 'text-(--ink-3)',
          );
          return isFuture ? (
            <div key={key} className={cls}>
              {body}
            </div>
          ) : (
            <Link key={key} href={`/pregnancy/log?date=${key}`} className={cls} aria-current={isToday ? 'date' : undefined}>
              {body}
            </Link>
          );
        })}
      </div>
    </section>
  );
}

// ── Due-date card ──────────────────────────────────────────────
function DueDateCard({ due, t, locale }: { due: DueCard; t: T; locale: Locale }) {
  const r = due.range ? range(due.range, locale) : null;
  return (
    <section className="card mx-4 flex items-center gap-3 p-4">
      <span className="pg2-tile pg2-tone-pink flex size-11 shrink-0 items-center justify-center rounded-xl">
        <Icon name="calendar" size={20} />
      </span>
      <div className="min-w-0">
        <div className="text-[12px] font-bold text-(--muted)">{t('common.dueDate')}</div>
        <div className="text-[16px] font-black text-(--ink)">
          {due.dateLabel ?? formatLongDate(fromApiDate(due.date), locale)}
        </div>
        <div className="text-[12px] font-bold text-(--steel)">
          {due.daysLeft != null && t('today.daysToDue', { days: due.daysLeft })}
          {due.daysLeft != null && r && ' · '}
          {r && t('today.usualRange', r)}
        </div>
      </div>
    </section>
  );
}

// ── 40-week progress with trimester segments ──────────────────
function ProgressCard({ progress, t, locale }: { progress: PregnancyProgressV2; t: T; locale: Locale }) {
  const fills = trimesterFills(progress);
  const byTrimester = new Map(fills.map((f) => [f.trimester, f]));
  return (
    <section className="card mx-4 p-4">
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-[14px] font-black text-(--ink)">
          {t('common.weekOf', { week: progress.week, total: V2_TERM_WEEKS })}
        </span>
        <span className="text-[12px] font-bold text-(--steel)">{t('today.percentDone', { percent: progress.percent })}</span>
      </div>
      <div
        className="mt-3 flex gap-1"
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={progress.percent}
      >
        {fills.map((f) => (
          <div key={f.trimester} className="h-2 flex-1 overflow-hidden rounded-full bg-(--brand-line-soft)">
            <div className="h-full rounded-full bg-(--brand-fill)" style={{ width: `${f.fill}%` }} />
          </div>
        ))}
      </div>
      <div className="mt-1.5 flex gap-1 text-[10.5px] font-bold text-(--muted)">
        {progress.trimesters.map((s) => {
          const name = t(`today.trimesterShort.${s.trimester}`);
          const date = s.startLabel ?? (s.startDate ? formatDayMonth(fromApiDate(s.startDate), locale) : null);
          return (
            <span key={s.trimester} className="flex-1 truncate">
              {date && byTrimester.get(s.trimester)?.started === false ? t('today.trimesterFrom', { name, date }) : name}
            </span>
          );
        })}
      </div>
    </section>
  );
}

// ── Quick actions ──────────────────────────────────────────────
function Action({ href, icon, label, badge, ariaLabel }: { href: string; icon: IconName; label: string; badge?: number; ariaLabel?: string }) {
  return (
    <Link
      href={href}
      aria-label={ariaLabel}
      className="card relative flex flex-col items-center gap-1.5 px-1 py-3 text-center no-underline"
    >
      <span className="pg2-tile pg2-tone-brand flex size-10 items-center justify-center rounded-xl">
        <Icon name={icon} size={20} />
      </span>
      <span className="text-[11.5px] font-bold text-(--ink)">{label}</span>
      {badge != null && badge > 0 && (
        <span className="absolute end-2 top-2 min-w-5 rounded-full bg-(--care-rose) px-1.5 text-[10.5px] font-black text-(--on-accent)">
          {badge}
        </span>
      )}
    </Link>
  );
}

function QuickActions({ data, t }: { data: PregnancyToday; t: T }) {
  return (
    <nav className="mx-4 grid grid-cols-4 gap-2">
      <Action href="/pregnancy/log" icon="plus" label={t('today.actions.log')} />
      <Action href="/pregnancy/log?tab=weekly" icon="stetho" label={t('today.actions.checkup')} />
      <Action href={`/pregnancy/weeks/${data.progress.week}`} icon="bookOpen" label={t('today.actions.weeks')} />
      <Action
        href="/pregnancy/alerts"
        icon="bell"
        label={t('today.actions.alerts')}
        badge={data.unreadAlerts}
        ariaLabel={t('today.newAlerts', { count: data.unreadAlerts })}
      />
    </nav>
  );
}

// ── Next visit ─────────────────────────────────────────────────
function NextVisitCard({ visit, t, locale }: { visit: NextVisit; t: T; locale: Locale }) {
  const date = visit.date ? fromApiDate(visit.date) : null;
  const parts = date ? toParts(date, locale) : null;
  const weekday = date
    ? new Intl.DateTimeFormat(locale === 'fa' ? 'fa-IR' : 'en-US', { weekday: 'long' }).format(date)
    : null;
  const time = visit.time ? visit.time.replace(/\d/g, (d) => formatNumber(Number(d), locale)) : null;
  const meta =
    weekday && time && visit.week != null
      ? t('today.visitMeta', { weekday, time, week: visit.week })
      : [weekday, time].filter(Boolean).join(' · ');
  return (
    <section className="card mx-4 flex items-center gap-3 p-4">
      {parts && (
        <div className="pg2-tile pg2-tone-teal flex size-13 shrink-0 flex-col items-center justify-center rounded-xl">
          <span className="text-[10.5px] font-bold">{monthName(parts.month, locale)}</span>
          <span className="text-[17px] font-black">{formatNumber(parts.day, locale)}</span>
        </div>
      )}
      <div className="min-w-0">
        {visit.daysUntil != null && (
          <div className="text-[11.5px] font-bold text-(--muted)">{t('today.nextVisit', { days: visit.daysUntil })}</div>
        )}
        <div className="text-[14.5px] font-black text-(--ink)">{visit.title}</div>
        {meta && <div className="text-[12px] font-bold text-(--steel)">{meta}</div>}
      </div>
    </section>
  );
}

// ── Smart tip ──────────────────────────────────────────────────
function TipCard({ tip, week, t }: { tip: WeekTip; week: number; t: T }) {
  const isRtl = useDirection() === 'rtl';
  const tipWeek = tip.week ?? week;
  const href = tip.link ?? `/pregnancy/weeks/${tipWeek}`;
  const external = /^https?:\/\//.test(href);
  const more = (
    <>
      {t('today.readMore', { week: tipWeek })}
      <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={16} />
    </>
  );
  const linkCls = 'mt-3 flex items-center gap-1 text-[12.5px] font-black text-(--brand) no-underline';
  return (
    <section className="card mx-4 p-4">
      <div className="flex items-center gap-2 text-[11.5px] font-bold text-(--muted)">
        <Icon name="sparkle" size={14} />
        <span>{t('today.tipEyebrow')}</span>
        {tip.readMinutes != null && <span className="ms-auto">{t('today.readMinutes', { minutes: tip.readMinutes })}</span>}
      </div>
      <h2 className="mt-2 text-[15px] font-black text-(--ink)">{tip.title}</h2>
      <p className="mt-1.5 text-[13px] leading-6 text-(--steel)">{tip.body}</p>
      {external ? (
        <a href={href} target="_blank" rel="noopener noreferrer" className={linkCls}>
          {more}
        </a>
      ) : (
        <Link href={href} className={linkCls}>
          {more}
        </Link>
      )}
    </section>
  );
}

// ── States ─────────────────────────────────────────────────────
function Skeleton() {
  const t = useTranslations('pregnancyV2');
  return (
    <Shell>
      <span className="sr-only" role="status">
        {t('common.loading')}
      </span>
      {[56, 300, 84, 70, 90, 110].map((h, k) => (
        <div key={k} className="mx-4 animate-pulse rounded-2xl bg-(--surface-2)" style={{ height: h }} aria-hidden />
      ))}
    </Shell>
  );
}

// ── Main export ────────────────────────────────────────────────
export function PregnancyPage() {
  const t = useTranslations('pregnancyV2');
  const locale = useLocale() as Locale;
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  const query = usePregnancyToday();

  if (!mounted || query.isLoading) return <Skeleton />;

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

  const disclaimerRange = data.due?.range ? range(data.due.range, locale) : null;

  return (
    <Shell>
      <DateStrip locale={locale} />
      {data.carousel.length > 0 && (
        <PregnancyWeekCarousel slides={data.carousel} confidence={data.confidence} uncertaintyDays={data.uncertaintyDays} />
      )}
      {data.due && <DueDateCard due={data.due} t={t} locale={locale} />}
      <ProgressCard progress={data.progress} t={t} locale={locale} />
      <QuickActions data={data} t={t} />
      {data.nextVisit && <NextVisitCard visit={data.nextVisit} t={t} locale={locale} />}
      <TodayRemindersCard />
      {data.tip && <TipCard tip={data.tip} week={data.progress.week} t={t} />}
      <PregnancyCareChecklist week={data.progress.week} tasks={data.tasks} />
      <aside className="pg2-warn mx-4 flex gap-2 rounded-2xl p-3 text-[12px] leading-6">
        <Icon name="info" size={16} className="pg2-warn-icon" />
        <p>
          <strong>{t('today.disclaimerTitle')}</strong>{' '}
          {disclaimerRange ? t('today.disclaimerBody', disclaimerRange) : t('common.estimateNote')}
        </p>
      </aside>
    </Shell>
  );
}
