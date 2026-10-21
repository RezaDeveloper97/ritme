import { describe, expect, it } from 'vitest';

import {
  cycleCalculationEnvelopeSchema,
  cycleMonthEnvelopeSchema,
  cycleViewSchema,
} from './schema';

/** Hand-copied from docs/engine-v6/SPEC.md §2.35, plus `today_ui` / `limits` (E-17, E-24). */
const SPEC_PAYLOAD = {
  date: '2026-07-18',
  cycle_day: 18,
  main_phase: 'luteal',
  subphase: 'early_luteal',
  fertility_level: 'low',
  days_to_ovulation: -2,
  days_to_period: 12,
  days_late: 0,
  anchors: {
    current_period_start: '2026-07-01',
    current_period_start_source: 'user_logged',
    current_period_end: '2026-07-05',
    current_period_end_source: 'user_logged',
    current_period_end_is_confirmed: true,
    predicted_next_period_start: '2026-07-30',
    estimated_ovulation_date: '2026-07-16',
    ovulation_source: 'calendar',
  },
  metrics: {
    effective_cycle_length: 29,
    effective_period_length: 5,
    cycle_variability: 3,
    cycle_regularity: 'regular',
  },
  clinical_flags: { frequency: 'normal', duration: 'normal', absent: false },
  future_cycles: [{ k: 2, period_start: '2026-08-28', ovulation: '2026-08-14' }],
  suggest_pregnancy_test: false,
  confidence: 'high',
  confidence_reasons: [],
  data_quality: 'good',
  resolution_source: 'prediction',
  is_predicted: true,
  requires_user_input: false,
  warnings: [],
  today_ui: {
    ring_action: 'edit',
    banner: null,
    show_pregnancy_test_button: false,
    prediction_card: {
      period: { start: '2026-07-30', end: '2026-08-03' },
      pms: { start: '2026-07-27', end: '2026-07-29' },
      fertile: { start: '2026-07-11', end: '2026-07-16' },
      ovulation: { date: '2026-07-16', source: 'calendar' },
      cycle_length: 29,
      cycle_regularity: 'regular',
    },
  },
  limits: {
    fertility_suppressed: false,
    suppression_reason: null,
    engine_suspended: false,
    prompt_pregnancy_mode: false,
  },
};

/** A Laravel-era `cycle_view`: no v6 keys, full value layers and predictions. */
const LARAVEL_VIEW = {
  date: '2026-07-24',
  cycle_day: 11,
  phase: 'ovulation',
  main_phase: 'fertile',
  fertility_level: 'medium',
  anchors: {
    current_period_start: '2026-07-14',
    current_period_start_source: 'user_logged',
    current_period_end: '2026-07-16',
    current_period_end_source: 'user_logged',
    current_period_end_is_confirmed: true,
    predicted_next_period_start: '2026-08-11',
    estimated_ovulation_date: '2026-07-28',
  },
  metrics: {
    effective_cycle_length: 28,
    effective_period_length: 3,
    cycle_variability: null,
  },
  predictions: {
    next_period_start: '2026-08-11',
    next_period_end: '2026-08-13',
    estimated_ovulation_date: '2026-07-28',
    fertile_window_start: '2026-07-23',
    fertile_window_end: '2026-07-29',
    source: 'profile',
    confidence: 'low',
    confidence_reasons: [],
  },
  profile_values: { cycle_length: 28, period_duration: 5 },
  calculated_values: {
    cycle_length: null,
    period_duration: 3,
    based_on_cycles: null,
  },
  effective_values: {
    cycle_length: 28,
    period_duration: 3,
    source: 'profile',
    period_duration_source: 'recent_valid_cycles',
  },
  daily_card: {
    title: 't',
    subtitle: 's',
    data_status: 'predicted',
    fertility_level: 'medium',
    fertility_label: 'm',
  },
};

/** Hormonal contraception: everything fertility-related is null (gap 02c risk 1). */
const PILL_VIEW = {
  date: '2026-07-18',
  cycle_day: 9,
  main_phase: 'follicular',
  fertility_level: 'none',
  days_to_ovulation: null,
  anchors: {
    current_period_start: '2026-07-10',
    current_period_start_source: 'user_logged',
    current_period_end: null,
    current_period_end_source: 'unknown',
    current_period_end_is_confirmed: false,
    predicted_next_period_start: '2026-08-07',
    estimated_ovulation_date: null,
    ovulation_source: 'unknown',
  },
  metrics: { effective_cycle_length: null, effective_period_length: null },
  predictions: {
    next_period_start: null,
    next_period_end: null,
    estimated_ovulation_date: null,
    fertile_window_start: null,
    fertile_window_end: null,
  },
  profile_values: null,
  warnings: ['hormonal_contraception_active'],
  limits: {
    fertility_suppressed: true,
    suppression_reason: 'hormonal_contraception',
  },
  future_cycles: null,
  daily_card: {
    title: 't',
    data_status: 'predicted',
    fertility_level: 'unknown',
  },
};

describe('cycleViewSchema — v6 payloads', () => {
  it('parses the SPEC §2.35 example with today_ui and limits', () => {
    const v = cycleViewSchema.parse(SPEC_PAYLOAD);

    expect(v.mainPhase).toBe('luteal');
    expect(v.anchors?.ovulationSource).toBe('calendar');
    expect(v.metrics?.cycleRegularity).toBe('regular');
    expect(v.clinicalFlags).toEqual({
      frequency: 'normal',
      duration: 'normal',
      absent: false,
    });
    expect(v.futureCycles).toEqual([{ k: 2, periodStart: '2026-08-28', ovulation: '2026-08-14' }]);
    expect(v.suggestPregnancyTest).toBe(false);
    expect(v.todayUi?.ringAction).toBe('edit');
    expect(v.todayUi?.predictionCard?.ovulation).toEqual({
      date: '2026-07-16',
      source: 'calendar',
    });
    expect(v.todayUi?.predictionCard?.cycleRegularity).toBe('regular');
    expect(v.limits?.fertilitySuppressed).toBe(false);
    // The value layers are absent from the spec payload: defaulted, not thrown on.
    expect(v.effectiveValues.cycleLength).toBeNull();
  });

  it('still parses a Laravel payload and leaves the v6 fields empty', () => {
    const v = cycleViewSchema.parse(LARAVEL_VIEW);

    expect(v.forecast?.nextPeriodStart).toBe('2026-08-11');
    expect(v.effectiveValues.cycleLength).toBe(28);
    expect(v.dailyCard?.fertilityLevel).toBe('medium');
    expect(v.anchors?.ovulationSource).toBe('unknown');
    expect(v.metrics?.cycleRegularity).toBeNull();
    expect(v.futureCycles).toEqual([]);
    expect(v.clinicalFlags).toBeNull();
    expect(v.suggestPregnancyTest).toBe(false);
    expect(v.todayUi).toBeNull();
    expect(v.limits).toBeNull();
  });

  it('parses a pill payload with null ovulation, window, predictions and metrics', () => {
    const v = cycleViewSchema.parse(PILL_VIEW);

    expect(v.anchors?.estimatedOvulationDate).toBeNull();
    expect(v.metrics).toEqual(
      expect.objectContaining({
        effectiveCycleLength: null,
        effectivePeriodLength: null,
      }),
    );
    expect(v.forecast?.estimatedOvulationDate).toBeNull();
    expect(v.forecast?.fertileWindowStart).toBeNull();
    expect(v.profileValues).toEqual({
      cycleLength: null,
      periodDuration: null,
    });
    expect(v.futureCycles).toEqual([]);
    expect(v.limits?.fertilitySuppressed).toBe(true);
    expect(v.dailyCard?.fertilityLevel).toBe('unknown');
  });

  it('defaults missing value layers instead of throwing', () => {
    const v = cycleViewSchema.parse({ date: '2026-07-18' });

    expect(v.profileValues).toEqual({
      cycleLength: null,
      periodDuration: null,
    });
    expect(v.calculatedValues.basedOnCycles).toBeNull();
    expect(v.effectiveValues.source).toBe('default');
  });

  it('degrades an unknown main_phase / level / source to unknown', () => {
    const v = cycleViewSchema.parse({
      ...SPEC_PAYLOAD,
      main_phase: 'brand_new_phase',
      fertility_level: 'astronomical',
      confidence: 'certain',
      resolution_source: 'telepathy',
      anchors: { ...SPEC_PAYLOAD.anchors, ovulation_source: 'crystal_ball' },
      metrics: { ...SPEC_PAYLOAD.metrics, cycle_regularity: 'wobbly' },
    });

    expect(v.mainPhase).toBe('unknown');
    expect(v.fertilityLevel).toBe('unknown');
    expect(v.confidence).toBe('unknown');
    expect(v.resolutionSource).toBe('unknown');
    expect(v.anchors?.ovulationSource).toBe('unknown');
    expect(v.metrics?.cycleRegularity).toBeNull();
  });

  it('accepts peak/unknown on the daily card and degrades anything else to unknown', () => {
    const card = (fertility_level: string) =>
      cycleViewSchema.parse({
        date: 'd',
        daily_card: { title: 't', data_status: 'predicted', fertility_level },
      }).dailyCard?.fertilityLevel;

    expect(card('peak')).toBe('peak');
    expect(card('unknown')).toBe('unknown');
    expect(card('weird')).toBe('unknown');
  });

  it('never throws on malformed v6 blocks', () => {
    const v = cycleViewSchema.parse({
      date: 'd',
      future_cycles: [{ k: 'x' }, { k: 3, period_start: '2026-09-25' }, 'junk'],
      clinical_flags: 'bad',
      suggest_pregnancy_test: 'yes',
      today_ui: 42,
      limits: [],
    });

    expect(v.futureCycles).toEqual([{ k: 3, periodStart: '2026-09-25', ovulation: null }]);
    expect(v.clinicalFlags).toBeNull();
    expect(v.suggestPregnancyTest).toBe(false);
    expect(v.todayUi).toBeNull();
    expect(v.limits).toBeNull();
  });
});

describe('envelopes — v6', () => {
  const day = (date: string, ovulation: number | null) => ({
    calculation_date: date,
    cycle_day: 3,
    phase: 'follicular',
    estimated_ovulation_day: ovulation,
    cycle_length_used: 28,
  });

  it('accepts a null per-day estimated_ovulation_day', () => {
    const e = cycleCalculationEnvelopeSchema.parse({
      calculation: day('2026-07-18', null),
    });

    expect(e.calculation?.estimatedOvulationDay).toBeNull();
  });

  it('drops one malformed day, not the month', () => {
    const e = cycleMonthEnvelopeSchema.parse({
      calculations: [day('2026-07-01', 14), { calculation_date: 5 }, null, day('2026-07-03', null)],
    });

    expect(e.calculations.map((c) => c.calculationDate)).toEqual(['2026-07-01', '2026-07-03']);
  });

  it('parses a full envelope with a v6 cycle_view', () => {
    const e = cycleCalculationEnvelopeSchema.parse({
      calculation: null,
      cycle_view: SPEC_PAYLOAD,
    });

    expect(e.cycleView?.todayUi?.banner).toBeNull();
  });
});
