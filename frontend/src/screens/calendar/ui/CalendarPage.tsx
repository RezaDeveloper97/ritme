'use client';

import { useFormatter, useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useRef, useState } from 'react';

import {
  calcToPhase,
  cycleDayMarker,
  normalizePhase,
  useCycleMonth,
  useCycleStatus,
  type CycleCalculation,
  type CycleDayMarker,
  type CyclePhase,
} from '@/entities/cycle';
import { useUserProfile } from '@/entities/user';
import {
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
import { Icon, Sheet } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { DayLogSummary } from './DayLogSummary';

const WEEKDAY_KEYS = ['sat', 'sun', 'mon', 'tue', 'wed', 'thu', 'fri'] as const;

// Colors for each calendar marker. Follicular/luteal days carry no marker and
// read as neutral. Kept in sync with the legend below.
const MARKER_STYLE: Record<CycleDayMarker, { bg: string; color: string }> = {
  period: { bg: '#FCE7F3', color: '#E91E63' },
  fertile: { bg: '#FEF3C6', color: '#F5A623' },
  ovulation: { bg: '#E7F8EF', color: '#22B07D' },
  pms: { bg: '#EDE9FE', color: '#8B5CF6' },
};

const NEUTRAL_STYLE = { bg: '#F3F0FF', color: '#7C7CF0' };

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
}

function MonthNav({ label, isRtl, onPrev, onNext, onJump, fullMonth, onToggle, toggleLabel, jumpLabel }: MonthNavProps) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '4px 4px 14px' }}>
      {/* First child → RIGHT in RTL = previous month */}
      <button className="iconbtn" onClick={isRtl ? onPrev : onNext} aria-label={isRtl ? 'ماه قبل' : 'Next month'}>
        <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
      </button>

      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        {/* Tapping the label jumps back to the current month. */}
        <button
          onClick={onJump}
          aria-label={jumpLabel}
          style={{ fontSize: 16, fontWeight: 800, color: 'var(--ink)', background: 'none', border: 'none', cursor: 'pointer', fontFamily: 'inherit', display: 'inline-flex', alignItems: 'center', gap: 4 }}
        >
          {label}
          <Icon name="chevronDown" size={14} />
        </button>
        <button
          onClick={onToggle}
          className={`chip${fullMonth ? ' on' : ''}`}
          style={{ padding: '5px 12px', fontSize: 12 }}
        >
          <Icon name={fullMonth ? 'grid' : 'calendar'} size={13} />
          {toggleLabel}
        </button>
      </div>

      {/* Last child → LEFT in RTL = next month */}
      <button className="iconbtn" onClick={isRtl ? onNext : onPrev} aria-label={isRtl ? 'ماه بعد' : 'Previous month'}>
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
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
        {/* First child → RIGHT in RTL = previous year */}
        <button
          className="iconbtn"
          onClick={() => setBrowseYear((y) => y + (isRtl ? -1 : 1))}
          aria-label={isRtl ? 'سال قبل' : 'Next year'}
        >
          <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
        </button>
        <div id="month-picker-title" style={{ fontSize: 17, fontWeight: 800, color: 'var(--ink)', fontVariantNumeric: 'tabular-nums' }}>
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

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 8 }}>
        {Array.from({ length: 12 }, (_, i) => i + 1).map((m) => {
          const isCurrent = m === month && browseYear === year;
          return (
            <button
              key={m}
              className={`chip${isCurrent ? ' on' : ''}`}
              onClick={() => onPick(browseYear, m)}
              style={{ justifyContent: 'center', padding: '10px 0', fontSize: 13, fontWeight: 700, borderRadius: 12 }}
            >
              {jalaliMonthName(m, locale)}
            </button>
          );
        })}
      </div>

      <button className="btn" onClick={onToday} style={{ borderRadius: 14, marginTop: 14, fontWeight: 800 }}>
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
}

function DayCell({ cell, selectedDate, onSelect, dayNumber, marker }: DayCellProps) {
  if (!cell) return <span />;

  const markerStyle = marker ? MARKER_STYLE[marker] : undefined;
  const selected = isSameDay(cell.date, selectedDate);
  const isToday = isSameDay(cell.date, today());
  // A period day still in the future is a prediction → hollow ring, not fill.
  const isPredicted = marker === 'period' && diffInDays(cell.date, today()) > 0;

  return (
    <button
      onClick={() => onSelect(cell.date)}
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
      {dayNumber}
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

// ── Legend ─────────────────────────────────────────────────────
function Legend({ t }: { t: T }) {
  const items: { key: CycleDayMarker }[] = [
    { key: 'period' },
    { key: 'fertile' },
    { key: 'ovulation' },
    { key: 'pms' },
  ];
  return (
    <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', padding: '4px 4px 0' }}>
      {items.map((it) => (
        <span key={it.key} style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
          <span style={{ width: 10, height: 10, borderRadius: '50%', background: MARKER_STYLE[it.key].color }} />
          <span style={{ fontSize: 11.5, fontWeight: 600, color: 'var(--muted)' }}>{t(`legend.${it.key}`)}</span>
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
}

function DayDetail({ t, locale, selectedDate, calc, marker }: DayDetailProps) {
  const phase = calc ? calcToPhase(calc) : undefined;
  const style = (marker && MARKER_STYLE[marker]) ?? NEUTRAL_STYLE;
  const isToday = isSameDay(selectedDate, today());
  // Prefer the marker for the label (so PMS/fertile read as themselves) and fall
  // back to the underlying phase.
  const labelKey = marker ?? phase;

  return (
    <div style={{ padding: '4px 2px 2px' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <span className="dot" style={{ width: 42, height: 42, background: style.bg, color: style.color }}>
            <Icon name="drop" size={20} fill="currentColor" strokeWidth={0} />
          </span>
          <div style={{ textAlign: 'start' }}>
            <div id="day-sheet-title" style={{ fontSize: 15, fontWeight: 800, color: 'var(--ink)' }}>
              {formatJalaliDayMonth(selectedDate, locale)}
            </div>
            {calc && (
              <div style={{ fontSize: 12, color: 'var(--muted)', marginTop: 2 }}>
                {t('day.cycleDay', { n: calc.cycleDay })}
              </div>
            )}
          </div>
        </div>
        {isToday && (
          <span style={{ fontSize: 11, fontWeight: 700, color: 'var(--brand)', background: '#FFF1F7', borderRadius: 20, padding: '4px 12px' }}>
            {t('today')}
          </span>
        )}
      </div>

      {phase && labelKey && (
        <div style={{ display: 'flex', gap: 10 }}>
          <div style={{ flex: 1, background: style.bg, borderRadius: 12, padding: '10px 12px', textAlign: 'start' }}>
            <div style={{ fontSize: 11, color: 'var(--muted)', fontWeight: 600 }}>{t('phaseLabel')}</div>
            <div style={{ fontSize: 13, fontWeight: 800, color: style.color, marginTop: 3 }}>{t(`phase.${labelKey}`)}</div>
          </div>
          <div style={{ flex: 1, background: 'var(--line)', borderRadius: 12, padding: '10px 12px', textAlign: 'start' }}>
            <div style={{ fontSize: 11, color: 'var(--muted)', fontWeight: 600 }}>{t('day.chanceLabel')}</div>
            <div style={{ fontSize: 13, fontWeight: 800, color: 'var(--steel)', marginTop: 3 }}>{t(`chance.${chanceKey(phase)}`)}</div>
          </div>
        </div>
      )}
    </div>
  );
}

// ── Smart tip (educational, non-diagnostic — §11) ──────────────
function SmartTip({ t }: { t: T }) {
  return (
    <div className="card" style={{ padding: '14px 12px' }}>
      <div style={{ fontSize: 14, fontWeight: 800, color: 'var(--ink)', textAlign: 'start', marginBottom: 10 }}>
        {t('smartTip.title')}
      </div>
      <p className="sub" style={{ textAlign: 'start', margin: '0 2px 14px' }}>{t('smartTip.body')}</p>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, background: 'linear-gradient(90deg,#FFF0F7,#F3F0FF)', borderRadius: 12, padding: 12 }}>
        <span style={{ color: 'var(--ritme-pink)' }}>
          <Icon name="sparkle" size={20} fill="currentColor" strokeWidth={0} />
        </span>
        <span style={{ flex: 1, textAlign: 'start', fontSize: 12.5, fontWeight: 700, color: 'var(--ink)' }}>
          {t('smartTip.quote')}
        </span>
      </div>
    </div>
  );
}

// ── Main export ────────────────────────────────────────────────
export function CalendarPage() {
  const t = useTranslations('calendar');
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
  const [daySheetOpen, setDaySheetOpen] = useState(false);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  // Watch the backend recalculation only right after we trigger one.
  const [watching, setWatching] = useState(false);
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

  const displayWeeks = useMemo(() => {
    if (fullMonth) return weeks;
    const withSelected = weeks.find((w) => w.some((c) => c && isSameDay(c.date, selectedDate)));
    return [withSelected ?? weeks[0]];
  }, [weeks, fullMonth, selectedDate]);

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
    setDaySheetOpen(true);
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
    setDaySheetOpen(false);
    setWatching(true);
    startPeriod.mutate(
      { date: iso },
      // Roll back the optimistic fill if the re-anchor fails.
      { onError: () => { setOverlay(null); setWatching(false); } },
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
    setDaySheetOpen(false);
    setWatching(true);
    updatePeriod.mutate(
      { id: target.period.id, start: target.newStart, end: target.newEnd },
      { onError: () => { setOverlay(null); setWatching(false); } },
    );
  };

  // Unmark the tapped day of a logged period. Removing the first day moves the
  // start forward; any other day truncates the period to the day before it
  // (days are contiguous, so "this day wasn't period" ends the bleed there);
  // a single-day period is deleted outright.
  const removeDayHere = (period: LoggedPeriod) => {
    const iso = toApiDate(selectedDate);
    const effEnd = effectiveEndOf(period);
    const rollback = { onError: () => { setOverlay(null); setWatching(false); } };
    const overlayFor = (cleared: string[]) => ({
      paint: new Set<string>(),
      clear: new Set(cleared),
      confirmIso: iso,
      expectPeriod: false,
      since: dataFreshness,
    });

    setDaySheetOpen(false);
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

  // Open the editor on a specific logged period (from the day sheet).
  const editPeriod = (period: LoggedPeriod) => {
    setEditingPeriod(period);
    setDaySheetOpen(false);
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
  const selectedMarker = markerFor(selectedDate);
  const selectedLoggedPeriod = loggedPeriodFor(selectedDate);
  const selectedExtendTarget = selectedLoggedPeriod ? null : extendTargetFor(selectedDate);
  // "Start period" only where no logged period is nearby — a day close to one
  // extends it instead, so two periods can't be started days apart.
  const canStartHere =
    diffInDays(selectedDate, today()) <= 0 &&
    selectedMarker !== 'period' &&
    !selectedLoggedPeriod &&
    !selectedExtendTarget &&
    gapToNearestLoggedPeriod(selectedDate) >= NEW_PERIOD_MIN_GAP_DAYS &&
    // Never offer it while the history is still loading — a stale/empty list
    // is exactly how duplicate periods were created.
    !historyQuery.isPending;
  const mutating = updatePeriod.isPending || deletePeriod.isPending;

  return (
    <div className="view" style={{ background: 'var(--page)' }}>
      <div className="home-grad" style={{ position: 'absolute', top: 0, insetInlineStart: 0, insetInlineEnd: 0, height: 260 }} />

      <div className="scroll" style={{ position: 'relative', zIndex: 1 }}>
        <div style={{ padding: '6px 20px 0', display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 12 }}>
          <div style={{ textAlign: 'start' }}>
            <div className="titr">{t('title')}</div>
            <p className="sub" style={{ margin: '6px 0 0' }}>{t('subtitle')}</p>
          </div>
          <button
            className="chip on"
            onClick={() => setEditorOpen(true)}
            style={{ padding: '8px 14px', fontSize: 13, gap: 6, flexShrink: 0, marginTop: 2 }}
          >
            <Icon name="drop" size={14} fill="currentColor" strokeWidth={0} />
            {t('logPeriodCta')}
          </button>
        </div>

        {isRecalculating && (
          <div style={{ padding: '10px 16px 0' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, background: '#FFF1F7', color: 'var(--brand)', borderRadius: 12, padding: '8px 12px', fontSize: 12.5, fontWeight: 700 }}>
              <Icon name="sparkle" size={14} fill="currentColor" strokeWidth={0} />
              {t('recalculating')}
            </div>
          </div>
        )}

        <div style={{ padding: '14px 16px 0' }}>
          <div className="card" style={{ padding: '14px 14px 16px' }}>
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
            />

            <div className="cal-grid" style={{ marginBottom: 6 }}>
              {WEEKDAY_KEYS.map((k) => (
                <span key={k} style={{ fontSize: 11, color: 'var(--muted)', fontWeight: 700, textAlign: 'center' }}>
                  {t(`weekdays.${k}`)}
                </span>
              ))}
            </div>

            <div
              onTouchStart={onTouchStart}
              onTouchMove={onTouchMove}
              onTouchEnd={onTouchEnd}
              style={{ display: 'flex', flexDirection: 'column', gap: 4 }}
            >
              {displayWeeks.map((week, wi) => (
                <div key={wi} className="cal-grid">
                  {week.map((cell, ci) => (
                    <DayCell
                      key={ci}
                      cell={cell}
                      selectedDate={selectedDate}
                      onSelect={onDaySelect}
                      dayNumber={cell ? format.number(cell.day) : ''}
                      marker={cell ? markerFor(cell.date) : null}
                    />
                  ))}
                </div>
              ))}
            </div>

            <div style={{ height: 1, background: 'var(--line)', margin: '14px 0' }} />
            <Legend t={t} />
          </div>
        </div>

        <div style={{ padding: '14px 16px 0' }}>
          <SmartTip t={t} />
        </div>

        <div style={{ height: 26 }} />
      </div>

      {/* Day detail sheet — everything logged/predicted for the tapped day. */}
      <Sheet open={daySheetOpen} onClose={() => setDaySheetOpen(false)} labelledBy="day-sheet-title">
        <DayDetail t={t} locale={locale} selectedDate={selectedDate} calc={selectedCalc} marker={selectedMarker} />

        {/* This day belongs to a logged period → edit the range, or unmark
            just this day with one tap. */}
        {selectedLoggedPeriod && (
          <div style={{ display: 'flex', gap: 10, marginTop: 14 }}>
            <button
              className="btn"
              onClick={() => editPeriod(selectedLoggedPeriod)}
              style={{ flex: 1, borderRadius: 14, gap: 8, background: '#FCE7F3', color: '#E91E63', fontWeight: 800 }}
            >
              <Icon name="drop" size={16} fill="currentColor" strokeWidth={0} />
              {t('editThisPeriod')}
            </button>
            <button
              className="btn"
              onClick={() => removeDayHere(selectedLoggedPeriod)}
              disabled={mutating}
              style={{ flex: 1, borderRadius: 14, gap: 8, background: 'var(--line)', color: 'var(--steel)', fontWeight: 800 }}
            >
              <Icon name="x" size={16} />
              {t('removeThisDay')}
            </button>
          </div>
        )}

        {/* This day sits just outside a logged period → extend it to here. */}
        {selectedExtendTarget && (
          <button
            className="btn btn-primary"
            onClick={() => addToPeriodHere(selectedExtendTarget)}
            disabled={mutating}
            style={{ borderRadius: 14, marginTop: 14, gap: 8 }}
          >
            <Icon name="drop" size={16} fill="#fff" strokeWidth={0} />
            {t('addToPeriod')}
          </button>
        )}

        {/* Quick "this is where my period started" action → fills cells + regenerates. */}
        {canStartHere && (
          <button
            className="btn btn-primary"
            onClick={startPeriodHere}
            disabled={startPeriod.isPending}
            style={{ borderRadius: 14, marginTop: 14, gap: 8 }}
          >
            <Icon name="drop" size={16} fill="#fff" strokeWidth={0} />
            {t('startPeriodHere')}
          </button>
        )}

        {/* Future days can't have logged data, so hide the log section entirely. */}
        {diffInDays(selectedDate, today()) <= 0 && (
          <div style={{ marginTop: 14 }}>
            <DayLogSummary tCal={t} selectedDate={selectedDate} onEdit={openLog} />
          </div>
        )}
      </Sheet>

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

      <BottomNav />
    </div>
  );
}
