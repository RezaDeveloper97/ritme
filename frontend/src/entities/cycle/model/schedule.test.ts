import { describe, expect, it } from 'vitest';

import { diffInDays, fromApiDate, toApiDate } from '@/shared/lib/date';

import {
  cycleProgressPercent,
  cycleScheduleFor,
  daysUntilNextPeriod,
  deriveCycleSchedule,
  hasFertileWindow,
  scheduleDayMarker,
} from './schedule';
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

  it('derives the PMS run as the four days before the next period', () => {
    const s = deriveCycleSchedule(makeView(), makeCalc())!;

    expect(iso(s.pmsStart)).toBe('2026-08-07');
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
    expect(at('2026-08-07')).toBe('pms');
    expect(at('2026-08-10')).toBe('pms');
  });

  it('rolls into the next cycle once the predicted start is reached', () => {
    expect(at('2026-08-11')).toBe('period');
  });
});
