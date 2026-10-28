import { describe, expect, it } from 'vitest';

import { usesHormonalMethod } from './hormonal';

const meds = (medications: string[] | null) => ({
  conditions: { chronic_illnesses: null, gyn_conditions: null, medications },
});

describe('usesHormonalMethod (ReadyStep hides the fertile window)', () => {
  it('is true for the pill and other hormonal medication', () => {
    expect(usesHormonalMethod(meds(['contraceptive_pill']))).toBe(true);
    expect(usesHormonalMethod(meds(['hormonal_medication']))).toBe(true);
  });
  it('is false for non-hormonal choices, an IUD (copper unknown), none or skipped', () => {
    expect(usesHormonalMethod(meds(['iud']))).toBe(false);
    expect(usesHormonalMethod(meds([]))).toBe(false);
    expect(usesHormonalMethod(meds(null))).toBe(false);
  });
});
