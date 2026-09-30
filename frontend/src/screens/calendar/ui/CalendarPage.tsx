'use client';

import { useFormatter, useLocale, useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import { useEffect, useMemo, useRef, useState } from 'react';

import {
  cycleDayMarkerAt,
  deriveCycleSchedule,
  markerPhase,
  normalizePhase,
  useCycleStatus,
  useCycleToday,
  type CycleCalculation,
  type CycleDayMarker,
} from '@/entities/cycle';
import { useUserMode } from '@/entities/message';
import { useUserProfile } from '@/entities/user';
import {
  PeriodDateEditor,
  useDeletePeriod,
  usePeriodHistory,
  useStartPeriod,
  useUpdatePeriod,
  type LoggedPeriod,
} from '@/features/log-period';
import { useDirection, useRouter, type Locale } from '@/shared/i18n';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  addDays,
  diffInDays,
  formatDayMonth,
  formatMonthLabel,
  fromApiDate,
  monthMatrix,
  monthName,
  shiftMonth,
  toApiDate,
  toParts,
  today,
  todayParts,
  weekdayKeys,
} from '@/shared/lib/date';
import { getApiErrorMessage } from '@/shared/api';
import {
  EmptyState,
  HeaderButton,
  Icon,
  PrimaryButton,
  ScreenHeader,
  SegmentedTabs,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';
import {
  CalendarLegend,
  CYCLE_LEGEND,
  MonthCard,
  TTC_LEGEND,
  YearView,
  dayTone,
  type CalendarDayInfo,
  type LegendKey,
} from '@/widgets/cycle-calendar';

import { useCalcMap } from '../model/useCalcMap';
import { DayCard } from './DayCard';

// Fallback bleeding length when the profile hasn't recorded one (backend default).
const DEFAULT_PERIOD_DAYS = 5;
// Horizontal drag past this many px switches month (mobile swipe).
const SWIPE_THRESHOLD_PX = 40;
// A day this close to a logged period extends it instead of starting a new one.
const EXTEND_GAP_DAYS = 7;
// Quick «start period» only a rough cycle away from every logged period.
const NEW_PERIOD_MIN_GAP_DAYS = 21;

type View = 'month' | 'year';

const isSameDay = (a: Date, b: Date) => diffInDays(a, b) === 0;

/** Every ISO day in the inclusive [start, end] range. */
function isoRange(start: string, end: string): string[] {
  const days: string[] = [];
  const first = fromApiDate(start);
  const count = diffInDays(fromApiDate(end), first) + 1;
  for (let i = 0; i < count; i += 1) days.push(toApiDate(addDays(first, i)));
  return days;
}

/** First and last real day of a locale month. */
function monthBounds(year: number, month: number, locale: Locale): { first: Date; last: Date } {
  const cells = monthMatrix(year, month, locale).flat().filter((c) => c !== null);
  return { first: cells[0]!.date, last: cells[cells.length - 1]!.date };
}

/**
 * Optimistic paint layer: `paint` renders as a logged period at once, `clear`
 * suppresses stale period cells (after an edit/delete), until the refetched
 * data confirms the change at `confirmIso`.
 */
interface Overlay {
  paint: Set<string>;
  clear: Set<string>;
  confirmIso: string;
  expectPeriod: boolean;
  since: number;
}

/**
 * Night & Bloom calendar (`Cycle_Calendar`, TTC `TTC_Calendar`): month ⇄ year
 * tabs, two stacked months with banded tones, legend, and the selected day's
 * card. The period range actions of the old calendar live on in the day card.
 */
export function CalendarPage() {
  const t = useTranslations('calendar');
  const tn = useTranslations('calendar.nb');
  const tLogPeriod = useTranslations('logPeriod');
  const locale = useLocale() as Locale;
  const format = useFormatter();
  const router = useRouter();
  const isRtl = useDirection() === 'rtl';
  const modeQuery = useUserMode();
  const isTtc = modeQuery.data?.isTtc ?? false;
  // Queries are gated on the token (client-only): hold the skeleton until mount so
  // the first client render matches the server HTML, and until the mode is known
  // so a TTC user never sees the cycle layout flash first.
  const mounted = useMounted();

  const [view, setView] = useState<View>('month');
  const [anchor, setAnchor] = useState(() => {
    const j = todayParts(locale);
    return { year: j.year, month: j.month };
  });
  const [yearShown, setYearShown] = useState(() => todayParts(locale).year);
  const [selectedDate, setSelectedDate] = useState<Date>(() => today());
  const [dateEditorOpen, setDateEditorOpen] = useState(false);
  const [dateEditorView, setDateEditorView] = useState<{ year: number; month: number } | null>(null);
  const searchParams = useSearchParams();
  useEffect(() => {
    if (searchParams.get('editDates') === '1') setDateEditorOpen(true);
    // `?view=year` deep-links the year view (QA screenshots, links from analysis).
    if (searchParams.get('view') === 'year') setView('year');
  }, [searchParams]);
  const [watching, setWatching] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [overlay, setOverlay] = useState<Overlay | null>(null);

  const profileQuery = useUserProfile();
  const periodDuration = profileQuery.data?.health?.periodDuration ?? DEFAULT_PERIOD_DAYS;
  const startPeriod = useStartPeriod();
  const updatePeriod = useUpdatePeriod();
  const deletePeriod = useDeletePeriod();
  const historyQuery = usePeriodHistory();
  const status = useCycleStatus({ poll: watching, enabled: watching });
  const isRecalculating = watching && (status.data?.is_processing ?? false);
  useEffect(() => {
    if (watching && status.data && !status.data.is_processing) setWatching(false);
  }, [watching, status.data]);

  // The two stacked months: the cycle artboard puts the anchor month first and the
  // previous one under it; the TTC artboard reads chronologically (QUESTIONS #B-N1-07).
  const prev = shiftMonth(anchor.year, anchor.month, -1);
  const months = isTtc ? [prev, anchor] : [anchor, prev];

  const range =
    view === 'year'
      ? { first: monthBounds(yearShown, 1, locale).first, last: monthBounds(yearShown, 12, locale).last }
      : { first: monthBounds(prev.year, prev.month, locale).first, last: monthBounds(anchor.year, anchor.month, locale).last };
  const calc = useCalcMap(range.first, range.last);
  const calcMap = calc.map;
  // Skeleton only on the first load; later month changes keep the cards on screen
  // (the status pill says they are refreshing) instead of flashing a skeleton.
  const [loadedOnce, setLoadedOnce] = useState(false);
  useEffect(() => {
    if (mounted && !calc.pending) setLoadedOnce(true);
  }, [mounted, calc.pending]);
  const loading =
    !mounted ||
    (!loadedOnce && calc.pending) ||
    (modeQuery.isPending && modeQuery.fetchStatus !== 'idle');

  useEffect(() => {
    if (!overlay) return;
    const c = calcMap.get(overlay.confirmIso);
    const isPeriod = c ? normalizePhase(c.phase) === 'period' : false;
    if (isPeriod === overlay.expectPeriod) setOverlay(null);
    else if (!watching && calc.freshness > overlay.since) setOverlay(null);
  }, [overlay, calcMap, watching, calc.freshness]);

  // Anchored engine schedule (same as home) decides fertile window / ovulation.
  const todayCycle = useCycleToday();
  const schedule = useMemo(
    () => deriveCycleSchedule(todayCycle.data?.cycleView ?? null, todayCycle.data?.calculation ?? null),
    [todayCycle.data],
  );
  const dayMarker = useMemo(
    () => (c: CycleCalculation) =>
      cycleDayMarkerAt(fromApiDate(c.calculationDate), c, schedule, schedule?.periodLength ?? periodDuration),
    [schedule, periodDuration],
  );

  const markerFor = (date: Date): CycleDayMarker | null => {
    const iso = toApiDate(date);
    if (overlay?.paint.has(iso)) return 'period';
    if (overlay?.clear.has(iso)) return null;
    const c = calcMap.get(iso);
    return c ? dayMarker(c) : null;
  };

  const effectiveEndOf = (p: LoggedPeriod): string =>
    p.period_end_date ?? toApiDate(addDays(fromApiDate(p.period_start_date), periodDuration - 1));

  // Last day of a period that has actually happened (edits never reach the future).
  const realEndOf = (p: LoggedPeriod): string => {
    const end = effectiveEndOf(p);
    const todayIso = toApiDate(today());
    return end > todayIso ? todayIso : end;
  };

  const loggedPeriodFor = (date: Date): LoggedPeriod | null => {
    const iso = toApiDate(date);
    for (const p of historyQuery.data ?? []) {
      if (iso >= p.period_start_date && iso <= effectiveEndOf(p)) return p;
    }
    return null;
  };

  const isActualPeriodDay = (date: Date): boolean => {
    const iso = toApiDate(date);
    if (overlay?.paint.has(iso)) return true;
    if (overlay?.clear.has(iso)) return false;
    return loggedPeriodFor(date) !== null;
  };

  const extendTargetFor = (date: Date): { period: LoggedPeriod; newStart: string; newEnd: string | null } | null => {
    if (diffInDays(date, today()) > 0) return null;
    const iso = toApiDate(date);
    let best: { period: LoggedPeriod; newStart: string; newEnd: string | null; gap: number } | null = null;
    for (const p of historyQuery.data ?? []) {
      const end = effectiveEndOf(p);
      let candidate: { newStart: string; newEnd: string | null; gap: number } | null = null;
      if (iso < p.period_start_date) {
        const gap = diffInDays(fromApiDate(p.period_start_date), date);
        if (gap <= EXTEND_GAP_DAYS) candidate = { newStart: iso, newEnd: p.period_end_date, gap };
      } else if (iso > end) {
        const gap = diffInDays(date, fromApiDate(end));
        if (gap <= EXTEND_GAP_DAYS) candidate = { newStart: p.period_start_date, newEnd: iso, gap };
      }
      if (candidate && (!best || candidate.gap < best.gap)) best = { period: p, ...candidate };
    }
    return best && { period: best.period, newStart: best.newStart, newEnd: best.newEnd };
  };

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

  const legendLabel = (key: LegendKey): string => tn(`legend.${key}`);

  const dayInfo = (date: Date): CalendarDayInfo => {
    const marker = markerFor(date);
    const tone = dayTone(marker, isActualPeriodDay(date));
    const parts = [formatDayMonth(date, locale)];
    if (tone) parts.push(legendLabel(tone));
    if (marker === 'ovulation') parts.push(legendLabel('ovulation'));
    if (isSameDay(date, today())) parts.push(t('today'));
    return { tone, ovulation: marker === 'ovulation', label: parts.join('، ') };
  };
  const toneOf = (date: Date) => dayTone(markerFor(date), isActualPeriodDay(date));

  const onMutationError = (error: unknown) => {
    setOverlay(null);
    setWatching(false);
    setActionError(getApiErrorMessage(error) ?? t('actionFailed'));
  };
  useEffect(() => {
    if (!actionError) return;
    const id = setTimeout(() => setActionError(null), 5000);
    return () => clearTimeout(id);
  }, [actionError]);

  const startPeriodHere = () => {
    const iso = toApiDate(selectedDate);
    const end = toApiDate(addDays(selectedDate, periodDuration - 1));
    setOverlay({ paint: new Set(isoRange(iso, end)), clear: new Set(), confirmIso: iso, expectPeriod: true, since: calc.freshness });
    setWatching(true);
    startPeriod.mutate({ date: iso }, { onError: onMutationError });
  };

  const addToPeriodHere = (target: { period: LoggedPeriod; newStart: string; newEnd: string | null }) => {
    const iso = toApiDate(selectedDate);
    const paintEnd = target.newEnd ?? effectiveEndOf(target.period);
    setOverlay({ paint: new Set(isoRange(target.newStart, paintEnd)), clear: new Set(), confirmIso: iso, expectPeriod: true, since: calc.freshness });
    setWatching(true);
    updatePeriod.mutate({ id: target.period.id, start: target.newStart, end: target.newEnd }, { onError: onMutationError });
  };

  // Unmark the tapped day: first day moves the start, any other truncates the
  // period to the day before; nothing left in the past → delete it.
  const removeDayHere = (period: LoggedPeriod) => {
    const iso = toApiDate(selectedDate);
    const effEnd = effectiveEndOf(period);
    const realEnd = realEndOf(period);
    const rollback = { onError: onMutationError };
    const overlayFor = (cleared: string[]): Overlay => ({
      paint: new Set<string>(),
      clear: new Set(cleared),
      confirmIso: iso,
      expectPeriod: false,
      since: calc.freshness,
    });
    setWatching(true);
    if (iso === period.period_start_date && iso >= realEnd) {
      setOverlay(overlayFor(isoRange(iso, effEnd)));
      deletePeriod.mutate({ id: period.id }, rollback);
    } else if (iso === period.period_start_date) {
      setOverlay(overlayFor([iso]));
      updatePeriod.mutate({ id: period.id, start: toApiDate(addDays(selectedDate, 1)), end: period.period_end_date }, rollback);
    } else {
      setOverlay(overlayFor(isoRange(iso, effEnd)));
      updatePeriod.mutate({ id: period.id, start: period.period_start_date, end: toApiDate(addDays(selectedDate, -1)) }, rollback);
    }
  };

  const openEditor = (period?: LoggedPeriod) => {
    if (period) {
      const j = toParts(fromApiDate(period.period_start_date), locale);
      setDateEditorView({ year: j.year, month: j.month });
    } else {
      setDateEditorView(null);
    }
    setDateEditorOpen(true);
  };

  const goMonth = (delta: number) => setAnchor((v) => shiftMonth(v.year, v.month, delta));

  // ── Swipe between months on the card with the arrows (RTL: next is on the left)
  const touchStart = useRef<{ x: number; y: number } | null>(null);
  const swiped = useRef(false);
  const onTouchStart = (e: React.TouchEvent) => {
    const p = e.touches[0];
    touchStart.current = p ? { x: p.clientX, y: p.clientY } : null;
    swiped.current = false;
  };
  const onTouchMove = (e: React.TouchEvent) => {
    const start = touchStart.current;
    const p = e.touches[0];
    if (!start || !p) return;
    const dx = p.clientX - start.x;
    if (Math.abs(dx) > SWIPE_THRESHOLD_PX && Math.abs(dx) > Math.abs(p.clientY - start.y)) swiped.current = true;
  };
  const onTouchEnd = (e: React.TouchEvent) => {
    const start = touchStart.current;
    touchStart.current = null;
    const p = e.changedTouches[0];
    if (!start || !p) return;
    const dx = p.clientX - start.x;
    if (Math.abs(dx) > SWIPE_THRESHOLD_PX && Math.abs(dx) > Math.abs(p.clientY - start.y)) {
      goMonth(dx < 0 === isRtl ? 1 : -1);
    }
  };
  const onDaySelect = (date: Date) => {
    if (swiped.current) {
      swiped.current = false;
      return;
    }
    setSelectedDate(date);
  };

  // ── Selected day
  const selectedCalc = calcMap.get(toApiDate(selectedDate));
  const selectedMarker = markerFor(selectedDate);
  const phaseKey = selectedCalc ? (selectedMarker ?? markerPhase(selectedCalc, selectedMarker)) : null;
  const isFuture = diffInDays(selectedDate, today()) > 0;
  const selectedLoggedPeriod = loggedPeriodFor(selectedDate);
  const selectedExtendTarget = selectedLoggedPeriod ? null : extendTargetFor(selectedDate);
  const removeDayIsInterior =
    selectedLoggedPeriod != null &&
    toApiDate(selectedDate) !== selectedLoggedPeriod.period_start_date &&
    toApiDate(selectedDate) !== realEndOf(selectedLoggedPeriod);
  const canRemoveSelectedDay = selectedLoggedPeriod != null && !isFuture;
  const canStartHere =
    !isFuture &&
    !isActualPeriodDay(selectedDate) &&
    !selectedLoggedPeriod &&
    !selectedExtendTarget &&
    gapToNearestLoggedPeriod(selectedDate) >= NEW_PERIOD_MIN_GAP_DAYS &&
    !historyQuery.isPending;
  const mutating = updatePeriod.isPending || deletePeriod.isPending;

  const actions =
    selectedLoggedPeriod || selectedExtendTarget || canStartHere ? (
      <div className="cc-actions" role="group" aria-label={t('day.actionsLabel')}>
        {selectedLoggedPeriod ? (
          <button type="button" className="cc-act" onClick={() => openEditor(selectedLoggedPeriod)}>
            <Icon name="pencil" size={16} />
            {t('editThisPeriod')}
          </button>
        ) : null}
        {selectedLoggedPeriod && canRemoveSelectedDay ? (
          <button type="button" className="cc-act is-quiet" onClick={() => removeDayHere(selectedLoggedPeriod)} disabled={mutating}>
            <Icon name="x" size={16} />
            {removeDayIsInterior ? t('endPeriodHere') : t('removeThisDay')}
          </button>
        ) : null}
        {selectedExtendTarget ? (
          <button type="button" className="cc-act is-period" onClick={() => addToPeriodHere(selectedExtendTarget)} disabled={mutating}>
            <Icon name="drop" size={16} fill="currentColor" strokeWidth={0} />
            {t('addToPeriod')}
          </button>
        ) : null}
        {canStartHere ? (
          <button type="button" className="cc-act is-period" onClick={startPeriodHere} disabled={startPeriod.isPending}>
            <Icon name="drop" size={16} fill="currentColor" strokeWidth={0} />
            {t('startPeriodHere')}
          </button>
        ) : null}
      </div>
    ) : null;

  // ── Header copy
  const subtitle =
    view === 'year'
      ? tn('yearSubtitle', { year: format.number(yearShown, { useGrouping: false }) })
      : isTtc
        ? prev.year === anchor.year
          ? tn('twoMonths', { first: monthName(prev.month, locale), second: formatMonthLabel(anchor.year, anchor.month, locale) })
          : tn('twoMonths', { first: formatMonthLabel(prev.year, prev.month, locale), second: formatMonthLabel(anchor.year, anchor.month, locale) })
        : formatMonthLabel(anchor.year, anchor.month, locale);

  const weekdays = weekdayKeys(locale).map((k) => t(`weekdays.${k}`));
  const formatDay = (n: number) => format.number(n);
  const currentCal = todayParts(locale);
  const isFutureMonth = (anchor.year - currentCal.year) * 12 + (anchor.month - currentCal.month) > 1;
  const noHistory = historyQuery.isSuccess && (historyQuery.data?.length ?? 0) === 0 && !overlay;

  const body = calc.error ? (
    <div className="nb-card cc-error" role="alert">
      <span className="cc-error-disc" aria-hidden>
        <Icon name="warning" size={20} />
      </span>
      <div className="cc-error-text">
        <p className="cc-error-title">{tn('loadError')}</p>
        <p className="cc-error-body">{tn('loadErrorBody')}</p>
      </div>
      <PrimaryButton className="cc-error-retry" onClick={calc.refetch}>
        {t('retry')}
      </PrimaryButton>
    </div>
  ) : loading ? (
    <SkeletonGroup label={t('loadingCalendar')} className="cc-skel">
      <Skeleton shape="card" />
      <Skeleton shape="card" />
      <Skeleton shape="block" />
    </SkeletonGroup>
  ) : view === 'year' ? (
    <YearView
      year={yearShown}
      locale={locale}
      weekdays={weekdays}
      toneOf={toneOf}
      formatDay={formatDay}
      onPickMonth={(m) => {
        setAnchor({ year: yearShown, month: m });
        setView('month');
      }}
      onPrevYear={() => setYearShown((y) => y - 1)}
      onNextYear={() => setYearShown((y) => y + 1)}
      prevYearLabel={tn('prevYear')}
      nextYearLabel={tn('nextYear')}
      openMonthLabel={(name) => tn('openMonth', { month: name })}
    />
  ) : (
    months.map((m, i) => (
      <MonthCard
        key={`${m.year}-${m.month}`}
        year={m.year}
        month={m.month}
        locale={locale}
        weekdays={weekdays}
        dayInfo={dayInfo}
        formatDay={formatDay}
        selected={selectedDate}
        onSelect={onDaySelect}
        showOvulation={isTtc}
        nav={i === 0 ? { onPrev: () => goMonth(-1), onNext: () => goMonth(1), prevLabel: t('prevMonth'), nextLabel: t('nextMonth') } : undefined}
        onTouchStart={i === 0 ? onTouchStart : undefined}
        onTouchMove={i === 0 ? onTouchMove : undefined}
        onTouchEnd={i === 0 ? onTouchEnd : undefined}
      />
    ))
  );

  return (
    <div className="view cc-page">
      <div className="scroll cc-screen">
        <SkyLayer />
        <ScreenHeader
          title={isTtc ? tn('ttcTitle') : t('title')}
          subtitle={subtitle}
          onBack={() => router.push('/home')}
          backLabel={tn('back')}
          action={
            <HeaderButton
              variant="soft"
              icon="pencil"
              label={tLogPeriod('dateEditor.open')}
              onClick={() => openEditor()}
            />
          }
        />

        <div className="cc-body">
          <SegmentedTabs
            label={tn('viewLabel')}
            value={view}
            tabs={[
              { value: 'month', label: tn('month') },
              { value: 'year', label: tn('year') },
            ]}
            onChange={(v) => {
              if (v === 'year') setYearShown(anchor.year);
              setView(v);
            }}
          />

          {mounted && (isRecalculating || (calc.fetching && !loading)) ? (
            <div className="cc-status">
              <StatusPill tone="brand" icon="sparkle">
                {isRecalculating ? t('recalculating') : t('loadingCalendar')}
              </StatusPill>
            </div>
          ) : null}

          {body}

          {!calc.error && !loading ? (
            <CalendarLegend items={isTtc ? TTC_LEGEND : CYCLE_LEGEND} label={legendLabel} title={tn('legendTitle')} />
          ) : null}

          {noHistory && !loading ? (
            <EmptyState
              icon="drop"
              title={tn('emptyTitle')}
              body={tn('emptyBody')}
              action={
                <PrimaryButton onClick={() => openEditor()} icon="plus">
                  {t('logPeriodCta')}
                </PrimaryButton>
              }
            />
          ) : null}

          {view === 'month' && !calc.error && !loading ? (
            <DayCard
              date={selectedDate}
              locale={locale}
              cycleDay={selectedCalc?.cycleDay ?? null}
              phaseLabel={phaseKey ? tn(`phase.${phaseKey}`) : null}
              ttc={isTtc}
              isFuture={isFuture}
              onLog={() => router.push(isTtc ? `/fertility/log?date=${toApiDate(selectedDate)}` : `/log?date=${toApiDate(selectedDate)}`)}
              actions={actions}
            />
          ) : null}
        </div>
      </div>

      <PeriodDateEditor
        open={dateEditorOpen}
        onClose={() => {
          setDateEditorOpen(false);
          setDateEditorView(null);
        }}
        // The editor renders history up to next month; further out it opens on today's month.
        initialView={dateEditorView ?? (isFutureMonth ? { year: currentCal.year, month: currentCal.month } : anchor)}
        onSaved={() => setWatching(true)}
      />

      {actionError ? (
        <div role="alert" className="cal-toast" onClick={() => setActionError(null)}>
          {actionError}
        </div>
      ) : null}

      <BottomNav />
    </div>
  );
}
