import { useLocale, useTranslations } from 'next-intl';

import {
  deriveCycleSchedule,
  hasFertileWindow,
  isFertilitySuppressed,
  serverFertileEvents,
  useCycleMonth,
  useCycleOverview,
  useCycleToday,
} from '@/entities/cycle';
import { useUserProfile } from '@/entities/user';
import type { Locale } from '@/shared/i18n';
import {
  addDays,
  diffInDays,
  formatDayMonth,
  formatNumber,
  fromApiDate,
  toApiDate,
  toParts,
  today,
} from '@/shared/lib/date';

import { homeLimits } from '../model/limits';
import { enginePredictionRows } from '../model/prediction-card';
import { isEnginePayload } from '../model/engine-payload';
import type { CycleLengthSummary, PredictionSlot } from './PredictionsCard';

/** One upcoming (or running) cycle event: its dates and the countdown to its start. */
export interface TimelineSlot {
  start: Date;
  end: Date;
  days: number;
}

const DEFAULT_PERIOD_DAYS = 5;

/** The Gregorian year/month `cycle/month` is keyed by. */
function gregYearMonth(date: Date): { year: number; month: number } {
  const [year, month] = toApiDate(date).split('-');
  return { year: Number(year), month: Number(month) };
}

/**
 * The «پیش‌بینی‌ها» dates — next period, PMS, fertile window, ovulation — and the cycle-length
 * summary, from the same cached queries the home reads (`/cycle/today`, `/home/cycle-overview`,
 * this and next month). Shared by the home (ring, chance card, PMS insight) and the
 * predictions card on `/analysis`, so both always show the same dates.
 */
export function usePredictionSlots() {
  const t = useTranslations('home');
  const loc = useLocale() as Locale;
  const todayQuery = useCycleToday();
  const todayData = todayQuery.data;
  const calc = todayData?.calculation ?? null;
  const profileQuery = useUserProfile();

  const base = today();
  // The cycle's own calendar (engine anchors → absolute dates). Everything the
  // page shows about upcoming events hangs off this, so the dates stay tied to
  // the period start rather than being re-measured from today each render.
  // Laravel payload only: a v6 payload (`today_ui` / `limits`) never gets a browser-side schedule.
  const engine = isEnginePayload(todayData?.cycleView ?? null);
  const schedule = engine ? null : deriveCycleSchedule(todayData?.cycleView ?? null, calc);
  const fertilitySuppressed = isFertilitySuppressed(todayData?.cycleView ?? null);
  // §2.40 / E-24: the engine's limits (hormonal contraception, postpartum, 45+, positive test).
  const limits = homeLimits(todayData?.cycleView ?? null);
  // Calendar formatting happens only here, at the display boundary (§7).
  const fmt = (date: Date) => formatDayMonth(date, loc);
  // «۱۷ تا ۲۱ مهر»: a range inside one month names the month once.
  const range = (from: Date, to: Date) => {
    const a = toParts(from, loc);
    const b = toParts(to, loc);
    const sameMonth = a.year === b.year && a.month === b.month;
    return t('dateRange', { from: sameMonth ? formatNumber(a.day, loc) : fmt(from), to: fmt(to) });
  };
  // Every prediction row describes the occurrence the user is actually waiting on:
  // an event whose window is already running is reported as in-progress, and one
  // that is wholly behind us rolls forward a cycle.
  // v6 (E-17): while the period is late there is no rolled-forward row — the late banner and the ring
  // already say so, and a made-up next date would hide the lateness.
  const todayView = todayData?.cycleView ?? null;
  const periodIsLate = engine && (todayView?.daysLate ?? 0) > 0;
  const slotFor = (start: Date, end: Date, rollForward = true): TimelineSlot | null => {
    if (engine) {
      // v6: the server dates are shown as sent; a window that is over is simply not listed.
      return diffInDays(end, base) < 0 ? null : { start, end, days: diffInDays(start, base) };
    }
    if (!schedule) return null;
    if (!rollForward && diffInDays(end, base) < 0) return null;
    let from = start;
    let to = end;
    while (diffInDays(to, base) < 0) {
      from = addDays(from, schedule.cycleLength);
      to = addDays(to, schedule.cycleLength);
    }
    return { start: from, end: to, days: diffInDays(from, base) };
  };
  // `/home/cycle-overview` (B-N1-06): the engine's period range, PMS window,
  // streak and cycle-length summary. Absent on a backend without the route —
  // the schedule derived from `cycle_view` stands in.
  const overviewQuery = useCycleOverview();
  const overview = overviewQuery.data ?? null;
  const periodDays =
    todayData?.cycleView?.effectiveValues.periodDuration ??
    profileQuery.data?.health?.periodDuration ??
    DEFAULT_PERIOD_DAYS;
  // v6: this month's and next month's server days + `future_cycles` (same cache as the week strip).
  const baseParts = gregYearMonth(base);
  const nextMonthParts = gregYearMonth(addDays(base, 31));
  const monthNow = useCycleMonth(baseParts.year, baseParts.month);
  const monthNext = useCycleMonth(nextMonthParts.year, nextMonthParts.month);
  const serverEvents = serverFertileEvents(
    todayView?.anchors?.estimatedOvulationDate,
    [...(monthNow.data?.days ?? []), ...(monthNext.data?.days ?? [])],
    [...(monthNow.data?.futureCycles ?? []), ...(todayView?.futureCycles ?? [])],
    toApiDate(base),
    fertilitySuppressed,
  );
  const predictedStart = todayView?.anchors?.predictedNextPeriodStart ?? null;
  // v6 `today_ui.prediction_card`: when present it owns every row of the «پیش‌بینی‌ها» card.
  const card = enginePredictionRows(todayView?.todayUi);
  const cardSlot = (r: { start: string; end: string } | null) =>
    r ? slotFor(fromApiDate(r.start), fromApiDate(r.end)) : null;
  const nextPeriodSlot = card
    ? cardSlot(card.period)
    : overview?.nextPeriod
    ? slotFor(fromApiDate(overview.nextPeriod.start), fromApiDate(overview.nextPeriod.end), !periodIsLate)
    : engine
      ? predictedStart
        ? slotFor(fromApiDate(predictedStart), fromApiDate(predictedStart), !periodIsLate)
        : null
      : schedule
        ? slotFor(schedule.nextPeriodStart, addDays(schedule.nextPeriodStart, periodDays - 1), !periodIsLate)
        : null;
  const ovulationSlot = card
    ? cardSlot(card.ovulation ? { start: card.ovulation.date, end: card.ovulation.date } : null)
    : engine
    ? serverEvents.ovulation
      ? slotFor(fromApiDate(serverEvents.ovulation), fromApiDate(serverEvents.ovulation))
      : null
    : schedule
      ? slotFor(schedule.ovulation, schedule.ovulation)
      : null;
  // The §19 display window — the same days `/fertility/bbt` and the insights
  // screen mark (none when a long period swallows it).
  const windowSlot = card
    ? cardSlot(card.fertile)
    : engine
    ? serverEvents.fertile
      ? slotFor(fromApiDate(serverEvents.fertile.start), fromApiDate(serverEvents.fertile.end))
      : null
    : schedule && hasFertileWindow(schedule)
      ? slotFor(schedule.fertileStart, schedule.fertileEnd)
      : null;
  const pmsSlot = card
    ? cardSlot(card.pms)
    : overview?.pmsWindow
    ? slotFor(fromApiDate(overview.pmsWindow.start), fromApiDate(overview.pmsWindow.end))
    : !engine && schedule
      ? slotFor(schedule.pmsStart, schedule.pmsEnd)
      : null;
  const nextPeriodDate = nextPeriodSlot ? fmt(nextPeriodSlot.start) : null;
  const ovulationDate = ovulationSlot ? fmt(ovulationSlot.start) : null;
  const toPrediction = (slot: TimelineSlot | null, single = false): PredictionSlot | null =>
    slot && {
      text: single ? fmt(slot.start) : range(slot.start, slot.end),
      startIn: slot.days,
      endIn: diffInDays(slot.end, base),
    };

  // «طول سیکل (میانه N سیکل)» — the engine's summary; without the overview
  // route the effective length alone stands in.
  const cycleLengthSummary = overview
    ? {
        median: card?.cycleLength ?? overview.cycleLength.median,
        basedOnCycles: overview.cycleLength.basedOnCycles,
        spreadDays: overview.cycleLength.spreadDays,
        regularity: card?.regularity ?? overview.cycleLength.regularity,
      }
    : todayData?.cycleView?.effectiveValues.cycleLength != null
      ? {
          median: card?.cycleLength ?? todayData.cycleView.effectiveValues.cycleLength,
          basedOnCycles: null,
          spreadDays: null,
          regularity: card?.regularity ?? null,
        }
      : null;

  return {
    todayQuery,
    base,
    engine,
    schedule,
    fertilitySuppressed,
    limits,
    fmt,
    overview,
    card,
    nextPeriodSlot,
    ovulationSlot,
    windowSlot,
    pmsSlot,
    nextPeriodDate,
    ovulationDate,
    toPrediction,
    cycleLengthSummary: cycleLengthSummary as CycleLengthSummary | null,
  };
}

export type { PredictionSlot };
