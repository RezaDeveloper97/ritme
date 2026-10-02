import { describe, expect, it } from 'vitest';

import type { LogCategory, LogDayValues, LogParam } from '@/entities/health-log';

import {
  acceptsNo,
  bleedingLogged,
  hasRow,
  isMenopausePreset,
  MENOPAUSE_GROUPS,
  nextBleeding,
  rowLevel,
  toggleTrigger,
  withRowLevel,
  type PresetRow,
} from './menopause-preset';

function param(p: Partial<LogParam> & Pick<LogParam, 'code' | 'type'>): LogParam {
  return {
    label: p.code,
    modes: [],
    detail: false,
    alert: false,
    options: [],
    levels: [],
    score: null,
    range: null,
    scale: null,
    unit: null,
    maxLength: null,
    dynamic: false,
    source: null,
    ...p,
  };
}

const opt = (value: string) => ({ value, label: value, modes: null, legacyOnly: false });
const lv = (value: string) => ({ value, label: value });

const categories: LogCategory[] = [
  {
    code: 'symptoms',
    label: 'symptoms',
    group: { value: 'body', label: 'body' },
    modes: [],
    conditions: {},
    params: [
      param({
        code: 'general',
        type: 'items',
        options: [opt('hot_flashes'), opt('fatigue')],
        levels: ['yes', 'no', 'mild', 'moderate', 'severe'].map(lv),
      }),
    ],
  },
  {
    code: 'pain',
    label: 'pain',
    group: { value: 'body', label: 'body' },
    modes: [],
    conditions: {},
    params: [param({ code: 'location', type: 'items', options: [opt('joints')], levels: ['mild', 'moderate', 'severe'].map(lv) })],
  },
];

const hot: PresetRow = { category: 'symptoms', param: 'general', item: 'hot_flashes' };
const joints: PresetRow = { category: 'pain', param: 'location', item: 'joints' };

describe('menopause preset', () => {
  it('mirrors the board: 5 groups, 13 rows', () => {
    expect(MENOPAUSE_GROUPS.map((g) => g.code)).toEqual(['vasomotor', 'sleep', 'mind', 'body', 'urogenital']);
    expect(MENOPAUSE_GROUPS.flatMap((g) => g.rows)).toHaveLength(13);
  });

  it('applies only to the menopause taxonomy', () => {
    expect(isMenopausePreset('menopause')).toBe(true);
    expect(isMenopausePreset('cycle')).toBe(false);
    expect(isMenopausePreset(null)).toBe(false);
  });

  it('draws only rows the taxonomy offers', () => {
    expect(hasRow(categories, hot)).toBe(true);
    expect(hasRow(categories, { category: 'symptoms', param: 'general', item: 'insomnia' })).toBe(false);
    expect(acceptsNo(categories, hot)).toBe(true);
    expect(acceptsNo(categories, joints)).toBe(false);
  });

  it('sets, re-taps to clear and keeps the other items', () => {
    const values: LogDayValues = { symptoms: { general: { fatigue: { level: 'mild', score: null } } } };
    const set = withRowLevel(values, hot, 'moderate', true);
    expect(set).toEqual({ fatigue: { level: 'mild', score: null }, hot_flashes: { level: 'moderate', score: null } });
    const next: LogDayValues = { symptoms: { general: set } };
    expect(rowLevel(next, hot)).toBe('moderate');
    expect(withRowLevel(next, hot, 'moderate', true)).toEqual({ fatigue: { level: 'mild', score: null } });
    expect(withRowLevel(next, hot, 'no', true).hot_flashes).toEqual({ level: 'no', score: null });
  });

  it('«ندارم» on a pain location removes the entry', () => {
    const values: LogDayValues = { pain: { location: { joints: { level: 'severe', score: 8 } } } };
    expect(withRowLevel(values, joints, 'no', false)).toEqual({});
    expect(rowLevel({}, joints)).toBeNull();
  });

  it('bleeding choice toggles and flags spotting/bleeding', () => {
    expect(nextBleeding({}, 'spotting')).toBe('spotting');
    expect(nextBleeding({ bleeding: { presence: 'spotting' } }, 'spotting')).toBeNull();
    expect(bleedingLogged({ bleeding: { presence: 'none' } })).toBe(false);
    expect(bleedingLogged({ bleeding: { presence: 'bleeding' } })).toBe(true);
  });

  it('toggles triggers', () => {
    expect(toggleTrigger({ menopause: { triggers: ['stress'] } }, 'caffeine')).toEqual(['stress', 'caffeine']);
    expect(toggleTrigger({ menopause: { triggers: ['stress'] } }, 'stress')).toEqual([]);
  });
});
