import { describe, expect, it } from 'vitest';

import { toBreastfeedingBody } from '@/entities/postpartum';

import { breastfeedingAnswer, breastfeedingView } from './breastfeeding';

describe('breastfeeding switch (I-04)', () => {
  it('maps the stored answer: null = not asked (off), true = on, false = off + «no»', () => {
    expect(breastfeedingView(null)).toEqual({ checked: false, caption: 'notAsked' });
    expect(breastfeedingView(true)).toEqual({ checked: true, caption: 'yes' });
    expect(breastfeedingView(false)).toEqual({ checked: false, caption: 'no' });
  });

  it('shows the value being saved while the request runs', () => {
    expect(breastfeedingView(null, true)).toEqual({ checked: true, caption: 'yes' });
    expect(breastfeedingView(true, false)).toEqual({ checked: false, caption: 'no' });
    expect(breastfeedingView(false, undefined)).toEqual({ checked: false, caption: 'no' });
  });

  it('a flip is an answer, and the body is the snake-case PATCH /postpartum payload', () => {
    expect(breastfeedingAnswer(true)).toBe(true);
    expect(breastfeedingAnswer(false)).toBe(false);
    expect(toBreastfeedingBody(true)).toEqual({ breastfeeding: true });
    expect(toBreastfeedingBody(null)).toEqual({ breastfeeding: null });
  });
});
