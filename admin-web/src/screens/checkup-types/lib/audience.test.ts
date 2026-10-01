import { describe, expect, it } from 'vitest';

import { checkupOptionsSchema, checkupTypeSchema } from '../api/checkup-types';
import { audiencesBody, audiencesError, filterByAudience, LIFE_MODES } from './audience';
import { rowToBody } from './payload';

const rows = [
  { id: 1, audiences: [] as string[] },
  { id: 2, audiences: ['menopause'] },
  { id: 3, audiences: ['cycle', 'ttc'] },
  { id: 4, audiences: ['menopause', 'teen'] },
];

describe('filterByAudience', () => {
  it('keeps every row without a filter', () => {
    expect(filterByAudience(rows, 'all').map((r) => r.id)).toEqual([1, 2, 3, 4]);
  });
  it('lists the rows that target a mode (menopause)', () => {
    expect(filterByAudience(rows, 'menopause').map((r) => r.id)).toEqual([2, 4]);
    expect(filterByAudience(rows, 'pregnancy')).toEqual([]);
  });
  it('lists shared rows under everyone', () => {
    expect(filterByAudience(rows, 'everyone').map((r) => r.id)).toEqual([1]);
  });
});

describe('audiencesBody', () => {
  it('sends the options order and null for none (= everyone)', () => {
    expect(audiencesBody(['teen', 'menopause'], LIFE_MODES)).toEqual(['menopause', 'teen']);
    expect(audiencesBody([], LIFE_MODES)).toBeNull();
  });
  it('keeps a stored mode the options no longer list', () => {
    expect(audiencesBody(['legacy', 'cycle'], ['cycle'])).toEqual(['cycle', 'legacy']);
  });
});

describe('audiencesError', () => {
  it('finds the field or item error', () => {
    expect(audiencesError({ audiences: ['too many'] })).toBe('too many');
    expect(audiencesError({ title: ['x'], 'audiences.1': ['bad mode'] })).toBe('bad mode');
    expect(audiencesError({ title: ['x'] })).toBeUndefined();
    expect(audiencesError(undefined)).toBeUndefined();
  });
});

const apiRow = {
  id: 7,
  key: 'meno_bone_density',
  category: 'age_based',
  title: { fa: 'تراکم استخوان', en: 'Bone density' },
  subtitle: {},
  why: {},
  performed_by: 'doctor',
  icon: null,
  tone: null,
  interval_months: 24,
  interval_months_max: null,
  age_min: 65,
  age_max: null,
  cycle_day_from: null,
  cycle_day_to: null,
  remind_lead_days: null,
  prep_steps: null,
  guide_steps: null,
  finding_options: null,
  hide_in_pregnancy: true,
  is_active: true,
  sort_order: 20,
  source_note: null,
};

describe('audiences in the checkup type schema and write body', () => {
  it('normalises null (everyone) to [] and keeps listed modes', () => {
    expect(checkupTypeSchema.parse({ ...apiRow, audiences: null }).audiences).toEqual([]);
    expect(checkupTypeSchema.parse(apiRow).audiences).toEqual([]);
    expect(checkupTypeSchema.parse({ ...apiRow, audiences: ['menopause'] }).audiences).toEqual(['menopause']);
  });
  it('rowToBody sends the audiences, null for everyone (the list toggle keeps them)', () => {
    expect(rowToBody(checkupTypeSchema.parse({ ...apiRow, audiences: ['menopause'] })).audiences).toEqual(['menopause']);
    expect(rowToBody(checkupTypeSchema.parse({ ...apiRow, audiences: null })).audiences).toBeNull();
  });
  it('options fall back to the life modes when the API omits them', () => {
    const base = {
      categories: [],
      performed_by: [],
      icons: [],
      tones: [],
      default_tone: 'neutral',
      default_remind_lead_days: 7,
      max_steps: 10,
      max_cycle_day: 45,
      max_interval_months: 120,
      next_sort_order: 1,
    };
    expect(checkupOptionsSchema.parse(base).audiences).toEqual([...LIFE_MODES]);
    expect(checkupOptionsSchema.parse({ ...base, audiences: ['menopause'] }).audiences).toEqual(['menopause']);
  });
});

describe('audience labels', () => {
  it('every life mode has fa and en labels', async () => {
    const en = (await import('../../../../messages/en.json')).default;
    const fa = (await import('../../../../messages/fa.json')).default;
    for (const messages of [en, fa]) {
      const labels = messages.checkupTypes.audienceOpt as Record<string, string>;
      for (const mode of LIFE_MODES) expect(labels[mode], mode).toBeTruthy();
    }
  });
});
