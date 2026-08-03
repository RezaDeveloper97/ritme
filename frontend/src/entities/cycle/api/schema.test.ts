import { describe, expect, it } from 'vitest';

import { cycleCalculationSchema } from './schema';

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
