import { describe, expect, it } from 'vitest';

import { cycleCalculationEnvelopeSchema, cycleCalculationSchema } from './schema';

/**
 * Boundary contract for the daily recommendations. They are admin-managed
 * (edited in the backend panel), so the parser has to survive a payload written
 * by whoever is on the other end — including a backend older than the one that
 * started sending `title`.
 */
describe('cycleCalculationSchema — daily tips', () => {
  const base = {
    calculation_date: '2026-07-30',
    cycle_day: 3,
    phase: 'menstruation',
    subphase: 'menstruation',
    estimated_ovulation_day: 14,
    cycle_length_used: 28,
  };

  const parse = (daily_tips: unknown[]) =>
    cycleCalculationSchema.parse({ ...base, daily_tips }).dailyTips;

  it('keeps the server-resolved title', () => {
    expect(parse([{ type: 'nutrition', title: 'تغذیه', text: 'آهن بخورید' }])).toEqual([
      { type: 'nutrition', title: 'تغذیه', text: 'آهن بخورید' },
    ]);
  });

  it('parses a tip from a backend that sends no title', () => {
    const [tip] = parse([{ type: 'sleep', text: 'زودتر بخوابید' }]);

    expect(tip.title).toBeUndefined();
    expect(tip.text).toBe('زودتر بخوابید');
  });

  /** A null title must cost the heading only — never the advice itself. */
  it('keeps the tip when the title is null', () => {
    const [tip] = parse([{ type: 'sleep', title: null, text: 'زودتر بخوابید' }]);

    expect(tip.title).toBeUndefined();
    expect(tip.text).toBe('زودتر بخوابید');
  });

  /** "Present" must mean renderable, so a blank heading never reaches the card. */
  it('treats a blank title as absent', () => {
    expect(parse([{ type: 'sleep', title: '   ', text: 'متن' }])[0].title).toBeUndefined();
  });

  it('defaults a missing category to general', () => {
    expect(parse([{ text: 'متن' }])[0].type).toBe('general');
  });

  /** A malformed entry is dropped; the rest of the card still renders. */
  it('drops an unparsable tip without losing the others', () => {
    expect(parse([{ type: 'sleep' }, { type: 'mood', text: 'متن' }])).toEqual([
      { type: 'mood', title: undefined, text: 'متن' },
    ]);
  });
});

/**
 * v6 payloads (captured from backend-go/contract/golden-v6/cycle/today.*.en.json):
 * the engine sends explicit nulls where Laravel sent numbers/strings.
 */
describe('cycleCalculationSchema — v6 nulls', () => {
  const v6 = {
    calculation_date: '2026-09-23',
    cycle_day: 49,
    phase: null,
    subphase: null,
    estimated_ovulation_day: 15,
    cycle_length_used: 28,
    is_fertile_window: false,
    is_pms_window: false,
    is_period_tomorrow: false,
    is_luteal_spotting: null,
    cycle_score: null,
    age_factor: null,
    base_probability: null,
    symptom_score: null,
    final_probability: null,
    cycle_variability: 'regular',
    uncertainty_range: null,
    text_flags: null,
    daily_tips: [],
    source_profile_data: null,
    source_daily_log_data: null,
  };

  it('parses final_probability: null as "no number", not 0%', () => {
    const c = cycleCalculationSchema.parse({ ...v6, phase: 'follicular' });

    expect(Number.isNaN(c.fertilityPercent)).toBe(true);
    expect(c.fertilityPercent).not.toBe(0);
  });

  it('keeps a real probability', () => {
    expect(cycleCalculationSchema.parse({ ...v6, final_probability: 12.5 }).fertilityPercent).toBe(
      12.5,
    );
  });

  it('parses phase: null (late 15+ persona, overdue_20)', () => {
    expect(cycleCalculationSchema.parse(v6).phase).toBe('');
  });

  it('parses subphase / cycle_variability / estimated_ovulation_day null', () => {
    const c = cycleCalculationSchema.parse({
      ...v6,
      subphase: null,
      cycle_variability: null,
      estimated_ovulation_day: null,
    });

    expect(c.subphase).toBeNull();
    expect(c.cycleVariability).toBeNull();
    expect(c.estimatedOvulationDay).toBeNull();
  });

  it('envelope with the overdue_20 calculation parses', () => {
    const r = cycleCalculationEnvelopeSchema.parse({ calculation: v6, cycle_view: null });

    expect(r.calculation?.cycleDay).toBe(49);
  });

  it('envelope with an unresolved (cycle_day null) calculation collapses to null', () => {
    const r = cycleCalculationEnvelopeSchema.parse({
      calculation: { ...v6, cycle_day: null, cycle_length_used: null, phase: null },
    });

    expect(r.calculation).toBeNull();
  });
});
