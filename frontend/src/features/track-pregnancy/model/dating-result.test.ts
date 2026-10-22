import { describe, expect, it } from 'vitest';

import { datingResultKey, type PregnancyDating, pregnancyDatingSchema } from './dating-result';

const compared = (over: Partial<PregnancyDating>): PregnancyDating => ({
  basis: 'lmp',
  replaced: false,
  discrepancy_days: 5,
  band_until_days: 62,
  limit_days: 5,
  ...over,
});

describe('datingResultKey', () => {
  it('says the due date follows the scan when the scan replaced the LMP', () => {
    expect(datingResultKey(compared({ basis: 'ultrasound', replaced: true, discrepancy_days: 6 }))).toBe('replaced');
  });

  it('says the LMP date stayed when the discrepancy was within the limit', () => {
    expect(datingResultKey(compared({}))).toBe('kept');
  });

  it('shows nothing without a comparison', () => {
    expect(datingResultKey(null)).toBeNull();
    expect(datingResultKey(undefined)).toBeNull();
    expect(
      datingResultKey({ basis: 'ultrasound', replaced: false, discrepancy_days: null, band_until_days: null, limit_days: null }),
    ).toBeNull();
  });

  it('shows nothing when the basis in effect contradicts the rule (a user-chosen basis)', () => {
    expect(datingResultKey(compared({ basis: 'ultrasound', replaced: false }))).toBeNull();
    expect(datingResultKey(compared({ basis: 'lmp', replaced: true }))).toBeNull();
    expect(datingResultKey(compared({ basis: 'manual' }))).toBeNull();
  });

  it('parses the API block', () => {
    const parsed = pregnancyDatingSchema.parse({
      basis: 'ultrasound',
      replaced: true,
      discrepancy_days: 6,
      band_until_days: null,
      limit_days: 21,
    });
    expect(datingResultKey(parsed)).toBe('replaced');
  });
});
