import { describe, expect, it } from 'vitest';

import { checkupTypeSchema } from '../api/checkup-types';
import { cleanTranslations, moveItem, registryLocked, rowToBody } from './payload';

describe('moveItem', () => {
  it('moves up and down', () => {
    expect(moveItem([1, 2, 3], 2, 0)).toEqual([3, 1, 2]);
    expect(moveItem([1, 2, 3], 0, 1)).toEqual([2, 1, 3]);
  });
  it('ignores out-of-range moves', () => {
    expect(moveItem([1, 2], 1, 2)).toEqual([1, 2]);
  });
});

describe('cleanTranslations', () => {
  it('drops blanks and returns null when empty', () => {
    expect(cleanTranslations({ fa: ' x ', en: '' })).toEqual({ fa: 'x' });
    expect(cleanTranslations({ fa: '  ' })).toBeNull();
  });
});

describe('v6 catalog fields (T-M10-21)', () => {
  const row = (over: Record<string, unknown>) =>
    checkupTypeSchema.parse({
      id: 6,
      key: 'mammography',
      category: 'age_based',
      title: { fa: 'ماموگرافی' },
      subtitle: {},
      why: {},
      performed_by: 'imaging',
      icon: null,
      tone: null,
      interval_months: 24,
      interval_months_max: null,
      age_min: 40,
      age_max: 74,
      cycle_day_from: null,
      cycle_day_to: null,
      remind_lead_days: null,
      prep_steps: [],
      guide_steps: [],
      finding_options: [],
      hide_in_pregnancy: true,
      audiences: null,
      is_active: true,
      sort_order: 6,
      source_note: null,
      ...over,
    });

  it('locks the interval and ages of a registry-driven row', () => {
    expect(registryLocked(row({ registry_key: 'screening.uspstf' }))).toBe(true);
    expect(registryLocked(row({ registry_key: null }))).toBe(false);
    expect(registryLocked(row({}))).toBe(false);
    expect(registryLocked(null)).toBe(false);
  });

  it('keeps informational on every PUT (absent = false on the API)', () => {
    expect(rowToBody(row({ informational: true })).informational).toBe(true);
    expect(rowToBody(row({})).informational).toBe(false);
  });
});
