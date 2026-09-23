import { describe, expect, it } from 'vitest';

import { filledCount, sectionsOf } from './sections';

describe('sectionsOf', () => {
  it('picks every listed field as translations, {} for null / packed []', () => {
    const row = { id: 1, faq: { fa: 'پرسش', en: null }, sleep: null, care_plan: [] };
    expect(sectionsOf(row, ['faq', 'sleep', 'care_plan', 'missing'])).toEqual({
      faq: { fa: 'پرسش' },
      sleep: {},
      care_plan: {},
      missing: {},
    });
  });

  it('counts sections with any text', () => {
    expect(filledCount({ a: { fa: 'x' }, b: { fa: ' ' }, c: {} })).toBe(1);
  });
});
