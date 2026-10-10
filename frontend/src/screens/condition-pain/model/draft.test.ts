import { describe, expect, it } from 'vitest';

import type { PainDay } from '@/entities/condition';
import type { LogTaxonomy } from '@/entities/health-log';

import { draftFrom, draftProblem, painLocations, toggleCode, toInput } from './draft';

const empty: PainDay = {
  date: '2026-10-08',
  score: null,
  locations: [],
  relief: [],
  types: [],
  associated: [],
  missedActivity: null,
  analgesic: null,
  analgesicTime: null,
  analgesicEffect: null,
};

describe('pain draft', () => {
  it('round-trips a saved day into the PUT body', () => {
    const d = draftFrom({
      ...empty,
      score: 7,
      locations: ['pelvis'],
      types: ['cramping'],
      missedActivity: true,
      analgesic: 'Ibuprofen',
      analgesicTime: '08:00',
      analgesicEffect: 'a_little',
    });
    expect(toInput(d)).toEqual({
      score: 7,
      locations: ['pelvis'],
      types: ['cramping'],
      associated: [],
      missed_activity: true,
      analgesic: 'Ibuprofen',
      analgesic_time: '08:00',
      analgesic_effect: 'a_little',
    });
  });

  it('sends no locations for «بدون درد» and blanks as null', () => {
    const d = { ...draftFrom(empty), score: 0, locations: ['back'], analgesic: '  ', analgesicTime: '8' };
    expect(toInput(d)).toMatchObject({ score: 0, locations: null, analgesic: null, analgesic_time: null, missed_activity: false });
  });

  it('checks the score / location pairing', () => {
    const d = draftFrom(empty);
    expect(draftProblem(d)).toBeNull();
    expect(draftProblem({ ...d, score: 5 })).toBe('locations');
    expect(draftProblem({ ...d, locations: ['back'] })).toBe('score');
    expect(draftProblem({ ...d, score: 0 })).toBeNull();
  });

  it('toggles codes', () => {
    expect(toggleCode(['a'], 'b')).toEqual(['a', 'b']);
    expect(toggleCode(['a', 'b'], 'a')).toEqual(['b']);
  });

  it('lists the mode’s locations plus saved ones', () => {
    const taxonomy = {
      mode: 'cycle',
      categories: [
        {
          code: 'pain',
          params: [
            {
              code: 'location',
              options: [
                { value: 'pelvis', label: 'Pelvis', modes: null, legacyOnly: false },
                { value: 'stitches', label: 'Stitches', modes: ['postpartum'], legacyOnly: false },
                { value: 'old', label: 'Old', modes: null, legacyOnly: true },
              ],
            },
          ],
        },
      ],
    } as unknown as LogTaxonomy;
    expect(painLocations(taxonomy, ['old', 'mystery'])).toEqual([
      { code: 'pelvis', label: 'Pelvis' },
      { code: 'old', label: 'Old' },
      { code: 'mystery', label: 'mystery' },
    ]);
    expect(painLocations(undefined, [])).toEqual([]);
  });
});
