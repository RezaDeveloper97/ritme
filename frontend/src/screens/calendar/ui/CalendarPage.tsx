'use client';


import clsx from 'clsx';
import { useFormatter, useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useRef, useState } from 'react';

import {
  calcToPhase,
  cycleDayMarker,
  cycleMarkerStyle,
  normalizePhase,
  useCycleMonth,
  useCycleStatus,
  type CycleCalculation,
  type CycleDayMarker,
  type CyclePhase,
} from '@/entities/cycle';
import { useUserProfile } from '@/entities/user';
import {
  PeriodDateEditor,
  PeriodEditor,
  useDeletePeriod,
  usePeriodHistory,
  useStartPeriod,
  useUpdatePeriod,
  type LoggedPeriod,
  type PeriodSaveInfo,
} from '@/features/log-period';
import { useRouter, type Locale } from '@/shared/i18n';
import {
  addDays,
  diffInDays,
  formatJalaliDayMonth,
  formatJalaliMonthLabel,
  formatJalaliYear,
  fromApiDate,
  jalaliMonthMatrix,
  jalaliMonthName,
  shiftJalaliMonth,
  toApiDate,
  today,
  todayJalali,
  type JalaliMonthCell,
} from '@/shared/lib/date';
import { getApiErrorMessage } from '@/shared/api';
import { Icon, Sheet } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { DayLogSummary } from './DayLogSummary';

const WEEKDAY_KEYS = ['sat', 'sun', 'mon', 'tue', 'wed', 'thu', 'fri'] as const;

// Colors for each calendar marker — shared with the home mini calendar so both
// surfaces stay in sync. Follicular/luteal days carry no marker and read as
// neutral. The legend below uses the same source.
const MARKER_STYLE = cycleMarkerStyle;

const NEUTRAL_STYLE = { bg: 'var(--indigo-soft)', color: 'var(--indigo)' };

// Fallback bleeding length when the profile hasn't recorded one (matches the
// backend default), used to optimistically fill cells on a quick period start.
const DEFAULT_PERIOD_DAYS = 5;

const isSameDay = (a: Date, b: Date) => diffInDays(a, b) === 0;

/** Every ISO day in the inclusive [start, end] range. */
function isoRange(start: string, end: string): string[] {
  const days: string[] = [];
  const first = fromApiDate(start);
  const count = diffInDays(fromApiDate(end), first) + 1;
  for (let i = 0; i < count; i += 1) days.push(toApiDate(addDays(first, i)));
  return days;
}

/**
 * Optimistic paint layer over the calendar: `paint` renders as period at once,
 * `clear` suppresses stale period cells (after an edit/delete), until the
 * refetched backend data confirms the change at `confirmIso`.
 */
interface Overlay {
  paint: Set<string>;
  clear: Set<string>;
  confirmIso: string;
  expectPeriod: boolean;
  /** Freshness of the month data when the overlay was created (see drop effect). */
  since: number;
}

// Horizontal drag past this many px switches month (mobile swipe).
const SWIPE_THRESHOLD_PX = 40;

// Vertical pitch of one calendar week row (40px cell + 4px flex gap).
const WEEK_ROW_PX = 44;

// Scroll-driven collapse thresholds. Deliberately asymmetric (hysteresis):
// the grid folds only once the user is clearly scrolling away, and unfolds
// only when they're back at the very top — a single shared threshold would
// let the state flutter around it.
const COLLAPSE_AT_PX = 48;
const EXPAND_AT_PX = 6;

// Approximate height of the rule + legend block that folds away with the
// grid, compensated at the page tail (see the tail spacer below).
const EXTRAS_PX = 56;
const PAGE_TAIL_PX = 26;

// A day this close to an existing logged period reads as part of the SAME
// bleed — tapping it extends that period instead of starting a new one, so
// "start period" can't create back-to-back periods days apart.
const EXTEND_GAP_DAYS = 7;

// Quick "start period" is only offered when the day is at least a rough cycle
// away from every logged period: two real periods can't be this close, so
// anything nearer is a correction (extend/edit), never a new period.
const NEW_PERIOD_MIN_GAP_DAYS = 21;

type T = ReturnType<typeof useTranslations>;

/** Gregorian year/month a date falls in, parsed from its API serialization. */
function gregYearMonth(date: Date): { year: number; month: number } {
  const [year, month] = toApiDate(date).split('-');
  return { year: Number(year), month: Number(month) };
}

/** Relative conception likelihood for the day, from its phase (informational, §11). */
function chanceKey(phase: CyclePhase): 'low' | 'medium' | 'high' {
  if (phase === 'ovulation') return 'high';
  if (phase === 'fertile') return 'medium';
  return 'low';
}

// ── Month navigation bar ───────────────────────────────────────
interface MonthNavProps {
  label: string;
  isRtl: boolean;
  onPrev: () => void;
  onNext: () => void;
  onJump: () => void;
  fullMonth: boolean;
  onToggle: () => void;
  toggleLabel: string;
  jumpLabel: string;
  prevMonthLabel: string;
  nextMonthLabel: string;
}

function MonthNav({ label, isRtl, onPrev, onNext, onJump, fullMonth, onToggle, toggleLabel, jumpLabel, prevMonthLabel, nextMonthLabel }: MonthNavProps) {
  return (
    <div className="cal-nav">
      {/* First child → RIGHT in RTL = previous month; label follows the action. */}
      <button className="iconbtn" onClick={isRtl ? onPrev : onNext} aria-label={isRtl ? prevMonthLabel : nextMonthLabel}>
        <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
      </button>

      <div className="cal-nav-mid">
        {/* Tapping the label jumps back to the current month. */}
        <button className="cal-nav-label" onClick={onJump} aria-label={jumpLabel}>
          {label}
          <Icon name="chevronDown" size={14} />
        </button>
        <button
          onClick={onToggle}
          className={clsx('chip', 'cal-nav-toggle', fullMonth && 'on')}
        >
          <Icon name={fullMonth ? 'grid' : 'calendar'} size={13} />
          {toggleLabel}
        </button>
      </div>

      {/* Last child → LEFT in RTL = next month; label follows the action. */}
      <button className="iconbtn" onClick={isRtl ? onNext : onPrev} aria-label={isRtl ? nextMonthLabel : prevMonthLabel}>
        <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={20} />
      </button>
    </div>
  );
}

// ── Month/year quick picker (opened by tapping the month label) ─
interface MonthYearPickerProps {
  t: T;
  locale: Locale;
  isRtl: boolean;
  open: boolean;
  onClose: () => void;
  year: number;
  month: number;
  onPick: (year: number, month: number) => void;
  onToday: () => void;
}

function MonthYearPicker({ t, locale, isRtl, open, onClose, year, month, onPick, onToday }: MonthYearPickerProps) {
  // Local year the user is browsing; resets to the shown month's year on open.
  const [browseYear, setBrowseYear] = useState(year);
  useEffect(() => {
    if (open) setBrowseYear(year);
  }, [open, year]);

  return (
    <Sheet open={open} onClose={onClose} labelledBy="month-picker-title">
      <div className="cal-pick-head">
        {/* First child → RIGHT in RTL = previous year */}
        <button
          className="iconbtn"
          onClick={() => setBrowseYear((y) => y + (isRtl ? -1 : 1))}
          aria-label={isRtl ? 'سال قبل' : 'Next year'}
        >
          <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
        </button>
        <div id="month-picker-title" className="cal-pick-year">
          {formatJalaliYear(browseYear, locale)}
        </div>
        <button
          className="iconbtn"
          onClick={() => setBrowseYear((y) => y + (isRtl ? 1 : -1))}
          aria-label={isRtl ? 'سال بعد' : 'Previous year'}
        >
          <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={20} />
        </button>
      </div>

      <div className="cal-pick-grid">
        {Array.from({ length: 12 }, (_, i) => i + 1).map((m) => {
          const isCurrent = m === month && browseYear === year;
          return (
            <button
              key={m}
              className={clsx('chip', 'cal-pick-month', isCurrent && 'on')}
              onClick={() => onPick(browseYear, m)}
            >
              {jalaliMonthName(m, locale)}
            </button>
          );
        })}
      </div>

      <button className="btn cal-pick-today" onClick={onToday}>
        {t('goToToday')}
      </button>
    </Sheet>
  );
}

// ── One day cell ───────────────────────────────────────────────
interface DayCellProps {
  cell: JalaliMonthCell | null;
  selectedDate: Date;
  onSelect: (date: Date) => void;
  dayNumber: string;
  marker: CycleDayMarker | null;
  /** Whether the day is inside a real logged period (vs an engine prediction). */
  isLogged: boolean;
  /** Full spoken label (date + cycle state) for screen readers — the visible digit is aria-hidden. */
  ariaLabel: string;
}

function DayCell({ cell, selectedDate, onSelect, dayNumber, marker, isLogged, ariaLabel }: DayCellProps) {
  if (!cell) return <span />;

  const markerStyle = marker ? MARKER_STYLE[marker] : undefined;
  const selected = isSameDay(cell.date, selectedDate);
  const isToday = isSameDay(cell.date, today());
  // A period marker the user hasn't actually logged is a prediction → hollow ring,
  // not a solid fill (spec §12: predicted periods must read differently from real ones).
  const isPredicted = marker === 'period' && !isLogged;

  return (
    <button
      onClick={() => onSelect(cell.date)}
      aria-label={ariaLabel}
      aria-pressed={selected}
      aria-current={isToday ? 'date' : undefined}
      style={{
        position: 'relative',
        height: 40,
        border: selected ? '2px solid var(--brand)' : '2px solid transparent',
        borderRadius: 14,
        background: markerStyle && !isPredicted ? markerStyle.bg : 'transparent',
        color: markerStyle ? markerStyle.color : 'var(--ink)',
        fontFamily: 'inherit',
        fontSize: 14,
        fontWeight: 700,
        cursor: 'pointer',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        boxShadow: isPredicted && markerStyle ? `inset 0 0 0 1.5px ${markerStyle.color}` : undefined,
        fontVariantNumeric: 'tabular-nums',
      }}
    >
      <span aria-hidden>{dayNumber}</span>
      {isToday && (
        <span
          style={{
            position: 'absolute',
            bottom: 4,
            width: 4,
            height: 4,
            borderRadius: '50%',
            background: markerStyle ? markerStyle.color : 'var(--brand)',
          }}
        />
      )}
    </button>
  );
}

// ── Loading overlay over the month grid ────────────────────────
// Animated, on-brand loader (pulsing drop + orbiting legend-colored dots)
// shown while month data is being fetched or the cycle is recalculating.
function CalendarLoader({ label }: { label: string }) {
  return (
    <div className="cal-loader-overlay" role="status" aria-live="polite">
      <div className="cal-loader" aria-hidden>
        <div className="cal-loader-orbit">
          <span /><span /><span /><span />
        </div>
        <span className="cal-loader-core">
          <Icon name="drop" size={20} fill="currentColor" strokeWidth={0} />
        </span>
      </div>
      <span className="cal-loader-label">{label}</span>
    </div>
  );
}

// ── Legend ─────────────────────────────────────────────────────
function Legend({ t }: { t: T }) {
  const items: { key: CycleDayMarker }[] = [
    { key: 'period' },
    { key: 'fertile' },
    { key: 'ovulation' },
    { key: 'pms' },
  ];
  return (
    <div className="legend">
      {items.map((it) => (
        <span key={it.key} className="legend-item">
          <span className="legend-dot" style={{ background: MARKER_STYLE[it.key].color }} />
          <span className="legend-label">{t(`legend.${it.key}`)}</span>
        </span>
      ))}
    </div>
  );
}

// ── Selected-day detail (inside the day sheet) ─────────────────
interface DayDetailProps {
  t: T;
  locale: Locale;
  selectedDate: Date;
  calc: CycleCalculation | undefined;
  marker: CycleDayMarker | null;
  /** Phase/chance tiles — hidden when the richer daily status card is shown below. */
  showTiles?: boolean;
}

function DayDetail({ t, locale, selectedDate, calc, marker, showTiles = true }: DayDetailProps) {
  const phase = calc ? calcToPhase(calc) : undefined;
  const style = (marker && MARKER_STYLE[marker]) ?? NEUTRAL_STYLE;
  const isToday = isSameDay(selectedDate, today());
  // Prefer the marker for the label (so PMS/fertile read as themselves) and fall
  // back to the underlying phase.
  const labelKey = marker ?? phase;

  return (
    <div className="cal-day-card">
      <div className="cal-day-top">
        <div className="cal-day-left">
          <span className="dot cal-day-dot" style={{ background: style.bg, color: style.color }}>
            <Icon name="drop" size={20} fill="currentColor" strokeWidth={0} />
          </span>
          <div className="text-start">
            <div id="day-sheet-title" className="cal-day-title">
              {formatJalaliDayMonth(selectedDate, locale)}
            </div>
            {calc && (
              <div className="cal-day-sub">
                {t('day.cycleDay', { n: calc.cycleDay })}
              </div>
            )}
          </div>
        </div>
        {isToday && (
          <span className="cal-day-chip">
            {t('today')}
          </span>
        )}
      </div>

      {showTiles && phase && labelKey && (
        <div className="cal-day-stats">
          <div className="cal-day-stat" style={{ background: style.bg }}>
            <div className="cal-day-stat-l">{t('phaseLabel')}</div>
            <div className="cal-day-stat-v" style={{ color: style.color }}>{t(`phase.${labelKey}`)}</div>
          </div>
          <div className="cal-day-stat">
            <div className="cal-day-stat-l">{t('day.chanceLabel')}</div>
            <div className="cal-day-stat-v">{t(`chance.${chanceKey(phase)}`)}</div>
          </div>
        </div>
      )}
    </div>
  );
}

// ── Main export ────────────────────────────────────────────────
export function CalendarPage() {
  const t = useTranslations('calendar');
  const tLogPeriod = useTranslations('logPeriod');
  const locale = useLocale() as Locale;
  const format = useFormatter();
  const router = useRouter();
  const isRtl = locale === 'fa';

  const [{ year, month }, setView] = useState(() => {
    const j = todayJalali();
    return { year: j.year, month: j.month };
  });
  const [selectedDate, setSelectedDate] = useState<Date>(() => today());
  const [fullMonth, setFullMonth] = useState(true);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  // The full-screen "Edit Period Date" toggle editor (opened from the header button).
  const [dateEditorOpen, setDateEditorOpen] = useState(false);
  // Watch the backend recalculation only right after we trigger one.
  const [watching, setWatching] = useState(false);
  // Transient message surfaced when a period edit is rejected by the backend
  // (overlap, previous period still open, …) — spec §16 validation feedback.
  const [actionError, setActionError] = useState<string | null>(null);
  // Optimistic paint/clear layer applied immediately after any period change
  // (quick start, edit, delete), until the fresh month data catches up.
  const [overlay, setOverlay] = useState<Overlay | null>(null);
  // The logged period the editor is currently editing (null = create mode).
  const [editingPeriod, setEditingPeriod] = useState<LoggedPeriod | null>(null);

  const profileQuery = useUserProfile();
  const periodDuration = profileQuery.data?.health?.periodDuration ?? DEFAULT_PERIOD_DAYS;

  const startPeriod = useStartPeriod();
  const updatePeriod = useUpdatePeriod();
  const deletePeriod = useDeletePeriod();
  const historyQuery = usePeriodHistory();
  const status = useCycleStatus({ poll: watching });
  const isRecalculating = watching && (status.data?.is_processing ?? false);
  useEffect(() => {
    if (watching && status.data && !status.data.is_processing) setWatching(false);
  }, [watching, status.data]);

  const openLog = () => router.push(`/log?date=${toApiDate(selectedDate)}`);

  const weeks = useMemo(() => jalaliMonthMatrix(year, month), [year, month]);

  // A Jalali month spans at most two Gregorian months; fetch both and merge.
  // Each calculation carries its own date, so overlap is harmless.
  const realCells = useMemo(
    () => weeks.flat().filter((c): c is JalaliMonthCell => c !== null),
    [weeks],
  );
  const gA = gregYearMonth(realCells[0]?.date ?? today());
  const gB = gregYearMonth(realCells[realCells.length - 1]?.date ?? today());
  const monthA = useCycleMonth(gA.year, gA.month);
  const monthB = useCycleMonth(gB.year, gB.month);

  const calcMap = useMemo(() => {
    const map = new Map<string, CycleCalculation>();
    for (const c of monthA.data?.calculations ?? []) map.set(c.calculationDate, c);
    for (const c of monthB.data?.calculations ?? []) map.set(c.calculationDate, c);
    return map;
  }, [monthA.data, monthB.data]);

  // Freshest fetch time across the two visible Gregorian months; the overlay
  // records this at creation so we know when genuinely new data has arrived.
  const dataFreshness = Math.max(monthA.dataUpdatedAt, monthB.dataUpdatedAt);

  // Drop the optimistic overlay once the refetched data confirms the change at
  // its anchor day — or, failing that (e.g. a deleted period whose start is
  // still a *predicted* period day), once fresher month data has landed and the
  // recalculation is no longer being watched.
  useEffect(() => {
    if (!overlay) return;
    const c = calcMap.get(overlay.confirmIso);
    const isPeriod = c ? normalizePhase(c.phase) === 'period' : false;
    if (isPeriod === overlay.expectPeriod) setOverlay(null);
    else if (!watching && dataFreshness > overlay.since) setOverlay(null);
  }, [overlay, calcMap, watching, dataFreshness]);

  const markerFor = (date: Date): CycleDayMarker | null => {
    const iso = toApiDate(date);
    if (overlay?.paint.has(iso)) return 'period';
    if (overlay?.clear.has(iso)) return null;
    const c = calcMap.get(iso);
    return c ? cycleDayMarker(c) : null;
  };

  // Spoken label for a day cell (§ a11y): the Jalali date plus its cycle state and
  // today/selected context, so screen-reader users get more than a bare digit.
  const dayAriaLabel = (date: Date): string => {
    const parts = [formatJalaliDayMonth(date, locale)];
    const marker = markerFor(date);
    if (marker) parts.push(t(`legend.${marker}`));
    if (isSameDay(date, today())) parts.push(t('today'));
    if (isSameDay(date, selectedDate)) parts.push(t('selectedDay'));
    return parts.join('، ');
  };

  // The logged (real, user-entered) period covering a date, if any — this is
  // what the day sheet offers to edit. An open period covers its start plus the
  // profile's usual bleeding length.
  const loggedPeriodFor = (date: Date): LoggedPeriod | null => {
    const iso = toApiDate(date);
    for (const p of historyQuery.data ?? []) {
      const end =
        p.period_end_date ??
        toApiDate(addDays(fromApiDate(p.period_start_date), periodDuration - 1));
      if (iso >= p.period_start_date && iso <= end) return p;
    }
    return null;
  };

  // A period-colored day is "actual" only when the user really logged it (or is
  // optimistically painting it right now); otherwise the engine merely predicted
  // it, so the cell renders as a hollow prediction (spec §12).
  const isActualPeriodDay = (date: Date): boolean => {
    const iso = toApiDate(date);
    if (overlay?.paint.has(iso)) return true;
    if (overlay?.clear.has(iso)) return false;
    return loggedPeriodFor(date) !== null;
  };

  // Last day a logged period visibly covers (open periods run for the profile's
  // usual bleeding length — mirrors `loggedPeriodFor`).
  const effectiveEndOf = (p: LoggedPeriod): string =>
    p.period_end_date ?? toApiDate(addDays(fromApiDate(p.period_start_date), periodDuration - 1));

  // A logged period the selected day could EXTEND (the day sits within
  // EXTEND_GAP_DAYS just before its start or after its end). When one exists,
  // the day sheet offers "add to period days" instead of "start period".
  const extendTargetFor = (date: Date): { period: LoggedPeriod; newStart: string; newEnd: string | null } | null => {
    if (diffInDays(date, today()) > 0) return null;
    const iso = toApiDate(date);
    let best: { period: LoggedPeriod; newStart: string; newEnd: string | null; gap: number } | null = null;
    for (const p of historyQuery.data ?? []) {
      const end = effectiveEndOf(p);
      let candidate: { newStart: string; newEnd: string | null; gap: number } | null = null;
      if (iso < p.period_start_date) {
        const gap = diffInDays(fromApiDate(p.period_start_date), date);
        // Extending backwards keeps the recorded end (possibly still open).
        if (gap <= EXTEND_GAP_DAYS) candidate = { newStart: iso, newEnd: p.period_end_date, gap };
      } else if (iso > end) {
        const gap = diffInDays(date, fromApiDate(end));
        if (gap <= EXTEND_GAP_DAYS) candidate = { newStart: p.period_start_date, newEnd: iso, gap };
      }
      if (candidate && (!best || candidate.gap < best.gap)) best = { period: p, ...candidate };
    }
    return best && { period: best.period, newStart: best.newStart, newEnd: best.newEnd };
  };

  // Days from `date` to the nearest logged period (0 = inside one).
  const gapToNearestLoggedPeriod = (date: Date): number => {
    let nearest = Infinity;
    for (const p of historyQuery.data ?? []) {
      const start = fromApiDate(p.period_start_date);
      const end = fromApiDate(effectiveEndOf(p));
      const gap = diffInDays(date, start) < 0 ? diffInDays(start, date) : Math.max(0, diffInDays(date, end));
      nearest = Math.min(nearest, gap);
    }
    return nearest;
  };

  // ── Scroll-driven collapse (iOS-calendar style) ──────────────
  // Scrolling down folds the month grid to the active week's strip (the card
  // stays pinned via sticky); returning to the top unfolds it. The manual
  // week-view toggle reuses the same folded state, so it animates too.
  const scrollRef = useRef<HTMLDivElement>(null);
  const [scrollCollapsed, setScrollCollapsed] = useState(false);
  // Extra tail height while folded — only as much as a short page needs so the
  // fold can't shrink the scroll range under the expand threshold (which would
  // re-expand the card in a loop). Long pages get 0 → no dead space at the end.
  const [tailPad, setTailPad] = useState(0);
  const onPageScroll = () => {
    const el = scrollRef.current;
    if (!el) return;
    const y = el.scrollTop;
    const next = scrollCollapsed ? y > EXPAND_AT_PX : y > COLLAPSE_AT_PX;
    if (next === scrollCollapsed) return;
    if (next) {
      const delta = (weeks.length - 1) * WEEK_ROW_PX + EXTRAS_PX;
      const roomAfterFold = el.scrollHeight - delta - el.clientHeight;
      setTailPad(Math.max(0, COLLAPSE_AT_PX + 12 - roomAfterFold));
    } else {
      setTailPad(0);
    }
    setScrollCollapsed(next);
  };
  const collapsed = !fullMonth || scrollCollapsed;

  // The week the folded strip shows: the selected day's week, else today's,
  // else the first (e.g. after jumping to a month containing neither).
  const activeWeekIndex = useMemo(() => {
    const bySelected = weeks.findIndex((w) => w.some((c) => c && isSameDay(c.date, selectedDate)));
    if (bySelected >= 0) return bySelected;
    const byToday = weeks.findIndex((w) => w.some((c) => c && isSameDay(c.date, today())));
    return byToday >= 0 ? byToday : 0;
  }, [weeks, selectedDate]);

  const goMonth = (delta: number) => setView((v) => shiftJalaliMonth(v.year, v.month, delta));
  const goToday = () => {
    const j = todayJalali();
    setView({ year: j.year, month: j.month });
    setSelectedDate(today());
    setPickerOpen(false);
  };

  // Month/year quick-picker: jump straight to any Jalali month.
  const pickMonth = (pickedYear: number, pickedMonth: number) => {
    setView({ year: pickedYear, month: pickedMonth });
    setPickerOpen(false);
  };

  const selectDay = (date: Date) => {
    setSelectedDate(date);
  };

  // Quick period logging: tapping "start here" re-anchors the cycle on `date`,
  // optimistically fills the next `periodDuration` cells as period, and lets the
  // backend regenerate every downstream prediction (§11: date only, never logged).
  const startPeriodHere = () => {
    const iso = toApiDate(selectedDate);
    const end = toApiDate(addDays(selectedDate, periodDuration - 1));
    setOverlay({
      paint: new Set(isoRange(iso, end)),
      clear: new Set(),
      confirmIso: iso,
      expectPeriod: true,
      since: dataFreshness,
    });
    setWatching(true);
    startPeriod.mutate(
      { date: iso },
      // Roll back the optimistic fill if the re-anchor fails.
      { onError: onMutationError },
    );
  };

  // Tapped day sits just outside a logged period → stretch that period to
  // cover it. Optimistically paints the whole widened range at once.
  const addToPeriodHere = (target: { period: LoggedPeriod; newStart: string; newEnd: string | null }) => {
    const iso = toApiDate(selectedDate);
    const paintEnd = target.newEnd ?? effectiveEndOf(target.period);
    setOverlay({
      paint: new Set(isoRange(target.newStart, paintEnd)),
      clear: new Set(),
      confirmIso: iso,
      expectPeriod: true,
      since: dataFreshness,
    });
    setWatching(true);
    updatePeriod.mutate(
      { id: target.period.id, start: target.newStart, end: target.newEnd },
      { onError: onMutationError },
    );
  };

  // Unmark the tapped day of a logged period. Removing the first day moves the
  // start forward; any other day truncates the period to the day before it
  // (days are contiguous, so "this day wasn't period" ends the bleed there);
  // a single-day period is deleted outright.
  const removeDayHere = (period: LoggedPeriod) => {
    const iso = toApiDate(selectedDate);
    const effEnd = effectiveEndOf(period);
    const rollback = { onError: onMutationError };
    const overlayFor = (cleared: string[]) => ({
      paint: new Set<string>(),
      clear: new Set(cleared),
      confirmIso: iso,
      expectPeriod: false,
      since: dataFreshness,
    });

    setWatching(true);
    if (iso === period.period_start_date && iso === effEnd) {
      setOverlay(overlayFor([iso]));
      deletePeriod.mutate({ id: period.id }, rollback);
    } else if (iso === period.period_start_date) {
      setOverlay(overlayFor([iso]));
      updatePeriod.mutate(
        { id: period.id, start: toApiDate(addDays(selectedDate, 1)), end: period.period_end_date },
        rollback,
      );
    } else {
      setOverlay(overlayFor(isoRange(iso, effEnd)));
      updatePeriod.mutate(
        { id: period.id, start: period.period_start_date, end: toApiDate(addDays(selectedDate, -1)) },
        rollback,
      );
    }
  };

  // Editor saved/deleted a period: paint the new range and un-paint the old one
  // immediately, then watch the backend recalculation replace the overlay.
  const onEditorSaved = (info: PeriodSaveInfo) => {
    const paint = new Set(info.saved ? isoRange(info.saved.start, info.saved.end) : []);
    const clear = new Set(
      (info.cleared ? isoRange(info.cleared.start, info.cleared.end) : []).filter((d) => !paint.has(d)),
    );
    setOverlay({
      paint,
      clear,
      confirmIso: info.saved?.start ?? info.cleared?.start ?? toApiDate(today()),
      expectPeriod: !!info.saved,
      since: dataFreshness,
    });
    setWatching(true);
  };

  // Open the editor on a specific logged period (from the day panel).
  const editPeriod = (period: LoggedPeriod) => {
    setEditingPeriod(period);
    setEditorOpen(true);
  };

  // ── Swipe between months (RTL puts "next" on the left) ───────
  const touchStart = useRef<{ x: number; y: number } | null>(null);
  const swiped = useRef(false);

  const onTouchStart = (e: React.TouchEvent) => {
    const p = e.touches[0];
    touchStart.current = p ? { x: p.clientX, y: p.clientY } : null;
    swiped.current = false;
  };
  const onTouchMove = (e: React.TouchEvent) => {
    if (!touchStart.current) return;
    const p = e.touches[0];
    if (!p) return;
    const dx = p.clientX - touchStart.current.x;
    const dy = p.clientY - touchStart.current.y;
    if (Math.abs(dx) > SWIPE_THRESHOLD_PX && Math.abs(dx) > Math.abs(dy)) swiped.current = true;
  };
  const onTouchEnd = (e: React.TouchEvent) => {
    const start = touchStart.current;
    touchStart.current = null;
    if (!start) return;
    const p = e.changedTouches[0];
    if (!p) return;
    const dx = p.clientX - start.x;
    const dy = p.clientY - start.y;
    if (Math.abs(dx) > SWIPE_THRESHOLD_PX && Math.abs(dx) > Math.abs(dy)) {
      // dx<0 → swipe left → the "left" chevron's month (next in RTL, prev in LTR).
      const nextIsLeft = isRtl;
      if (dx < 0) goMonth(nextIsLeft ? 1 : -1);
      else goMonth(nextIsLeft ? -1 : 1);
    }
  };

  // Swallow the day tap that ends a swipe gesture.
  const onDaySelect = (date: Date) => {
    if (swiped.current) {
      swiped.current = false;
      return;
    }
    selectDay(date);
  };

  const selectedCalc = calcMap.get(toApiDate(selectedDate));

  // Roll back an optimistic overlay and surface the backend's rejection message
  // (e.g. "close the previous period first", "overlaps another period").
  const onMutationError = (error: unknown) => {
    setOverlay(null);
    setWatching(false);
    setActionError(getApiErrorMessage(error) ?? t('actionFailed'));
  };

  // Auto-dismiss the error banner a few seconds after it appears.
  useEffect(() => {
    if (!actionError) return;
    const id = setTimeout(() => setActionError(null), 5000);
    return () => clearTimeout(id);
  }, [actionError]);

  const selectedMarker = markerFor(selectedDate);
  const selectedLoggedPeriod = loggedPeriodFor(selectedDate);
  const selectedExtendTarget = selectedLoggedPeriod ? null : extendTargetFor(selectedDate);
  // Unmarking an interior day of a period truncates it to end the day before — it
  // removes this day AND every later one (a period is contiguous), so the button
  // says "end period here" instead of the misleading "not a period day" (§ honesty).
  const removeDayIsInterior =
    selectedLoggedPeriod != null &&
    (() => {
      const iso = toApiDate(selectedDate);
      return iso !== selectedLoggedPeriod.period_start_date && iso !== effectiveEndOf(selectedLoggedPeriod);
    })();
  // "Start period" only where no logged period is nearby — a day close to one
  // extends it instead, so two periods can't be started days apart. A *predicted*
  // period day still offers it: the prediction is exactly what the user confirms
  // or corrects, so gating on the pink marker made the predicted start day — the
  // one day the action matters most — the only day it was missing.
  const canStartHere =
    diffInDays(selectedDate, today()) <= 0 &&
    !isActualPeriodDay(selectedDate) &&
    !selectedLoggedPeriod &&
    !selectedExtendTarget &&
    gapToNearestLoggedPeriod(selectedDate) >= NEW_PERIOD_MIN_GAP_DAYS &&
    // Never offer it while the history is still loading — a stale/empty list
    // is exactly how duplicate periods were created.
    !historyQuery.isPending;
  const mutating = updatePeriod.isPending || deletePeriod.isPending;

  // The month grid is being (re)loaded — first visit, a stale refetch after
  // coming back, or the backend recalculating after a period change. The
  // animated overlay covers the card until fresh cells are ready.
  const calendarLoading =
    monthA.isPending || monthB.isPending || monthA.isFetching || monthB.isFetching || isRecalculating;

  // A month entirely in the future has no days the user could have bled on, so
  // the "edit period date" button is disabled there.
  const currentJalali = todayJalali();
  const isFutureMonth =
    year > currentJalali.year || (year === currentJalali.year && month > currentJalali.month);

  return (
    <div className="view cal-page">
      <div className="home-grad cal-grad" />

      <div className="scroll page-scroll" ref={scrollRef} onScroll={onPageScroll}>
        <div className="cal-hdr">
          <div className="cal-hdr-txt">
            <div className="titr">{t('title')}</div>
            <p className="sub cal-hdr-sub">{t('subtitle')}</p>
          </div>
          {/* Full-screen "Edit Period Date" toggle editor — tap days on/off to mark
              the bleeding range, saved as one period (§ contiguous range). */}
          <button
            className="chip on cal-edit-btn"
            onClick={() => setDateEditorOpen(true)}
            disabled={isFutureMonth}
          >
            <Icon name="pencil" size={14} />
            {tLogPeriod('dateEditor.open')}
          </button>
        </div>

        {isRecalculating && (
          <div className="cal-note">
            <div className="cal-note-box">
              <Icon name="sparkle" size={14} fill="currentColor" strokeWidth={0} />
              {t('recalculating')}
            </div>
          </div>
        )}

        <div className="sec-tight">
          <div className="card cal-grid-card">
            {calendarLoading && <CalendarLoader label={t('loadingCalendar')} />}
            <MonthNav
              label={formatJalaliMonthLabel(year, month, locale)}
              isRtl={isRtl}
              onPrev={() => goMonth(-1)}
              onNext={() => goMonth(1)}
              onJump={() => setPickerOpen(true)}
              fullMonth={fullMonth}
              onToggle={() => setFullMonth((v) => !v)}
              toggleLabel={fullMonth ? t('weekView') : t('fullMonth')}
              jumpLabel={t('jumpToMonth')}
              prevMonthLabel={t('prevMonth')}
              nextMonthLabel={t('nextMonth')}
            />

            <div className="cal-grid cal-legend-wrap">
              {WEEKDAY_KEYS.map((k) => (
                <span key={k} className="cal-weekday">
                  {t(`weekdays.${k}`)}
                </span>
              ))}
            </div>

            <div
              onTouchStart={onTouchStart}
              onTouchMove={onTouchMove}
              onTouchEnd={onTouchEnd}
              className="cal-weeks"
              // Folded → one row tall; open → every week row (count varies 5–6).
              style={{ height: (collapsed ? 1 : weeks.length) * WEEK_ROW_PX - 4 }}
            >
              <div
                className="cal-log-list"
                // Slides the whole month up so the active week lands on the
                // strip's single visible row while the others fold away.
                style={{ transform: collapsed ? `translateY(-${activeWeekIndex * WEEK_ROW_PX}px)` : 'none' }}
              >
                {weeks.map((week, wi) => (
                  <div
                    key={wi}
                    className={clsx('cal-grid', 'cal-week-row', collapsed && wi !== activeWeekIndex && 'is-off')}
                    aria-hidden={collapsed && wi !== activeWeekIndex}
                  >
                    {week.map((cell, ci) => (
                      <DayCell
                        key={ci}
                        cell={cell}
                        selectedDate={selectedDate}
                        onSelect={onDaySelect}
                        dayNumber={cell ? format.number(cell.day) : ''}
                        marker={cell ? markerFor(cell.date) : null}
                        isLogged={cell ? isActualPeriodDay(cell.date) : false}
                        ariaLabel={cell ? dayAriaLabel(cell.date) : ''}
                      />
                    ))}
                  </div>
                ))}
              </div>
            </div>

            <div className={clsx('cal-extras', collapsed && 'is-collapsed')}>
              <div className="cal-extras-inner">
                <div className="cal-rule" />
                <Legend t={t} />
              </div>
            </div>
          </div>
        </div>

        {/* Selected-day panel — everything logged/predicted for the tapped day,
            inline under the grid (no sheet) and split into labelled groups. */}
        <div className="sec-tight">
          <div className="card pad-card cal-day-panel" aria-live="polite">
            <section className="cal-day-group">
              {/* Tiles are the fallback only once the day query has settled with no rich
                  card — they don't flash in and get swapped out while it's still loading. */}
              <DayDetail t={t} locale={locale} selectedDate={selectedDate} calc={selectedCalc} marker={selectedMarker} />
            </section>

            {/* Everything that mutates this day's period range, kept in one group so
                the actions read as a set rather than a stack of loose buttons. */}
            {(selectedLoggedPeriod || selectedExtendTarget || canStartHere) && (
              <section className="cal-day-group">
                <div className="cal-day-group-label">{t('day.actionsLabel')}</div>

                {/* This day belongs to a logged period → edit the range, or unmark
                    just this day with one tap. */}
                {selectedLoggedPeriod && (
                  <div className="cal-actions">
                    <button
                      className="btn cal-action is-primary"
                      onClick={() => editPeriod(selectedLoggedPeriod)}
                    >
                      <Icon name="drop" size={16} fill="currentColor" strokeWidth={0} />
                      {t('editThisPeriod')}
                    </button>
                    <button
                      className="btn cal-action is-neutral"
                      onClick={() => removeDayHere(selectedLoggedPeriod)}
                      disabled={mutating}
                    >
                      <Icon name="x" size={16} />
                      {removeDayIsInterior ? t('endPeriodHere') : t('removeThisDay')}
                    </button>
                  </div>
                )}

                {/* This day sits just outside a logged period → extend it to here.
                    Demoted to a soft style when the card already shows a filled
                    primary, so the panel never has two competing pink buttons. */}
                {selectedExtendTarget && (
                  <button
                    className="btn btn-primary cal-day-btn"
                    onClick={() => addToPeriodHere(selectedExtendTarget)}
                    disabled={mutating}
                  >
                    <Icon name="drop" size={16} fill="var(--on-accent)" strokeWidth={0} />
                    {t('addToPeriod')}
                  </button>
                )}

                {/* Quick "this is where my period started" action → fills cells +
                    regenerates every downstream prediction. */}
                {canStartHere && (
                  <button
                    className="btn btn-primary cal-start"
                    onClick={startPeriodHere}
                    disabled={startPeriod.isPending}
                  >
                    <Icon name="drop" size={16} fill="var(--on-accent)" strokeWidth={0} />
                    {t('startPeriodHere')}
                  </button>
                )}
              </section>
            )}

            {/* Future days can't have logged data, so hide the log section entirely. */}
            {diffInDays(selectedDate, today()) <= 0 && (
              <section className="cal-day-group">
                <div className="cal-day-group-label">{t('day.logLabel')}</div>
                <DayLogSummary tCal={t} selectedDate={selectedDate} onEdit={openLog} />
              </section>
            )}
          </div>
        </div>

        <div
          className="page-tail cal-tail"
          style={{ height: PAGE_TAIL_PX + (fullMonth && scrollCollapsed ? tailPad : 0) }}
        />
      </div>

      {/* Quick month/year jump — tap the month label to open. */}
      <MonthYearPicker
        t={t}
        locale={locale}
        isRtl={isRtl}
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        year={year}
        month={month}
        onPick={pickMonth}
        onToday={goToday}
      />

      <PeriodEditor
        open={editorOpen}
        onClose={() => { setEditorOpen(false); setEditingPeriod(null); }}
        initialDateIso={toApiDate(selectedDate)}
        editing={editingPeriod}
        onSaved={onEditorSaved}
      />

      {/* Full-screen toggle editor opened from the header. Opens on the month the
          calendar is showing, pre-fills every logged period, and reconciles on save. */}
      <PeriodDateEditor
        open={dateEditorOpen}
        onClose={() => setDateEditorOpen(false)}
        initialView={{ year, month }}
        onSaved={() => setWatching(true)}
      />

      {/* Transient validation feedback (spec §16): a period edit the backend rejected
          — surfaced instead of a silent rollback, auto-dismissed after a few seconds. */}
      {actionError && (
        <div
          role="alert"
          className="cal-toast"
          onClick={() => setActionError(null)}
        >
          {actionError}
        </div>
      )}

      <BottomNav />
    </div>
  );
}
