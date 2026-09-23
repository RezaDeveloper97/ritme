import { describe, expect, it } from 'vitest';

import { toPregnancyDayBody } from './v2-body';

describe('toPregnancyDayBody', () => {
  it('maps the Log screen input to the PUT body', () => {
    expect(
      toPregnancyDayBody({
        mood: 4,
        symptoms: { spotting: 'mild', nausea: 'moderate' },
        waterGlasses: 5,
        weight: 62.44,
        visitNote: '  سردرد خفیف  ',
      }),
    ).toEqual({
      mood: 4,
      // Canonical (artboard) order, whatever order the user tapped.
      symptoms: { nausea: 'moderate', spotting: 'mild' },
      water_glasses: 5,
      weight: 62.4,
      visit_note: 'سردرد خفیف',
    });
  });

  it('omits undefined fields and sends explicit nulls (clear)', () => {
    expect(toPregnancyDayBody({})).toEqual({});
    expect(toPregnancyDayBody({ mood: null, waterGlasses: null, weight: null, visitNote: null, symptoms: {} })).toEqual({
      mood: null,
      water_glasses: null,
      weight: null,
      visit_note: null,
      symptoms: {},
    });
  });

  it('clamps water, nulls bad weights and blank notes, drops unknown symptoms', () => {
    expect(
      toPregnancyDayBody({
        waterGlasses: 22,
        weight: Number.NaN,
        visitNote: '   ',
        symptoms: { hiccups: 'mild' } as never,
      }),
    ).toEqual({ water_glasses: 15, weight: null, visit_note: null, symptoms: {} });
    expect(toPregnancyDayBody({ waterGlasses: -2, weight: -1 })).toEqual({ water_glasses: 0, weight: null });
  });
});
