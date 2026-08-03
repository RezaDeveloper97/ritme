'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useState, type ReactNode } from 'react';

import { useCycleArticles } from '@/entities/article';
import {
  CycleValuesCard,
  cycleDayMarker,
  cycleMarkerBg,
  cycleMarkerStyle,
  cycleScheduleFor,
  daysUntilNextPeriod,
  deriveCycleSchedule,
  deriveCyclePredictions,
  deriveDayHighlights,
  fertilityBadgeStyle,
  markerIntensityByDate,
  useCycleForDate,
  useCycleMonth,
  useCycleStatus,
  useCycleToday,
  useRecalculateCycle,
  type CycleCalculation,
  type CycleDailyTip,
  type CycleDayHighlight,
  type CycleDayMarker,
  type CyclePredictions,
  type CycleSchedule,
  type MarkerIntensity,
  type FertilityBadgeStyle,
} from '@/entities/cycle';
import { useDailyMessage, type DailyMessage } from '@/entities/message';
import { useUserProfile } from '@/entities/user';
import { QuickEditSheet } from '@/features/edit-profile';
import { usePeriodHistory } from '@/features/log-period';
import { Link, useRouter } from '@/shared/i18n';
import type { Locale } from '@/shared/i18n';
import {
  addDays,
  diffInDays,
  formatDayMonth,
  formatMonthLabel,
  fromApiDate,
  monthMatrix,
  toApiDate,
  toParts,
  today,
  todayParts,
  weekdayKeys,
  type MonthCell,
} from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { DropSolid, Icon, type IconName } from '@/shared/ui';
import { BannerSlideshow } from '@/widgets/banner-slideshow';
import { BottomNav } from '@/widgets/bottom-nav';
// import { DayTasks } from '@/widgets/day-tasks';
import { TodayChallengeCard } from '@/widgets/today-challenge';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);
const localizeNum = (n: string | number, loc: Locale) => (loc === 'fa' ? faNum(n) : String(n));

type T = ReturnType<typeof useTranslations>;

// ── App header (Figma: bare icons — stars on the start side, bell on the end) ──
// The start-side button carries the cycle screen's "recalculate" action, so the
// engine can be re-run from home without visiting /cycle.
function HomeHeader({
  tagline, onRecalculate, recalculating, recalculateLabel,
}: {
  tagline: string;
  onRecalculate: () => void;
  recalculating: boolean;
  recalculateLabel: string;
}) {
  return (
    <div className="home-hdr">
      {/* The dimmed look while recalculating comes from `.iconbtn:disabled`,
          so the disabled attribute is the single source of truth for it. */}
      <button
        className="iconbtn home-recalc"
        onClick={onRecalculate}
        disabled={recalculating}
        aria-label={recalculateLabel}
      >
        <Icon name={recalculating ? 'loader' : 'refresh'} size={22} />
      </button>
      <div className="home-brand">
        <div className="home-logo">ریـــــتمی</div>
        <div className="home-tagline">{tagline}</div>
      </div>
      <button className="iconbtn home-bell">
        <Icon name="bell" size={21} />
      </button>
    </div>
  );
}

// ── Week strip ─────────────────────────────────────────────────
// The month grid comes from the centralized date layer (§7) — no
// hardcoded month/day. Day names come from i18n.

// Fallback bleeding length when the profile hasn't recorded one (matches the
// backend default) — used to tell a logged period day from a predicted one.
const DEFAULT_PERIOD_DAYS = 5;

/** Gregorian year/month a date falls in, parsed from its API serialization (§7). */
function gregYearMonth(date: Date): { year: number; month: number } {
  const [year, month] = toApiDate(date).split('-');
  return { year: Number(year), month: Number(month) };
}

/** How a day cell is painted: its cycle marker, and whether a period day is real. */
interface DayMark {
  marker: CycleDayMarker | null;
  /** Tint depth for the marker — graded by the day's conception probability. */
  intensity: MarkerIntensity;
  /** Period marker the user actually logged (vs. one the engine predicted). */
  isLogged: boolean;
}

const NO_MARK: DayMark = { marker: null, intensity: 'medium', isLogged: false };

/**
 * Cycle markers for the days of the shown month, from the same
 * `cycle/month` cache the calendar screen reads — so editing a period there
 * (which invalidates `cycleKeys.all`) repaints this mini calendar too.
 * A shown month can span two Gregorian months (Jalali always does), so both
 * are fetched.
 */
function useMonthMarks(cells: MonthCell[]): (date: Date) => DayMark {
  const gA = gregYearMonth(cells[0]?.date ?? today());
  const gB = gregYearMonth(cells[cells.length - 1]?.date ?? today());
  const monthA = useCycleMonth(gA.year, gA.month);
  const monthB = useCycleMonth(gB.year, gB.month);
  const historyQuery = usePeriodHistory();
  const profileQuery = useUserProfile();
  const periodDuration = profileQuery.data?.health?.periodDuration ?? DEFAULT_PERIOD_DAYS;
  const history = historyQuery.data;

  const calcMap = useMemo(() => {
    const map = new Map<string, CycleCalculation>();
    for (const c of monthA.data?.calculations ?? []) map.set(c.calculationDate, c);
    for (const c of monthB.data?.calculations ?? []) map.set(c.calculationDate, c);
    return map;
  }, [monthA.data, monthB.data]);

  // Days covered by a real, user-entered period. An open period covers its start
  // plus the profile's usual bleeding length (mirrors the calendar screen).
  const loggedDays = useMemo(() => {
    const set = new Set<string>();
    for (const p of history ?? []) {
      const start = fromApiDate(p.period_start_date);
      const end = p.period_end_date
        ? fromApiDate(p.period_end_date)
        : addDays(start, periodDuration - 1);
      for (let i = 0; i <= Math.max(0, diffInDays(end, start)); i += 1) {
        set.add(toApiDate(addDays(start, i)));
      }
    }
    return set;
  }, [history, periodDuration]);

  // Tint depth per day, graded by conception probability within each marker
  // group — same grading the calendar screen applies.
  const intensityMap = useMemo(() => markerIntensityByDate(calcMap.values()), [calcMap]);

  return (date: Date) => {
    const iso = toApiDate(date);
    const calc = calcMap.get(iso);
    const marker = calc ? cycleDayMarker(calc) : null;
    if (!marker) return NO_MARK;
    return { marker, intensity: intensityMap.get(iso) ?? 'medium', isLogged: loggedDays.has(iso) };
  };
}

// Figma «TodayCalender»: white card (r12). Closed → weekday names + the
// current week only. Open → the whole month grid, rows in calendar order.
// Tapping a day selects it so the connected info card below shows that day.
function WeekRow({
  days, todayDay, selectedDay, loc, onSelect, markOf,
}: {
  days: (MonthCell | null)[];
  todayDay: number;
  selectedDay: number | null;
  loc: Locale;
  onSelect: (cell: MonthCell) => void;
  markOf: (date: Date) => DayMark;
}) {
  return (
    <div className="home-week-row">
      {days.map((cell, i) => {
        const isToday = cell?.day === todayDay;
        const isSelected = cell != null && cell.day === selectedDay && !isToday;
        const { marker, intensity, isLogged } = cell ? markOf(cell.date) : NO_MARK;
        const mk = marker ? cycleMarkerStyle[marker] : null;
        const markerBg = marker ? cycleMarkerBg[marker][intensity] : null;
        // A period marker the user hasn't logged is a prediction → hollow ring
        // instead of a solid fill, exactly as on the calendar screen (§12).
        const isPredicted = marker === 'period' && !isLogged;
        // Today stays the prominent filled circle, but takes the phase's color so
        // its marker isn't lost under the brand pink.
        const todayFill = mk && !isPredicted ? mk.color : 'var(--brand)';
        return (
          <div key={i} className="home-day-slot">
            <button
              type="button"
              className={clsx('home-day', isSelected && 'is-selected')}
              disabled={!cell}
              onClick={cell ? () => onSelect(cell) : undefined}
              aria-pressed={isToday || isSelected}
              // Data-driven only: which marker colour this specific day carries.
              // The geometry lives in `.home-day` (CLAUDE.md §10).
              style={
                isToday
                  ? { background: todayFill, color: 'var(--on-accent)', fontWeight: 700, boxShadow: '0 8px 16px -6px rgba(123,97,255,.6)' }
                  : isSelected
                    ? { background: markerBg ?? 'var(--surface-2)', color: mk?.color ?? 'var(--brand)', fontWeight: 700 }
                    : cell && mk
                      ? {
                          background: isPredicted ? 'transparent' : markerBg ?? mk.bg,
                          color: mk.color,
                          fontWeight: 700,
                          boxShadow: isPredicted ? `inset 0 0 0 1.5px ${mk.color}` : undefined,
                        }
                      : cell
                        ? { background: 'var(--surface-3)', color: 'var(--ink-3)', fontWeight: 600 }
                        : undefined
              }
            >
              {cell ? localizeNum(cell.day, loc) : ''}
            </button>
          </div>
        );
      })}
    </div>
  );
}

// Marker colour key shown under the expanded month grid — same items and
// colours as the calendar screen's legend so both surfaces read alike.
const LEGEND_KEYS: CycleDayMarker[] = ['pms', 'period', 'fertile', 'ovulation'];

function CalendarLegend({ t }: { t: T }) {
  return (
    <div className="legend">
      {LEGEND_KEYS.map(key => (
        <span key={key} className="legend-item">
          <span className="legend-dot" style={{ background: cycleMarkerStyle[key].color }} />
          <span className="legend-label">{t(`legend.${key}`)}</span>
        </span>
      ))}
    </div>
  );
}

function WeekStrip({
  calOpen, onToggle, loc, t, selectedDay, onSelect,
}: {
  calOpen: boolean;
  onToggle: () => void;
  loc: Locale;
  t: T;
  selectedDay: number | null;
  onSelect: (cell: MonthCell) => void;
}) {
  const tj = todayParts(loc);
  const weeks = useMemo(() => monthMatrix(tj.year, tj.month, loc), [tj.year, tj.month, loc]);
  const monthLabel = formatMonthLabel(tj.year, tj.month, loc);
  // Cycle colors for this month's days, shared with the calendar screen's cache.
  const realCells = useMemo(
    () => weeks.flat().filter((c): c is MonthCell => c !== null),
    [weeks],
  );
  const markOf = useMonthMarks(realCells);
  // Show the week that contains the selected day (falls back to today's week).
  const selIdx = weeks.findIndex(w => w.some(c => c?.day === selectedDay));
  const todayIdx = weeks.findIndex(w => w.some(c => c?.day === tj.day));
  const currentIdx = Math.max(0, selIdx >= 0 ? selIdx : todayIdx);

  return (
    // Top half of the connected calendar↔info card unit — flat bottom so it
    // butts against the pink info panel below (no own margin/radius).
    <div className="home-cal">
        <div className="home-cal-top">
          <span className="home-cal-month">{monthLabel}</span>
          {/* aria-expanded drives both the a11y state and the chevron rotation,
              so the open state is declared once. */}
          <button className="home-cal-toggle" onClick={onToggle} aria-expanded={calOpen}>
            <span className="home-cal-toggle-label">
              {calOpen ? t('week.close') : t('week.fullMonth')}
            </span>
            <Icon name="chevronDown" size={14} className="home-cal-chev" />
          </button>
        </div>

        {/* weekday names — one header row for both states */}
        <div className="home-weekdays">
          {weekdayKeys(loc).map(k => (
            <span key={k} className="home-weekday">
              {t(`week.${k}`)}
            </span>
          ))}
        </div>

        {calOpen ? (
          <div className="cal-drop home-weeks">
            {weeks.map((week, i) => (
              <WeekRow key={i} days={week} todayDay={tj.day} selectedDay={selectedDay} loc={loc} onSelect={onSelect} markOf={markOf} />
            ))}
            <CalendarLegend t={t} />
          </div>
        ) : (
          <WeekRow days={weeks[currentIdx] ?? []} todayDay={tj.day} selectedDay={selectedDay} loc={loc} onSelect={onSelect} markOf={markOf} />
        )}
    </div>
  );
}

// Icon per day-highlight badge on the pink info card. The pills themselves are
// frosted white-on-brand (Apple material), so the icon shape — not a colour —
// carries the identity; accents like amber/indigo never read well on the red.
const HIGHLIGHT_ICONS: Record<CycleDayHighlight, IconName> = {
  period: 'drop',
  fertile: 'heart',
  ovulation: 'sparkle',
  pms: 'info',
  period_tomorrow: 'calendar',
};

// Frosted translucent pills that surface every notable state of the selected
// day (fertile window, PMS, ovulation, imminent period) on top of the red card.
function DayHighlights({ t, items }: { t: T; items: CycleDayHighlight[] }) {
  if (items.length === 0) return null;
  return (
    <div className="home-highlights">
      {items.map(code => (
        <span key={code} className="home-highlight">
          <Icon name={HIGHLIGHT_ICONS[code]} size={13} stroke="currentColor" />
          {t(`dayStatus.${code}`)}
        </span>
      ))}
    </div>
  );
}

// ── Next period hero card ──────────────────────────────────────
function NextPeriodCard({
  t, pred, highlights, daysUntilNextPeriod, cycleDay, cycleLength, nextPeriodDate, phaseLabel, phaseDesc, cardTitle, fertilityBadge, isToday, selectedDateLabel, showPhaseDetails, loading,
}: {
  t: T;
  pred: CyclePredictions | null;
  highlights: CycleDayHighlight[];
  /** Engine headline for the selected day (`daily_card.title`); falls back to the static label. */
  cardTitle: string | null;
  /** Engine fertility read-out for the day, styled exactly as on the day-status card. */
  fertilityBadge: { label: string; style: FertilityBadgeStyle } | null;
  daysUntilNextPeriod: number | null;
  /** Day X of N, resolved by the engine (`cycle_view`) with a local fallback. */
  cycleDay: number | null;
  cycleLength: number | null;
  nextPeriodDate: string | null;
  phaseLabel: string;
  phaseDesc: string;
  isToday: boolean;
  selectedDateLabel: string;
  showPhaseDetails: boolean;
  /** A tapped day's data is still in flight — blank the values out to dashes. */
  loading: boolean;
}) {
  const expanded = true;
  const tLogPeriod = useTranslations('logPeriod');
  const daysValue = daysUntilNextPeriod != null ? t('days', { n: daysUntilNextPeriod }) : t('unavailable');
  // While the tapped day loads, every datum drops to its null/dash fallback
  // instead of showing the previous day's numbers as if they were this day's.
  const shownCycleDay = loading ? null : cycleDay;
  const shownCycleLength = loading ? null : cycleLength;
  // Ring fill = how far the selected day sits into its own cycle, same reading as
  // the cycle screen's day donut (day X of N) rather than a bare fertility number.
  const ringPct =
    shownCycleDay != null && shownCycleLength
      ? Math.min(100, Math.max(0, (shownCycleDay / shownCycleLength) * 100))
      : 0;
  return (
    // Bottom half of the connected calendar↔info unit — fills the wrapper
    // (no own margin/radius); the wrapper owns the rounding + shadow.
    <div className="home-hero">
      {/* Fetch-in-flight overlay for a tapped day — indicator only, the values
          underneath already read as dashes. */}
      {loading && (
        <div className="home-hero-busy" aria-hidden>
          <span className="home-hero-busy-chip">
            <Icon name="loader" size={20} className="home-hero-busy-icon" />
          </span>
        </div>
      )}
      {/* Which day the info below reflects — updates as the user taps a day. */}
      <div className="home-hero-daybar">
        <span className="home-hero-daychip">
          <Icon name="calendar" size={13} stroke="var(--on-accent)" />
          {isToday ? t('selectedDay.today') : t('selectedDay.date', { date: selectedDateLabel })}
        </span>
        {!loading && fertilityBadge && (
          // The badge alone reads as a bare "high/low"; the caption says what the
          // level is *of* — a cycle-timing estimate, not a diagnosis.
          <span className="home-hero-fert-group">
            <span className="home-hero-fert-caption">{t('fertility.caption')}</span>
            <span
              className="home-hero-fert"
              style={{ background: fertilityBadge.style.bg, color: fertilityBadge.style.fg }}
            >
              {fertilityBadge.label}
            </span>
          </span>
        )}
      </div>

      {/* Type ramp: small phase overline → the engine headline → state pills,
          with the cycle-day ring sitting beside the whole stack. */}
      <div className="home-hero-cols">
        <div className="home-hero-left">
          <div className="home-hero-overline">
            {loading
              ? t('unavailable')
              : pred
                ? t('nextPeriod.currentPhase', { phase: phaseLabel })
                : t('nextPeriod.phase')}
          </div>
          <div className="home-hero-days">
            {loading ? t('unavailable') : cardTitle || t('nextPeriod.label')}
          </div>
          <DayHighlights t={t} items={loading ? [] : highlights} />
        </div>

        {/* Cycle-day ring (ported from the cycle screen): the day number inside
            a progress donut over a frosted core. */}
        <div className="home-hero-right">
          {/* Only the sweep angle (the datum) crosses inline; the gradient and
              ring geometry live in the class. */}
          <div
            className="home-ring"
            style={{ '--ring-sweep': `${ringPct * 3.6}deg` } as React.CSSProperties}
          >
            <div className="home-ring-core">
              <span className="home-ring-day">
                {shownCycleDay != null ? t('cycleDay.value', { n: shownCycleDay }) : t('unavailable')}
              </span>
              <span className="home-ring-of">
                {shownCycleLength != null ? t('cycleDay.ofN', { n: shownCycleLength }) : t('cycleDay.label')}
              </span>
            </div>
          </div>
        </div>
      </div>

      {phaseDesc && (
          <div className="home-phase-desc">
            {loading ? t('unavailable') : phaseDesc}
          </div>
      )}

      {/* Deep-link into the phase's full educational content (ported from the
          cycle screen). Only once the engine has a confident sub-phase; the
          target reads the phase from live cycle data, never from the URL (§11).
          Always rendered so the open/close animates; kept out of the tab order
          while collapsed. */}
      {showPhaseDetails && (
        <div className={clsx('home-cta-reveal', expanded && 'is-open')} aria-hidden={!expanded}>
          <div className="home-cta-reveal-inner home-cta-row">
            <Link href="/cycle/phase" className="home-phase-cta home-cta-half" tabIndex={expanded ? undefined : -1}>
              <Icon name="info" size={15} /> {t('phaseDetailsCta')}
            </Link>
            <Link
              href="/calendar?editDates=1"
              className="home-phase-cta home-cta-half"
              tabIndex={expanded ? undefined : -1}
            >
              <Icon name="pencil" size={15} /> {tLogPeriod('dateEditor.open')}
            </Link>
          </div>
        </div>
      )}

    </div>
  );
}

// ── Cycle timeline bar (ported from the cycle screen) ──────────
// A linear day-1 → day-N reading of the cycle: the fertile band, the ovulation
// tick and where today sits. Pinned LTR because a cycle always runs 1 → N
// (§12-safe: it is a chart, not layout chrome).
function CycleTimelineBar({ pred, ovulationDay }: { pred: CyclePredictions; ovulationDay: number }) {
  const at = (day: number) => Math.min(100, Math.max(0, (day / pred.cycleLength) * 100));
  const fertileStart = ovulationDay - 5; // matches the backend fertile window (ovulation − 5)
  const todayPos = at(pred.cycleDay);

  return (
    // Only the positions along the bar stay inline — they are the data.
    <div dir="ltr" className="cyclebar">
      {/* Fertile band */}
      <span
        className="cyclebar-band"
        style={{ left: `${at(fertileStart)}%`, width: `${at(ovulationDay + 1) - at(fertileStart)}%` }}
      />
      {/* Progress up to today */}
      <span className="cyclebar-fill" style={{ width: `${todayPos}%` }} />
      {/* Ovulation tick */}
      <span className="cyclebar-tick" style={{ left: `${at(ovulationDay)}%` }} />
      {/* Today marker */}
      <span className="cyclebar-now" style={{ left: `${todayPos}%` }} />
    </div>
  );
}

/** One upcoming (or currently running) cycle event: its dates and the countdown
 *  to its start — negative once the event itself is under way. */
interface TimelineSlot {
  start: Date;
  end: Date;
  days: number;
}

// ── Phase rows ─────────────────────────────────────────────────
// «رویدادهای پیش‌رو» — the cycle timeline bar, the dates of the upcoming events,
// and the two cycle facts (length, ovulation day) the cycle screen showed.
function PhaseRows({
  t, pred, ovulationDay, windowRange, ovulationDate, pmsRange, nextPeriodDate, daysTo, footer,
}: {
  t: T;
  pred: CyclePredictions | null;
  ovulationDay: number | null;
  windowRange: string | null;
  ovulationDate: string | null;
  pmsRange: string | null;
  nextPeriodDate: string | null;
  /** Days from today to each event's start — the countdown chips. */
  daysTo: {
    pms: TimelineSlot | null;
    nextPeriod: TimelineSlot | null;
    window: TimelineSlot | null;
    ovulation: TimelineSlot | null;
  };
  /** The §12 value layers, rendered where the two cycle facts used to sit. */
  footer?: ReactNode;
}) {
  const dash = t('unavailable');
  // Countdown label: today / in N days; an already-started event shows no chip
  // rather than a negative count.
  const badge = (slot: TimelineSlot | null) => {
    if (!slot) return null;
    if (slot.days > 0) return t('timeline.inDays', { n: slot.days });
    if (slot.days === 0) return t('timeline.today');
    return t('timeline.ongoing');
  };
  // Ordered as the events unfold from here: PMS → period → fertile window,
  // closing on ovulation (per product request: PMS first, ovulation last).
  const rows = [
    { l: t('pms.label'),         d: pmsRange ?? dash,       n: badge(daysTo.pms),        c: 'var(--violet)', bg: 'var(--violet-soft)' },
    { l: t('phases.nextPeriod'), d: nextPeriodDate ?? dash, n: badge(daysTo.nextPeriod), c: 'var(--pink)', bg: 'var(--pink-bg)' },
    { l: t('phases.window'),     d: windowRange ?? dash,    n: badge(daysTo.window),     c: 'var(--amber)', bg: 'var(--amber-soft)' },
    { l: t('phases.ovulation'),  d: ovulationDate ?? dash,  n: badge(daysTo.ovulation),  c: 'var(--green-dot)', bg: 'var(--teal-soft)' },
  ];

  // «جادهٔ چرخه»: the events as stations on a vertical rail — each row a
  // phase-coloured node on the line, its date beneath the label, and the
  // planner's number (the countdown) as the end-side chip.
  return (
    <div className="sec">
      <div className="home-panel">
        <div className="card-titr">
          {t('timeline.title')}
        </div>

        {pred && ovulationDay != null && <CycleTimelineBar pred={pred} ovulationDay={ovulationDay} />}

        <div className="ev-list">
          {rows.map(r => (
            <div key={r.l} className="ev-row">
              <span className="dot ev-node" style={{ background: r.bg, color: r.c }}>
                <DropSolid size={15} color={r.c} />
              </span>
              <div className="ev-body">
                <span className="ev-label">{r.l}</span>
                <span className="ev-date">{r.d}</span>
              </div>
              {r.n && (
                <span className="ev-count" style={{ background: r.bg, color: r.c }}>
                  {r.n}
                </span>
              )}
            </div>
          ))}
        </div>

        {footer}
      </div>
    </div>
  );
}

// ── Daily recommendations ──────────────────────────────────────
/**
 * Category icon per engine tip `type`. Unknown/new backend categories fall back
 * to `sparkle` rather than rendering nothing, so a server-side addition can't
 * blank a row.
 */
const TIP_ICONS: Record<string, IconName> = {
  nutrition: 'apple',
  hydration: 'glass',
  warmth: 'flame',
  rest: 'moon',
  energy: 'zap',
  exercise: 'walk',
  fertility: 'heart',
  pms: 'drop',
  mood: 'smile',
  mental_health: 'brain',
  pain_relief: 'pill',
  sleep: 'moon',
  digestion: 'thermo',
};

/**
 * "توصیه‌های امروز" — the admin-managed recommendations the engine resolved for
 * today, carried on the cycle calculation we already fetch (no extra request).
 * Each tip shows its heading and the localized advice below it. When the engine
 * has no tips (incomplete profile), the short "do" suggestions from the daily
 * message stand in; with neither, the section renders nothing rather than
 * showing invented advice.
 */
function Recommendations({ t, tips, dos }: { t: T; tips: CycleDailyTip[]; dos: string[] }) {
  const items = tips.length > 0
    ? tips.slice(0, 4).map(tip => ({
        // `hasOwn`, not `in`: `in` also matches Object.prototype keys, so a
        // category named `toString` would hand <Icon> a function and then ask
        // for a message key that doesn't exist.
        icon: Object.hasOwn(TIP_ICONS, tip.type) ? TIP_ICONS[tip.type] : ('sparkle' as IconName),
        // The backend resolves the heading (an admin's per-recommendation
        // override, else the category label), so a text edit in the panel shows
        // up without a client release. Only when it sends none do we fall back
        // to our own translation — and only for categories we ship a key for,
        // so a new backend category can't raise a missing-key error. The icon
        // table stays client-side by design: the backend's icon vocabulary is
        // its own, so a brand-new category renders the generic sparkle until a
        // client release adds its glyph.
        title: tip.title
          ?? (Object.hasOwn(TIP_ICONS, tip.type)
            ? t(`recommendations.types.${tip.type}` as 'recommendations.types.nutrition')
            : t('recommendations.fallbackTitle')),
        desc: tip.text as string | undefined,
      }))
    : dos.slice(0, 4).map(text => ({
        icon: 'check' as IconName,
        title: text,
        desc: undefined as string | undefined,
      }));

  if (items.length === 0) return null;

  return (
    <div className="sec">
      <div className="card pad-card-sm">
        <div className="home-rec-title">
          {t('recommendations.title')}
        </div>
        {/* Figma item: soft green→pink gradient, white circular category badge */}
        {items.map((item, i) => (
          <div key={i} className="home-rec-item">
            <span className="home-rec-badge">
              <Icon name={item.icon} size={18} stroke="currentColor" />
            </span>
            <div className="home-rec-body">
              <div className="home-rec-name">{item.title}</div>
              {item.desc && <div className="home-rec-desc">{item.desc}</div>}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// ── Cycle-based articles (published from the admin panel) ─────────
// Figma «Frame 36»: white card, 18px title, 162px blog cards, primary CTA.
// The list comes from `GET /home/sections/articles`, which returns what an
// admin tagged for the phase the user is in today (plus general articles), so
// nothing here is hardcoded. The whole section disappears when there is
// nothing published for this phase.
function Articles({ t, locale }: { t: T; locale: Locale }) {
  const { data, isPending } = useCycleArticles();
  const articles = data?.articles ?? [];

  if (isPending) return <ArticlesSkeleton t={t} />;
  if (articles.length === 0) return null;

  return (
    <div className="sec">
      <div className="home-articles">
        {/* The backend words and localizes the heading; the message file is
            the fallback for when the section omits it. */}
        <div className="home-articles-title">{data?.title ?? t('articles.title')}</div>
        <div className="scroll-x">
          <div className="home-articles-track">
            {articles.map(article => (
              // The card opens the article page; the slug is public content, so
              // unlike the cycle phase it is safe to carry in the URL (§11).
              <Link key={article.id} href={`/articles/${article.slug}`} className="home-article">
                <div className="home-article-cover">
                  {article.imageUrl ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={article.imageUrl}
                      alt=""
                      loading="lazy"
                      draggable={false}
                      className="home-article-img"
                    />
                  ) : (
                    <Icon name="bookOpen" size={30} stroke="currentColor" />
                  )}
                </div>
                <div className="home-article-name">{article.title}</div>
                {(article.readTimeMinutes !== null || article.category) && (
                  <div className="home-article-meta">
                    <Icon name="bookOpen" size={16} stroke="currentColor" />
                    {article.readTimeMinutes !== null
                      ? t('articles.min', { n: localizeNum(article.readTimeMinutes, locale) })
                      : article.category}
                  </div>
                )}
              </Link>
            ))}
          </div>
        </div>
        <Link href="/articles" className="btn btn-primary home-articles-cta">
          {t('articles.readMore')}
        </Link>
      </div>
    </div>
  );
}

/** Placeholder cards while the articles load — same rhythm as the real row. */
function ArticlesSkeleton({ t }: { t: T }) {
  return (
    <div className="sec">
      <div className="home-articles" aria-hidden>
        <div className="home-articles-title">{t('articles.title')}</div>
        <div className="scroll-x">
          <div className="home-articles-track">
            {[0, 1, 2].map(i => (
              <div key={i} className="home-article">
                <span className="skeleton-line home-article-skel" />
                {['90%', '55%'].map(width => (
                  <span key={width} className="skeleton-line" style={{ width }} />
                ))}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

// ── Main export ────────────────────────────────────────────────
export function HomePage() {
  const t = useTranslations('home');
  const loc = useLocale() as Locale;
  const router = useRouter();
  // This route is statically prerendered, so anything derived from "now" would
  // be frozen at build time in the server HTML and disagree with the client on
  // the next day — a hydration text mismatch (React #418). The whole screen is
  // date-driven, so it renders only after mount; its data is client-fetched
  // anyway, so nothing meaningful is lost from the prerendered HTML.
  const mounted = useMounted();
  const [calOpen, setCalOpen] = useState(false);
  // The §12 sync nudge edits the profile in place, in the same bottom sheet the
  // profile screen uses for cycle length — no detour through /profile/health.
  const [cycleSheetOpen, setCycleSheetOpen] = useState(false);
  // The day the user tapped in the mini calendar (defaults to today). Only the
  // connected info card reflects it; the rest of the page stays on today.
  const [selectedDate, setSelectedDate] = useState<Date>(() => today());

  // Server state (§8) — cycle math + personalized message for today.
  const todayQuery = useCycleToday();
  const todayData = todayQuery.data;
  const { data: daily } = useDailyMessage();
  const recalc = useRecalculateCycle();

  const calc = todayData?.calculation ?? null;
  const pred = calc ? deriveCyclePredictions(calc) : null;

  // While the backend recalculates, poll status and refetch today's calc once it
  // settles, so the page reflects the fresh result without a manual reload.
  const recalculating = Boolean(todayData?.isRecalculating) || recalc.isPending;
  const { data: cycleStatus } = useCycleStatus({ poll: recalculating });
  const settled = cycleStatus ? !cycleStatus.is_processing : false;
  useEffect(() => {
    if (recalculating && settled) {
      void todayQuery.refetch();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recalculating, settled]);

  const base = today();
  // The cycle's own calendar (engine anchors → absolute dates). Everything the
  // page shows about upcoming events hangs off this, so the dates stay tied to
  // the period start rather than being re-measured from today each render.
  const schedule = deriveCycleSchedule(todayData?.cycleView ?? null, calc);
  // Calendar formatting happens only here, at the display boundary (§7).
  const fmt = (date: Date) => formatDayMonth(date, loc);
  const range = (from: Date, to: Date) => t('dateRange', { from: fmt(from), to: fmt(to) });
  // Every timeline row describes the occurrence the user is actually waiting on:
  // an event whose window is already running is reported as in-progress, and one
  // that is wholly behind us rolls forward a cycle. Without this the chip simply
  // vanished the moment an event started (a negative countdown), which is what
  // made the fertile-window row lose its badge for a whole week.
  const slotFor = (start: Date, end: Date): TimelineSlot | null => {
    if (!schedule) return null;
    let from = start;
    let to = end;
    while (diffInDays(to, base) < 0) {
      from = addDays(from, schedule.cycleLength);
      to = addDays(to, schedule.cycleLength);
    }
    return { start: from, end: to, days: diffInDays(from, base) };
  };
  const nextPeriodSlot = schedule
    ? slotFor(schedule.nextPeriodStart, schedule.nextPeriodStart)
    : null;
  const ovulationSlot = schedule ? slotFor(schedule.ovulation, schedule.ovulation) : null;
  const windowSlot = schedule ? slotFor(schedule.fertileStart, schedule.fertileEnd) : null;
  const pmsSlot = schedule ? slotFor(schedule.pmsStart, schedule.pmsEnd) : null;
  const nextPeriodDate = nextPeriodSlot ? fmt(nextPeriodSlot.start) : null;
  const ovulationDate = ovulationSlot ? fmt(ovulationSlot.start) : null;
  const windowRange = windowSlot ? range(windowSlot.start, windowSlot.end) : null;
  const pmsRange = pmsSlot ? range(pmsSlot.start, pmsSlot.end) : null;

  const message: DailyMessage | undefined = daily;
  const dos = message?.primary.dos ?? [];

  // ── Selected-day info for the connected calendar↔info card ──
  const selApiDate = toApiDate(selectedDate);
  const isToday = selApiDate === toApiDate(base);
  const selParts = toParts(selectedDate, loc);
  const monthParts = todayParts(loc);
  const selectedDay =
    selParts.year === monthParts.year && selParts.month === monthParts.month
      ? selParts.day
      : null;

  // A tapped past/future day loads its own calculation + message; today reuses
  // the queries above (same cache keys — no extra fetch).
  const { data: dateData, isFetching: dateFetching } = useCycleForDate(selApiDate, !isToday);
  const { data: infoMessage } = useDailyMessage(isToday ? undefined : selApiDate);

  const infoCalc = (isToday ? todayData : dateData)?.calculation ?? null;
  const infoPred = infoCalc ? deriveCyclePredictions(infoCalc) : null;
  const infoHighlights = infoCalc ? deriveDayHighlights(infoCalc) : [];

  // The countdown measures the selected day against *its own* cycle's predicted
  // start: the engine's anchors for that day once they load (a past day then
  // shows the period it actually led to), and until then today's schedule rolled
  // whole cycles — so scrubbing counts down smoothly instead of jumping.
  const selectedSchedule: CycleSchedule | null =
    (isToday ? null : deriveCycleSchedule(dateData?.cycleView ?? null, dateData?.calculation ?? null)) ??
    (schedule ? cycleScheduleFor(schedule, selectedDate) : null);
  // The engine's own read-out for the selected day — the very same `cycle_view`
  // that renders the day-status card — so the hero can never disagree with it.
  const infoView = (isToday ? todayData : dateData)?.cycleView ?? null;
  const infoDaysUntilNextPeriod =
    infoView?.daysToPeriod ??
    (selectedSchedule ? daysUntilNextPeriod(selectedSchedule, selectedDate) : null);
  const infoNextPeriodStart =
    infoView?.forecast?.nextPeriodStart
      ? fromApiDate(infoView.forecast.nextPeriodStart)
      : (selectedSchedule?.nextPeriodStart ?? null);
  const infoNextPeriodDate = infoNextPeriodStart ? fmt(infoNextPeriodStart) : null;
  // Day X of N, again straight from the engine (falls back to the local derivation).
  const infoCycleDay = infoView?.cycleDay ?? infoPred?.cycleDay ?? null;
  const infoCycleLength =
    infoView?.effectiveValues.cycleLength ?? infoPred?.cycleLength ?? null;
  const infoPhaseLabel = infoPred ? t(`phaseLabel.${infoPred.phase}`) : '';
  // The line inside the hero is the engine's own day-status copy — the exact
  // subtitle the day-status card shows (§19) — so the two never tell the user
  // different things about the same day. Behind it: the personalized message for
  // today, then a tense-neutral phase blurb (a tapped day must never say «امروز»).
  // Headline + fertility badge of the engine's day-status card, reused verbatim.
  const infoCardTitle = infoView?.dailyCard?.title || null;
  const infoFertilityBadge = infoView?.dailyCard?.fertilityLabel
    ? {
        label: infoView.dailyCard.fertilityLabel,
        style: fertilityBadgeStyle(infoView.dailyCard.fertilityLevel),
      }
    : null;
  const infoPhaseDesc =
    infoView?.dailyCard?.subtitle ||
    (isToday
      ? (infoMessage?.primary.shortMessage || t('nextPeriod.phaseDesc'))
      : (infoPred ? t(`phaseDescription.${infoPred.phase}`) : t('nextPeriod.phaseDesc')));
  // Selecting a past/future day fetches its data; dim the card meanwhile so the
  // placeholder values read as "loading", not as a broken empty state (§ loading).
  const infoLoading = !isToday && dateFetching && !dateData;
  const selectedDateLabel = formatDayMonth(selectedDate, loc);

  // Server pass / first client render: backdrop only, so both sides match.
  if (!mounted) {
    return (
      <div className="view">
        <div className="home-grad home-grad-fill" />
      </div>
    );
  }

  return (
    <div className="view">
      {/* Full-page gradient backdrop (lavender → soft turquoise, §10.2) */}
      <div className="home-grad home-grad-fill" />

      <div className="scroll page-scroll">
        <HomeHeader
          tagline={t('tagline')}
          onRecalculate={() => recalc.mutate()}
          recalculating={recalculating}
          recalculateLabel={t('recalculate')}
        />
        {recalculating && (
          <div className="page-updating">
            {t('updating')}
          </div>
        )}
        {/* Connected unit: the mini calendar butts directly against the info
            card below it, and tapping a day updates that card (§ home request). */}
        <div className="home-cal-unit">
          <div className="home-cal-shell">
            <WeekStrip
              calOpen={calOpen}
              onToggle={() => setCalOpen(v => !v)}
              loc={loc}
              t={t}
              selectedDay={selectedDay}
              onSelect={(cell) => setSelectedDate(cell.date)}
            />
          </div>
          <div className={clsx('home-cal-shell', infoLoading && 'is-loading')}>
            <NextPeriodCard
              t={t}
              pred={infoPred}
              highlights={infoHighlights}
              daysUntilNextPeriod={infoDaysUntilNextPeriod}
              cycleDay={infoCycleDay}
              cycleLength={infoCycleLength}
              nextPeriodDate={infoNextPeriodDate}
              phaseLabel={infoPhaseLabel}
              phaseDesc={infoPhaseDesc}
              cardTitle={infoCardTitle}
              fertilityBadge={infoFertilityBadge}
              isToday={isToday}
              selectedDateLabel={selectedDateLabel}
              showPhaseDetails={isToday && Boolean(todayData?.cycleView?.subphase)}
              loading={infoLoading}
            />
          </div>
        </div>
        {/* Admin-managed promo slot — renders nothing until a banner is active */}
        <BannerSlideshow position="home_top" />
        <PhaseRows
          t={t}
          pred={pred}
          ovulationDay={calc?.estimatedOvulationDay ?? null}
          windowRange={windowRange}
          ovulationDate={ovulationDate}
          pmsRange={pmsRange}
          nextPeriodDate={nextPeriodDate}
          daysTo={{
            pms: pmsSlot,
            nextPeriod: nextPeriodSlot,
            window: windowSlot,
            ovulation: ovulationSlot,
          }}
          /* The §12 value layers — what the profile says, what recent cycles
             suggest, and which layer today's prediction actually used. They
             took over the slot the two cycle facts used to hold. */
          footer={todayData?.cycleView && (
            <CycleValuesCard
              title={t('values.title')}
              loggedLabel={t('values.logged')}
              loggedValue={
                todayData.cycleView.profileValues.cycleLength != null
                  ? t('days', { n: todayData.cycleView.profileValues.cycleLength })
                  : t('unavailable')
              }
              suggestion={
                todayData.cycleView.calculatedValues.cycleLength != null &&
                todayData.cycleView.profileValues.cycleLength != null &&
                todayData.cycleView.calculatedValues.cycleLength !==
                  todayData.cycleView.profileValues.cycleLength
                  ? {
                      text: t('values.suggestion', {
                        n: todayData.cycleView.calculatedValues.cycleLength,
                      }),
                      ctaLabel: t('values.syncCta', {
                        n: todayData.cycleView.calculatedValues.cycleLength,
                      }),
                      onSync: () => setCycleSheetOpen(true),
                    }
                  : null
              }
              basedOnText={t('values.basedOn', {
                source: t(
                  todayData.cycleView.effectiveValues.source === 'recent_valid_cycles'
                    ? 'values.source.recent_valid_cycles'
                    : todayData.cycleView.effectiveValues.source === 'profile'
                      ? 'values.source.profile'
                      : 'values.source.default',
                ),
              })}
            />
          )}
        />
        <Recommendations t={t} tips={calc?.dailyTips ?? []} dos={dos} />
        <BannerSlideshow position="home_middle" />
        {/* Today's doctor/medication reminders and to-dos — same source as the
            daily-log day planner, so items set there appear here (§ home request).
            Temporarily hidden per product request. */}
        {/* <DayTasks date={base} /> */}
        <TodayChallengeCard />
        <Articles t={t} locale={loc} />
        <BannerSlideshow position="home_bottom" />
        <div className="page-tail" />
      </div>

      {/* Seeded with what recent cycles measured, so the wheel opens on the
          value the nudge proposes and saving is a single tap. */}
      <QuickEditSheet
        field={cycleSheetOpen ? 'cycleDuration' : null}
        values={{
          cycleDuration:
            todayData?.cycleView?.calculatedValues.cycleLength ??
            todayData?.cycleView?.profileValues.cycleLength ??
            null,
        }}
        onClose={() => setCycleSheetOpen(false)}
      />

      <BottomNav />
    </div>
  );
}
