import { describe, expect, it } from 'vitest';

import { painDaySchema } from './pain';

describe('painDaySchema', () => {
  it('parses the golden day', () => {
    expect(
      painDaySchema.parse({
        date: '2026-09-22',
        score: 7,
        locations: ['pelvis', 'back'],
        relief: ['painkiller'],
        types: ['cramping'],
        associated: ['dyspareunia', 'dyschezia'],
        missed_activity: true,
        analgesic: 'Ibuprofen',
        analgesic_time: '08:00',
        analgesic_effect: 'a_little',
      }),
    ).toEqual({
      date: '2026-09-22',
      score: 7,
      locations: ['pelvis', 'back'],
      relief: ['painkiller'],
      types: ['cramping'],
      associated: ['dyspareunia', 'dyschezia'],
      missedActivity: true,
      analgesic: 'Ibuprofen',
      analgesicTime: '08:00',
      analgesicEffect: 'a_little',
    });
  });

  it('degrades malformed fields instead of failing', () => {
    expect(
      painDaySchema.parse({
        date: '2026-09-22',
        score: 12,
        locations: null,
        relief: ['heat', 3],
        types: 'cramping',
        associated: [],
        missed_activity: null,
        analgesic: '  ',
        analgesic_time: '8am',
        analgesic_effect: 'maybe',
      }),
    ).toEqual({
      date: '2026-09-22',
      score: null,
      locations: [],
      relief: ['heat'],
      types: [],
      associated: [],
      missedActivity: null,
      analgesic: null,
      analgesicTime: null,
      analgesicEffect: null,
    });
  });
});
