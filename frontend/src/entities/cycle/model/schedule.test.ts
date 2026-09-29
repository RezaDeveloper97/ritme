import { describe, expect, it } from 'vitest';

import { addDays, diffInDays, fromApiDate, toApiDate } from '@/shared/lib/date';

import {
  cycleDayMarkerAt,
  cycleProgressPercent,
  cycleScheduleFor,
  daysUntilNextPeriod,
  deriveCycleSchedule,
  fertileWindowDays,
  hasFertileWindow,
  scheduleDayMarker,
} from './schedule';
import type { CycleSchedule } from './schedule';
import type { CycleCalculation, CycleView } from './types';

function makeCalc(overrides: Partial<CycleCalculation> = {}): CycleCalculation {
  return {
    calculationDate: '2026-07-24',
    cycleDay: 11,
    phase: 'follicular',
    subphase: null,
    estimatedOvulationDay: 15,
    cycleLength: 28,
    isFertileWindow: true,
    isPmsWindow: false,
    isPeriodTomorrow: false,
    fertilityPercent: 8.92,
    cycleVariability: 'regular',
    dailyTips: [],
    ...overrides,
  };
}

/** A `cycle_view` shaped like the engine's real answer for the calc above. */
function makeView(overrides: Partial<CycleView> = {}): CycleView {
  return {
    date: '2026-07-24',
    cycleDay: 11,
    phase: 'ovulation',
    subphase: 'fertile_rising',
    mainPhase: 'fertile',
    fertilityLevel: 'medium',
    dataStatus: 'predicted',
    daysToOvulation: 4,
    daysToPeriod: 18,
    daysLate: 0,
    anchors: {
      currentPeriodStart: '2026-07-14',
      currentPeriodStartSource: 'user_logged',
      currentPeriodEnd: '2026-07-16',
      currentPeriodEndSource: 'user_logged',
      currentPeriodEndIsConfirmed: true,
      predictedNextPeriodStart: '2026-08-11',
      estimatedOvulationDate: '2026-07-28',
    },
    metrics: { effectiveCycleLength: 28, effectivePeriodLength: 3, cycleVariability: null },
    confidence: 'low',
    confidenceReasons: [],
    dataQuality: 'partial',
    dataQualityDetails: null,
    resolutionSource: 'prediction',
    isPredicted: true,
    requiresUserInput: false,
    warnings: [],
    forecast: {
      nextPeriodStart: '2026-08-11',
      nextPeriodEnd: '2026-08-13',
      estimatedOvulationDate: '2026-07-28',
      fertileWindowStart: '2026-07-23',
      fertileWindowEnd: '2026-07-29',
      source: 'profile',
      confidence: 'low',
      confidenceReasons: [],
    },
    profileValues: { cycleLength: 28, periodDuration: 5 },
    calculatedValues: { cycleLength: null, periodDuration: 3, basedOnCycles: null },
    effectiveValues: {
      cycleLength: 28,
      periodDuration: 3,
      source: 'profile',
      periodDurationSource: 'recent_valid_cycles',
    },
    dailyCard: null,
    ...overrides,
  };
}

const iso = (d: Date) => toApiDate(d);

describe('deriveCycleSchedule', () => {
  it('uses the engine anchors and forecast verbatim', () => {
    const s = deriveCycleSchedule(makeView(), makeCalc())!;

    expect(iso(s.cycleStart)).toBe('2026-07-14');
    expect(iso(s.nextPeriodStart)).toBe('2026-08-11');
    expect(iso(s.ovulation)).toBe('2026-07-28');
    expect(iso(s.fertileStart)).toBe('2026-07-23');
    // §19 display window ends on ovulation, not on the forecast's biological O+1 (07-29).
    expect(iso(s.fertileEnd)).toBe('2026-07-28');
    expect(hasFertileWindow(s)).toBe(true);
    expect(s.cycleLength).toBe(28);
  });

  // The T-M5-11 staging case: last start 09-13 (5 days), 28-day cycle, today =
  // cycle day 16. `/fertility/bbt` draws the window on cycle days 10–15; the
  // home schedule must give the same days (it used to end on day 16).
  it('matches /fertility/bbt on cycle day 16: window = days 10–15, ending on ovulation', () => {
    const s = deriveCycleSchedule(
      makeView({
        date: '2026-09-28',
        cycleDay: 16,
        anchors: {
          currentPeriodStart: '2026-09-13',
          currentPeriodStartSource: 'user_logged',
          currentPeriodEnd: '2026-09-17',
          currentPeriodEndSource: 'user_logged',
          currentPeriodEndIsConfirmed: true,
          predictedNextPeriodStart: '2026-10-11',
          estimatedOvulationDate: '2026-09-27',
        },
        forecast: {
          nextPeriodStart: '2026-10-11',
          nextPeriodEnd: '2026-10-15',
          estimatedOvulationDate: '2026-09-27',
          fertileWindowStart: '2026-09-22',
          fertileWindowEnd: '2026-09-28',
          source: 'recent_valid_cycles',
          confidence: 'high',
          confidenceReasons: [],
        },
      }),
      makeCalc({ calculationDate: '2026-09-28', cycleDay: 16 }),
    )!;
    const cycleDay = (d: Date) => diffInDays(d, s.cycleStart) + 1;

    expect(cycleDay(s.fertileStart)).toBe(10);
    expect(cycleDay(s.fertileEnd)).toBe(15);
    expect(iso(s.fertileEnd)).toBe('2026-09-27');
    expect(scheduleDayMarker(s, fromApiDate('2026-09-28'), 5)).toBeNull();
  });

  it('starts the window the day after a long period (max(O−5, period end + 1))', () => {
    const view = makeView();
    const s = deriveCycleSchedule(
      makeView({ anchors: { ...view.anchors!, currentPeriodEnd: '2026-07-24' } }),
      makeCalc(),
    )!;
    expect(iso(s.fertileStart)).toBe('2026-07-25');
    expect(iso(s.fertileEnd)).toBe('2026-07-28');
  });

  it('has no window when the period runs past ovulation (§19 empty window)', () => {
    const view = makeView();
    const s = deriveCycleSchedule(
      makeView({ anchors: { ...view.anchors!, currentPeriodEnd: '2026-07-28' } }),
      makeCalc(),
    )!;
    expect(hasFertileWindow(s)).toBe(false);
  });

  it('derives the PMS run as the three days before the next period (task.md §25.2)', () => {
    const s = deriveCycleSchedule(makeView(), makeCalc())!;

    expect(iso(s.pmsStart)).toBe('2026-08-08');
    expect(iso(s.pmsEnd)).toBe('2026-08-10');
  });

  it('anchors to the calculation date, not to today, when only a calculation exists', () => {
    const s = deriveCycleSchedule(null, makeCalc())!;

    // cycle day 11 on 2026-07-24 ⇒ day 1 was 2026-07-14.
    expect(iso(s.cycleStart)).toBe('2026-07-14');
    expect(iso(s.nextPeriodStart)).toBe('2026-08-11');
    expect(iso(s.ovulation)).toBe('2026-07-28');
    expect(iso(s.fertileStart)).toBe('2026-07-23');
  });

  it('returns null when the cycle cannot be placed', () => {
    expect(deriveCycleSchedule(null, null)).toBeNull();
  });
});

describe('cycleScheduleFor', () => {
  const schedule = deriveCycleSchedule(makeView(), makeCalc())!;

  it('keeps the same cycle for a day inside it', () => {
    expect(iso(cycleScheduleFor(schedule, fromApiDate('2026-08-10')).cycleStart)).toBe('2026-07-14');
  });

  it('rolls forward whole cycles for a later day', () => {
    const rolled = cycleScheduleFor(schedule, fromApiDate('2026-08-11'));

    expect(iso(rolled.cycleStart)).toBe('2026-08-11');
    expect(iso(rolled.nextPeriodStart)).toBe('2026-09-08');
  });

  it('rolls back for a day in a previous cycle', () => {
    expect(iso(cycleScheduleFor(schedule, fromApiDate('2026-07-13')).cycleStart)).toBe('2026-06-16');
  });
});

describe('daysUntilNextPeriod', () => {
  const schedule = deriveCycleSchedule(makeView(), makeCalc())!;

  it('counts from the given day to that cycle’s predicted start', () => {
    expect(daysUntilNextPeriod(schedule, fromApiDate('2026-07-25'))).toBe(17);
    expect(daysUntilNextPeriod(schedule, fromApiDate('2026-08-10'))).toBe(1);
  });

  it('never counts down to a period that has already started', () => {
    expect(daysUntilNextPeriod(schedule, fromApiDate('2026-08-11'))).toBe(28);
  });
});

describe('cycleProgressPercent', () => {
  const schedule = deriveCycleSchedule(makeView(), makeCalc())!;

  it('is 0 on day 1 and grows through the cycle', () => {
    expect(cycleProgressPercent(schedule, fromApiDate('2026-07-14'))).toBe(0);
    expect(cycleProgressPercent(schedule, fromApiDate('2026-07-28'))).toBe(50);
    expect(cycleProgressPercent(schedule, fromApiDate('2026-08-10'))).toBe(96);
  });
});

describe('scheduleDayMarker', () => {
  // Cycle starts 2026-07-14, ovulation 2026-07-28, fertile 07-23…07-28,
  // PMS 08-07…08-10, next period 2026-08-11.
  const schedule = deriveCycleSchedule(makeView(), makeCalc())!;
  const at = (iso: string) => scheduleDayMarker(schedule, fromApiDate(iso), 5);

  it('marks the first `periodLength` days of the cycle as period', () => {
    expect(at('2026-07-14')).toBe('period');
    expect(at('2026-07-18')).toBe('period');
    expect(at('2026-07-19')).toBeNull();
  });

  it('lets ovulation win over the fertile window around it', () => {
    expect(at('2026-07-23')).toBe('fertile');
    expect(at('2026-07-28')).toBe('ovulation');
    // The display window ends on ovulation (§19): the day after is luteal.
    expect(at('2026-07-29')).toBeNull();
  });

  it('marks the run of days before the next period as PMS', () => {
    expect(at('2026-08-07')).toBeNull();
    expect(at('2026-08-08')).toBe('pms');
    expect(at('2026-08-10')).toBe('pms');
  });

  it('rolls into the next cycle once the predicted start is reached', () => {
    expect(at('2026-08-11')).toBe('period');
  });
});

/**
 * A legacy per-day calculation (`cycle/month`) as the engine builds it: phase
 * `menstruation` for the bleeding days, `ovulation` on O and O+1, the
 * biological fertile flag on O−5 … O+1 minus bleeding days, PMS from L−6.
 */
function legacyDay(isoDate: string, cycleStart: string, o: number, length: number, bleeding: number) {
  const cycleDay = diffInDays(fromApiDate(isoDate), fromApiDate(cycleStart)) + 1;
  const period = cycleDay <= bleeding;
  const phase = period
    ? 'menstruation'
    : cycleDay < o
      ? 'follicular'
      : cycleDay <= o + 1
        ? 'ovulation'
        : 'luteal';
  return makeCalc({
    calculationDate: isoDate,
    cycleDay,
    phase,
    estimatedOvulationDay: o,
    cycleLength: length,
    isFertileWindow: !period && cycleDay >= o - 5 && cycleDay <= o + 1,
    isPmsWindow: !period && cycleDay >= length - 6,
  });
}

/** Marker per ISO day over [from, to], resolved by the shared helper. */
function markers(
  schedule: CycleSchedule | null,
  from: string,
  to: string,
  calcOf: (iso: string) => CycleCalculation | undefined,
  periodLength = 5,
): Record<string, string | null> {
  const out: Record<string, string | null> = {};
  for (let d = fromApiDate(from); diffInDays(fromApiDate(to), d) >= 0; d = addDays(d, 1)) {
    out[iso(d)] = cycleDayMarkerAt(d, calcOf(iso(d)), schedule, periodLength);
  }
  return out;
}

const fertileDays = (m: Record<string, string | null>) =>
  Object.entries(m)
    .filter(([, v]) => v === 'fertile' || v === 'ovulation')
    .map(([k, v]) => (v === 'ovulation' ? `${k}*` : k));

describe('cycleDayMarkerAt — one §19 window for every surface (T-M5-13)', () => {
  // The T-M5-11/12 staging case: 5-day period from 09-13, 28-day cycle, today =
  // cycle day 16 (09-28). `/fertility/bbt` + home text: days 10–15 = 09-22…09-27.
  const view = makeView({
    date: '2026-09-28',
    cycleDay: 16,
    anchors: {
      currentPeriodStart: '2026-09-13',
      currentPeriodStartSource: 'user_logged',
      currentPeriodEnd: '2026-09-17',
      currentPeriodEndSource: 'user_logged',
      currentPeriodEndIsConfirmed: true,
      predictedNextPeriodStart: '2026-10-11',
      estimatedOvulationDate: '2026-09-27',
    },
    metrics: { effectiveCycleLength: 28, effectivePeriodLength: 5, cycleVariability: null },
    forecast: {
      nextPeriodStart: '2026-10-11',
      nextPeriodEnd: '2026-10-15',
      estimatedOvulationDate: '2026-09-27',
      fertileWindowStart: '2026-09-22',
      fertileWindowEnd: '2026-09-28',
      source: 'recent_valid_cycles',
      confidence: 'high',
      confidenceReasons: [],
    },
  });
  const schedule = deriveCycleSchedule(view, makeCalc({ calculationDate: '2026-09-28', cycleDay: 16 }))!;
  const legacy = (isoDate: string) => {
    // Legacy engine: this cycle from 09-13, the next from 10-11, the one before from 08-16.
    const start = isoDate >= '2026-10-11' ? '2026-10-11' : isoDate >= '2026-09-13' ? '2026-09-13' : '2026-08-16';
    return legacyDay(isoDate, start, 15, 28, 5);
  };

  it('cycle day 16: the calendar paints days 10–15, like home and /fertility/bbt', () => {
    const m = markers(schedule, '2026-09-13', '2026-10-10', legacy);
    expect(fertileDays(m)).toEqual([
      '2026-09-22',
      '2026-09-23',
      '2026-09-24',
      '2026-09-25',
      '2026-09-26',
      '2026-09-27*',
    ]);
    // The legacy O+1 day (flag + phase `ovulation`) is luteal now.
    expect(m['2026-09-28']).toBeNull();
    expect(m['2026-09-13']).toBe('period');
    // PMS = the §25.2 three days before 10-11, as on the home timeline — not
    // the legacy 7-day `is_pms_window` run (10-04 … 10-10).
    expect(m['2026-10-04']).toBeNull();
    expect(m['2026-10-07']).toBeNull();
    expect(m['2026-10-08']).toBe('pms');
    expect(m['2026-10-10']).toBe('pms');
    // The same days as the timeline bar (cycle days) and the home rows (dates).
    expect(fertileWindowDays(schedule)).toEqual({ startDay: 10, endDay: 15, ovulationDay: 15 });
    expect(iso(schedule.fertileStart)).toBe('2026-09-22');
    expect(iso(schedule.fertileEnd)).toBe('2026-09-27');
  });

  it('predicted next cycle: the rolled window, days 10–15 again', () => {
    const m = markers(schedule, '2026-10-11', '2026-11-07', legacy);
    expect(fertileDays(m)).toEqual([
      '2026-10-20',
      '2026-10-21',
      '2026-10-22',
      '2026-10-23',
      '2026-10-24',
      '2026-10-25*',
    ]);
  });

  it('past cycles read the legacy day by the same §19 rule (O+1 dropped)', () => {
    const m = markers(schedule, '2026-08-16', '2026-09-12', legacy);
    expect(fertileDays(m)).toEqual([
      '2026-08-25',
      '2026-08-26',
      '2026-08-27',
      '2026-08-28',
      '2026-08-29',
      '2026-08-30*',
    ]);
  });

  it('the anchored schedule wins over a legacy day that disagrees in the current cycle', () => {
    // A legacy calc a day off (ovulation on day 16): the schedule still decides.
    const off = (isoDate: string) => legacyDay(isoDate, '2026-09-13', 16, 29, 5);
    const m = markers(schedule, '2026-09-18', '2026-09-30', off);
    expect(fertileDays(m)).toEqual([
      '2026-09-22',
      '2026-09-23',
      '2026-09-24',
      '2026-09-25',
      '2026-09-26',
      '2026-09-27*',
    ]);
  });

  it('falls back to the schedule alone for a day without a calculation', () => {
    expect(cycleDayMarkerAt(fromApiDate('2026-09-27'), undefined, schedule, 5)).toBe('ovulation');
    expect(cycleDayMarkerAt(fromApiDate('2026-09-28'), undefined, schedule, 5)).toBeNull();
    expect(cycleDayMarkerAt(fromApiDate('2026-09-28'), undefined, null, 5)).toBeNull();
  });

  it('a logged period day always wins over the window', () => {
    const bleeding = makeCalc({ calculationDate: '2026-09-25', cycleDay: 13, phase: 'menstruation' });
    expect(cycleDayMarkerAt(fromApiDate('2026-09-25'), bleeding, schedule, 5)).toBe('period');
  });
});

describe('short cycle whose period runs past O−5', () => {
  // 21-day cycle, 7-day period from 07-01 (ends 07-07): O = 07-08 (day 8),
  // O−5 = day 3 sits inside the period, so §19 leaves only the ovulation day.
  const view = makeView({
    date: '2026-07-05',
    cycleDay: 5,
    anchors: {
      currentPeriodStart: '2026-07-01',
      currentPeriodStartSource: 'user_logged',
      currentPeriodEnd: '2026-07-07',
      currentPeriodEndSource: 'user_logged',
      currentPeriodEndIsConfirmed: true,
      predictedNextPeriodStart: '2026-07-22',
      estimatedOvulationDate: '2026-07-08',
    },
    metrics: { effectiveCycleLength: 21, effectivePeriodLength: 7, cycleVariability: null },
    forecast: null,
  });
  const schedule = deriveCycleSchedule(view, null)!;
  const legacy = (isoDate: string) =>
    legacyDay(isoDate, isoDate >= '2026-07-22' ? '2026-07-22' : '2026-07-01', 8, 21, 7);

  it('the window is the ovulation day alone, in the schedule and on the calendar', () => {
    expect(fertileWindowDays(schedule)).toEqual({ startDay: 8, endDay: 8, ovulationDay: 8 });
    const m = markers(schedule, '2026-07-01', '2026-07-21', legacy, 7);
    expect(fertileDays(m)).toEqual(['2026-07-08*']);
    // Legacy O+1 (flag + `ovulation` phase) is not part of it.
    expect(m['2026-07-09']).toBeNull();
    expect(m['2026-07-07']).toBe('period');
  });

  it('the predicted next cycle opens after its own (effective-length) period', () => {
    const next = cycleScheduleFor(schedule, fromApiDate('2026-07-25'));
    expect(iso(next.fertileStart)).toBe('2026-07-29');
    expect(iso(next.fertileEnd)).toBe('2026-07-29');
    const m = markers(schedule, '2026-07-22', '2026-08-11', legacy, 7);
    expect(fertileDays(m)).toEqual(['2026-07-29*']);
  });

  it('a long logged period does not shrink later, predicted windows', () => {
    // Current period logged 8 days, effective length 5: the next cycle's window
    // is the full O−5 … O again.
    const long = deriveCycleSchedule(
      makeView({ anchors: { ...makeView().anchors!, currentPeriodEnd: '2026-07-24' } }),
      makeCalc(),
    )!;
    expect(iso(long.fertileStart)).toBe('2026-07-25');
    const next = cycleScheduleFor(long, fromApiDate('2026-08-20'));
    expect(iso(next.fertileStart)).toBe('2026-08-20');
    expect(iso(next.fertileEnd)).toBe('2026-08-25');
  });
});
