import { describe, expect, it } from 'vitest';

import { toFertilityDayBody } from './body';

describe('toFertilityDayBody', () => {
  it('maps a full day to the snake_case body', () => {
    expect(
      toFertilityDayBody({
        lh: 'positive',
        mucus: 'egg_white',
        bbt: 36.42,
        bbtTime: '06:45',
        intercourse: 'unprotected',
        symptoms: ['ovarian_pain'],
        note: '  یادداشت  ',
      }),
    ).toEqual({
      lh: 'positive',
      mucus: 'egg_white',
      bbt: 36.42,
      bbt_time: '06:45',
      intercourse: 'unprotected',
      symptoms: ['ovarian_pain'],
      note: 'یادداشت',
    });
  });

  it('omits undefined fields so the PUT stays partial', () => {
    expect(toFertilityDayBody({})).toEqual({});
    expect(toFertilityDayBody({ lh: 'faint' })).toEqual({ lh: 'faint' });
  });

  it('sends explicit null to clear', () => {
    expect(
      toFertilityDayBody({ lh: null, mucus: null, bbt: null, bbtTime: null, intercourse: null, note: null }),
    ).toEqual({ lh: null, mucus: null, bbt: null, bbt_time: null, intercourse: null, note: null });
  });

  it('rounds bbt, nulls a non-finite one and blanks', () => {
    expect(toFertilityDayBody({ bbt: 36.4249 })).toEqual({ bbt: 36.42 });
    expect(toFertilityDayBody({ bbt: Number.NaN })).toEqual({ bbt: null });
    expect(toFertilityDayBody({ note: '   ', bbtTime: '' })).toEqual({ note: null, bbt_time: null });
  });

  it('de-duplicates symptoms in canonical order; [] clears them', () => {
    expect(toFertilityDayBody({ symptoms: ['spotting', 'bloating', 'spotting'] })).toEqual({
      symptoms: ['bloating', 'spotting'],
    });
    expect(toFertilityDayBody({ symptoms: [] })).toEqual({ symptoms: [] });
  });
});
