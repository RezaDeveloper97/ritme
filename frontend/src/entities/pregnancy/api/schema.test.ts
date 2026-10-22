import { describe, expect, it } from 'vitest';

import { onboardingDatingSchema } from './schema';

describe('onboardingDatingSchema (POST /pregnancy/onboarding `dating`, D-169)', () => {
  it('keeps the server block as is', () => {
    const block = { basis: 'lmp', replaced: false, discrepancy_days: 2, band_until_days: 97, limit_days: 7 };
    expect(onboardingDatingSchema.parse(block)).toEqual(block);
  });

  it('reads a missing or malformed block as null (no result line)', () => {
    expect(onboardingDatingSchema.parse(null)).toBeNull();
    expect(onboardingDatingSchema.parse('x')).toBeNull();
  });

  it('nulls a malformed field instead of failing the whole save', () => {
    expect(onboardingDatingSchema.parse({ basis: 'ultrasound', replaced: true, discrepancy_days: '12' })).toEqual({
      basis: 'ultrasound',
      replaced: true,
      discrepancy_days: null,
      band_until_days: null,
      limit_days: null,
    });
  });
});
